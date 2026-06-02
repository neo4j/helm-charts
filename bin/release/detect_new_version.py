#!/usr/bin/env python3
"""Detect whether a new Neo4j docker image warrants an auto-release.

Compares the ``appVersion`` pinned in ``neo4j/Chart.yaml`` against GA calver
tags published to Docker Hub for ``library/neo4j`` and emits a decision as
GitHub Actions outputs on stdout (key=value lines, ready for ``>> $GITHUB_OUTPUT``).

A release is triggered only if **all** of these hold:

* a strictly-greater GA calver tag exists,
* the explicit ``<tag>-trixie``, ``<tag>-enterprise-trixie``,
  ``<tag>-ubi10``, and ``<tag>-enterprise-ubi10`` variants exist
  (they are ``FROM``-referenced during backup-image builds in tests.yml /
  package-and-release.yml),
* no GitHub release already exists for the derived helm-chart version.

Required env: ``GITHUB_TOKEN`` (used for idempotency check against
``/releases``) and optionally ``DOCKERHUB_USERNAME`` + ``DOCKERHUB_TOKEN`` to
lift Docker Hub's anonymous rate limit. ``GITHUB_REPOSITORY`` defaults to
``neo4j/helm-charts``.
"""

from __future__ import annotations

import json
import os
import re
import sys
from pathlib import Path
from typing import Iterable
from urllib import request, error, parse

CHART_PATH = Path("neo4j/Chart.yaml")
GA_TAG_RE = re.compile(r"^(\d{4})\.(\d{2})\.(\d+)$")
APPVERSION_RE = re.compile(r"^appVersion:\s*(\S+)\s*$", re.MULTILINE)
DOCKER_HUB_IMAGE = "library/neo4j"
REQUIRED_RELEASE_SUFFIXES = (
    "-trixie",
    "-enterprise-trixie",
    "-ubi10",
    "-enterprise-ubi10",
)


def read_current_app_version(path: Path = CHART_PATH) -> str:
    text = path.read_text()
    m = APPVERSION_RE.search(text)
    if not m:
        raise RuntimeError(f"Could not find appVersion in {path}")
    return m.group(1).strip('"').strip("'")


def parse_ga_tag(tag: str) -> tuple[int, int, int] | None:
    m = GA_TAG_RE.match(tag)
    if not m:
        return None
    return (int(m.group(1)), int(m.group(2)), int(m.group(3)))


def dockerhub_auth_token() -> str | None:
    """Obtain a bearer token for Docker Hub if credentials are present.

    Anonymous requests work but are rate-limited. When DOCKERHUB_USERNAME /
    DOCKERHUB_TOKEN are set we trade them for a short-lived JWT.
    """
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


def list_all_tags(image: str, token: str | None) -> list[str]:
    headers: dict[str, str] = {"User-Agent": "helm-charts-auto-release"}
    if token:
        headers["Authorization"] = f"Bearer {token}"
    url = (
        f"https://hub.docker.com/v2/repositories/{image}/tags/"
        f"?page_size=100&ordering=last_updated"
    )
    tags: list[str] = []
    pages = 0
    while url and pages < 10:
        pages += 1
        req = request.Request(url, headers=headers)
        with request.urlopen(req) as resp:
            data = json.loads(resp.read().decode("utf-8"))
        for r in data.get("results", []):
            name = r.get("name")
            if name:
                tags.append(name)
        url = data.get("next")
    return tags


def tag_exists(image: str, tag: str, token: str | None) -> bool:
    headers: dict[str, str] = {"User-Agent": "helm-charts-auto-release"}
    if token:
        headers["Authorization"] = f"Bearer {token}"
    url = f"https://hub.docker.com/v2/repositories/{image}/tags/{parse.quote(tag)}/"
    try:
        with request.urlopen(request.Request(url, headers=headers)) as resp:
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
            "User-Agent": "helm-charts-auto-release",
        },
    )
    try:
        with request.urlopen(req) as resp:
            return resp.status == 200
    except error.HTTPError as e:
        if e.code == 404:
            return False
        raise


def derive_helm_chart_version(docker_tag: str) -> str:
    # 2026.03.1 -> 2026.3.1 ; 2027.11.0 -> 2027.11.0 (no leading zero).
    return re.sub(r"^(\d{4})\.0(\d)\.", r"\1.\2.", docker_tag)


def required_release_tags(docker_tag: str) -> list[str]:
    return [f"{docker_tag}{suffix}" for suffix in REQUIRED_RELEASE_SUFFIXES]


def pick_latest_ga(tags: Iterable[str]) -> str | None:
    best: tuple[tuple[int, int, int], str] | None = None
    for t in tags:
        key = parse_ga_tag(t)
        if key is None:
            continue
        if best is None or key > best[0]:
            best = (key, t)
    return best[1] if best else None


def write_outputs(fields: dict[str, str]) -> None:
    for k, v in fields.items():
        sys.stdout.write(f"{k}={v}\n")


def main(argv: list[str]) -> int:
    gh_token = os.environ.get("GITHUB_TOKEN")
    if not gh_token:
        print("GITHUB_TOKEN is required", file=sys.stderr)
        return 2
    repo = os.environ.get("GITHUB_REPOSITORY", "neo4j/helm-charts")
    owner, name = repo.split("/", 1)

    current = read_current_app_version()
    current_key = parse_ga_tag(current)
    print(f"Current appVersion: {current} -> {current_key}", file=sys.stderr)

    dh_token = dockerhub_auth_token()
    tags = list_all_tags(DOCKER_HUB_IMAGE, dh_token)
    print(f"Fetched {len(tags)} tags from Docker Hub", file=sys.stderr)

    latest = pick_latest_ga(tags)
    if not latest:
        print("No GA calver tags found", file=sys.stderr)
        write_outputs({"should_release": "false"})
        return 0
    print(f"Latest GA tag: {latest}", file=sys.stderr)

    latest_key = parse_ga_tag(latest)
    if current_key is not None and latest_key is not None and latest_key <= current_key:
        print("No newer GA tag than current appVersion", file=sys.stderr)
        write_outputs({"should_release": "false"})
        return 0

    helm_chart_version = derive_helm_chart_version(latest)
    missing_variants = [
        variant
        for variant in required_release_tags(latest)
        if not tag_exists(DOCKER_HUB_IMAGE, variant, dh_token)
    ]
    if missing_variants:
        missing = ", ".join(missing_variants)
        print(
            f"::error::Required Neo4j Docker image variant(s) missing for {latest}: {missing}",
            file=sys.stderr,
        )
        write_outputs(
            {
                "should_release": "false",
                "docker_image_version": latest,
                "helm_chart_version": helm_chart_version,
                "failure_reason": "missing_required_image_variants",
                "missing_variants": missing,
            }
        )
        return 1

    if github_release_exists(owner, name, helm_chart_version, gh_token):
        print(
            f"Release already exists for {helm_chart_version} — skipping",
            file=sys.stderr,
        )
        write_outputs({"should_release": "false"})
        return 0

    print(
        f"Triggering release: docker={latest} chart={helm_chart_version}",
        file=sys.stderr,
    )
    write_outputs(
        {
            "should_release": "true",
            "docker_image_version": latest,
            "helm_chart_version": helm_chart_version,
        }
    )
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
