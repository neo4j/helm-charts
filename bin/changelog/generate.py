#!/usr/bin/env python3
"""Aggregate merged PRs since the last git tag into release-notes markdown.

Only PRs labeled ``changelog`` are included, and only ``cl:`` lines from their
bodies are extracted. The output always begins with a link to the Neo4j
database release notes.

Usage:
    python3 bin/changelog/generate.py \\
        --chart-version 5.26.26 \\
        --docker-version 5.26.26 \\
        --out tmp/release-notes.md

Required env: ``GITHUB_TOKEN``. Optional env: ``GITHUB_REPOSITORY`` (defaults
to ``neo4j/helm-charts``).
"""

from __future__ import annotations

import argparse
import json
import os
import re
import subprocess
import sys
from pathlib import Path
from typing import Iterable
from urllib import request, error

CL_RE = re.compile(r"^cl:\s*(.+)$", re.IGNORECASE | re.MULTILINE)
NEO4J_CALVER_RE = re.compile(r"^(\d{4})\.(\d{2})\.\d+")
NEO4J_MAJOR_RE = re.compile(r"^(\d+)\.\d+\.\d+")


def sh(cmd: list[str]) -> str:
    return subprocess.check_output(cmd, text=True).strip()


def previous_tag() -> str | None:
    try:
        return sh(["git", "describe", "--tags", "--abbrev=0"])
    except subprocess.CalledProcessError:
        return None


def commits_since(ref: str | None) -> list[str]:
    rng = f"{ref}..HEAD" if ref else "HEAD"
    out = sh(["git", "log", "--format=%H", rng])
    return [line for line in out.splitlines() if line]


def gh_get(path: str, token: str) -> object:
    req = request.Request(
        f"https://api.github.com{path}",
        headers={
            "Authorization": f"Bearer {token}",
            "Accept": "application/vnd.github+json",
            "X-GitHub-Api-Version": "2022-11-28",
            "User-Agent": "helm-charts-changelog-aggregator",
        },
    )
    try:
        with request.urlopen(req) as resp:
            return json.loads(resp.read().decode("utf-8"))
    except error.HTTPError as e:
        raise RuntimeError(
            f"GitHub API {e.code} for {path}: {e.read().decode('utf-8', 'replace')}"
        ) from e


def prs_for_commit(owner: str, repo: str, sha: str, token: str) -> list[dict]:
    return gh_get(
        f"/repos/{owner}/{repo}/commits/{sha}/pulls?per_page=100", token
    )


def extract_cl(body: str | None) -> list[str]:
    if not body:
        return []
    return [m.group(1).strip() for m in CL_RE.finditer(body)]


def neo4j_release_notes_url(docker_version: str) -> str | None:
    if m := NEO4J_CALVER_RE.match(docker_version):
        return f"https://neo4j.com/release-notes/database/neo4j-{m.group(1)}-{m.group(2)}/"
    if m := NEO4J_MAJOR_RE.match(docker_version):
        return f"https://neo4j.com/release-notes/database/neo4j-{m.group(1)}/"
    return None


def main(argv: Iterable[str]) -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("--chart-version", required=True)
    ap.add_argument("--docker-version", required=True)
    ap.add_argument("--out", required=True)
    args = ap.parse_args(list(argv))

    token = os.environ.get("GITHUB_TOKEN")
    if not token:
        print("GITHUB_TOKEN is required", file=sys.stderr)
        return 2

    repo = os.environ.get("GITHUB_REPOSITORY", "neo4j/helm-charts")
    owner, name = repo.split("/", 1)

    prev = previous_tag()
    print(f"Previous tag: {prev or '(none)'}", file=sys.stderr)

    commits = commits_since(prev)
    print(f"Commits in range: {len(commits)}", file=sys.stderr)

    seen: dict[int, dict] = {}
    for sha in commits:
        try:
            for pr in prs_for_commit(owner, name, sha, token):
                if not pr.get("merged_at"):
                    continue
                seen.setdefault(pr["number"], pr)
        except Exception as e:
            print(f"  skip {sha}: {e}", file=sys.stderr)
    print(f"Unique merged PRs discovered: {len(seen)}", file=sys.stderr)

    entries: list[tuple[str, int, str]] = []
    for pr in seen.values():
        labels = {label["name"] for label in pr.get("labels") or []}
        if "changelog" not in labels:
            continue
        for text in extract_cl(pr.get("body")):
            entries.append((text, pr["number"], pr["html_url"]))
    print(f"Changelog-labeled entries: {len(entries)}", file=sys.stderr)

    lines: list[str] = []
    url = neo4j_release_notes_url(args.docker_version)
    if url:
        lines.append(f"- See [Neo4j release notes]({url}) for upstream DB changes")

    if entries:
        lines.extend(["", "### Helm-charts changes", ""])
        for text, num, html_url in entries:
            lines.append(f"- {text} ([#{num}]({html_url}))")
    else:
        lines.append("- No user-facing helm-charts changes (maintenance release).")

    output = "\n".join(lines) + "\n"
    out_path = Path(args.out)
    out_path.parent.mkdir(parents=True, exist_ok=True)
    out_path.write_text(output)
    print(f"Wrote {out_path} ({len(output)} bytes)", file=sys.stderr)
    sys.stdout.write(output)
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
