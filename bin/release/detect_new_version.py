#!/usr/bin/env python3
"""Detect whether a new Neo4j 5.26 patch image warrants an auto-release.

This is intentionally scoped to the 5.26 train. It compares the ``appVersion``
pinned in ``neo4j/Chart.yaml`` against Docker Hub tags matching
``5.26.<patch>`` for ``library/neo4j`` and emits GitHub Actions outputs on
stdout (``key=value`` lines, ready for ``>> $GITHUB_OUTPUT``).

A release is triggered only if all of these hold:

* a strictly-greater ``5.26.<patch>`` tag exists,
* the ``<tag>-enterprise`` and ``<tag>-enterprise-ubi10`` variants exist
  (they are ``FROM``-referenced during backup-image builds),
* no GitHub release already exists for the same ``5.26.<patch>`` tag.

Required env: ``GITHUB_TOKEN``. Optional env:
``DOCKERHUB_USERNAME``/``DOCKERHUB_TOKEN`` to avoid Docker Hub anonymous rate
limits. ``GITHUB_REPOSITORY`` defaults to ``neo4j/helm-charts``.
"""

from __future__ import annotations

import json
import os
import re
import sys
from pathlib import Path
from typing import Iterable
from urllib import error, parse, request

CHART_PATH = Path("neo4j/Chart.yaml")
PATCH_TAG_RE = re.compile(r"^5\.26\.(\d+)$")
APPVERSION_RE = re.compile(r"^appVersion:\s*(\S+)\s*$", re.MULTILINE)
DOCKER_HUB_IMAGE = "library/neo4j"


def read_current_app_version(path: Path = CHART_PATH) -> str:
    text = path.read_text()
    m = APPVERSION_RE.search(text)
    if not m:
        raise RuntimeError(f"Could not find appVersion in {path}")
    return m.group(1).strip('"').strip("'")


def parse_526_patch(tag: str) -> int | None:
    m = PATCH_TAG_RE.match(tag)
    if not m:
        return None
    return int(m.group(1))


def dockerhub_auth_token() -> str | None:
    user = os.environ.get("DOCKERHUB_USERNAME")
    pw = os.environ.get("DOCKERHUB_TOKEN")
    if not user or not pw:
        return None
    req = request.Request(
        "https://hub.docker.com/v2/users/login",
        data=json.dumps({"username": user, "password": pw}).encode("utf-8"),
        headers={"Content-Type": "application/json"},
        method="POST",
    )
    with request.urlopen(req) as resp:
        return json.loads(resp.read().decode("utf-8")).get("token")


def dockerhub_headers(token: str | None) -> dict[str, str]:
    headers = {"User-Agent": "helm-charts-5.26-auto-release"}
    if token:
        headers["Authorization"] = f"Bearer {token}"
    return headers


def list_all_tags(image: str, token: str | None) -> list[str]:
    url = (
        f"https://hub.docker.com/v2/repositories/{image}/tags/"
        f"?page_size=100&ordering=last_updated"
    )
    tags: list[str] = []
    pages = 0
    while url and pages < 50:
        pages += 1
        req = request.Request(url, headers=dockerhub_headers(token))
        with request.urlopen(req) as resp:
            data = json.loads(resp.read().decode("utf-8"))
        tags.extend(r["name"] for r in data.get("results", []) if r.get("name"))
        url = data.get("next")
    return tags


def tag_exists(image: str, tag: str, token: str | None) -> bool:
    url = f"https://hub.docker.com/v2/repositories/{image}/tags/{parse.quote(tag)}/"
    try:
        with request.urlopen(
            request.Request(url, headers=dockerhub_headers(token))
        ) as resp:
            return resp.status == 200
    except error.HTTPError as e:
        if e.code == 404:
            return False
        raise


def github_release_exists(owner: str, repo: str, tag: str, token: str) -> bool:
    req = request.Request(
        f"https://api.github.com/repos/{owner}/{repo}/releases/tags/{parse.quote(tag)}",
        headers={
            "Authorization": f"Bearer {token}",
            "Accept": "application/vnd.github+json",
            "X-GitHub-Api-Version": "2022-11-28",
            "User-Agent": "helm-charts-5.26-auto-release",
        },
    )
    try:
        with request.urlopen(req) as resp:
            return resp.status == 200
    except error.HTTPError as e:
        if e.code == 404:
            return False
        raise


def pick_latest_526(tags: Iterable[str]) -> str | None:
    best: tuple[int, str] | None = None
    for tag in tags:
        patch = parse_526_patch(tag)
        if patch is None:
            continue
        if best is None or patch > best[0]:
            best = (patch, tag)
    return best[1] if best else None


def write_outputs(fields: dict[str, str]) -> None:
    for key, value in fields.items():
        sys.stdout.write(f"{key}={value}\n")


def main(argv: list[str]) -> int:
    gh_token = os.environ.get("GITHUB_TOKEN")
    if not gh_token:
        print("GITHUB_TOKEN is required", file=sys.stderr)
        return 2
    repo = os.environ.get("GITHUB_REPOSITORY", "neo4j/helm-charts")
    owner, name = repo.split("/", 1)

    current = read_current_app_version()
    current_patch = parse_526_patch(current)
    if current_patch is None:
        print(
            f"Current appVersion {current!r} is not a 5.26 patch version; refusing auto-release",
            file=sys.stderr,
        )
        write_outputs({"should_release": "false"})
        return 0
    print(f"Current 5.26 appVersion: {current}", file=sys.stderr)

    dh_token = dockerhub_auth_token()
    tags = list_all_tags(DOCKER_HUB_IMAGE, dh_token)
    print(f"Fetched {len(tags)} tags from Docker Hub", file=sys.stderr)

    latest = pick_latest_526(tags)
    if latest is None:
        print("No 5.26 patch tags found", file=sys.stderr)
        write_outputs({"should_release": "false"})
        return 0
    print(f"Latest 5.26 patch tag: {latest}", file=sys.stderr)

    latest_patch = parse_526_patch(latest)
    if latest_patch is None or latest_patch <= current_patch:
        print("No newer 5.26 patch tag than current appVersion", file=sys.stderr)
        write_outputs({"should_release": "false"})
        return 0

    for variant in (f"{latest}-enterprise", f"{latest}-enterprise-ubi10"):
        if not tag_exists(DOCKER_HUB_IMAGE, variant, dh_token):
            print(f"Variant missing: {variant}; skipping", file=sys.stderr)
            write_outputs({"should_release": "false"})
            return 0

    if github_release_exists(owner, name, latest, gh_token):
        print(f"Release already exists for {latest}; skipping", file=sys.stderr)
        write_outputs({"should_release": "false"})
        return 0

    print(f"Triggering 5.26 release: docker={latest} chart={latest}", file=sys.stderr)
    write_outputs(
        {
            "should_release": "true",
            "docker_image_version": latest,
            "helm_chart_version": latest,
        }
    )
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
