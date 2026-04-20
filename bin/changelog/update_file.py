#!/usr/bin/env python3
"""Prepend a new release section into ``CHANGELOG.md``.

Creates the file with a top-level ``# Changelog`` heading if it does not yet
exist. The new section is inserted right below the top-level heading so the
file stays reverse-chronological.

Usage:
    python3 bin/changelog/update_file.py \\
        --notes tmp/release-notes.md \\
        --changelog CHANGELOG.md \\
        --chart-version 2026.3.2 \\
        --docker-version 2026.03.2
"""

from __future__ import annotations

import argparse
import sys
from datetime import datetime, timezone
from pathlib import Path


def today_utc() -> str:
    return datetime.now(timezone.utc).strftime("%Y-%m-%d")


def build_section(chart_version: str, docker_version: str, notes: str) -> str:
    header = (
        f"## [{chart_version}] - {today_utc()}\n"
        f"\n"
        f"*Neo4j docker image: `{docker_version}`*\n"
    )
    return f"{header}\n{notes.rstrip()}\n"


def main(argv: list[str]) -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("--notes", required=True)
    ap.add_argument("--changelog", default="CHANGELOG.md")
    ap.add_argument("--chart-version", required=True)
    ap.add_argument("--docker-version", required=True)
    args = ap.parse_args(argv)

    notes = Path(args.notes).read_text()
    section = build_section(args.chart_version, args.docker_version, notes)

    target = Path(args.changelog)
    existing = target.read_text() if target.exists() else ""

    if not existing.strip():
        output = f"# Changelog\n\n{section}"
    elif existing.startswith("# Changelog"):
        nl = existing.index("\n")
        head = existing[: nl + 1]
        rest = existing[nl + 1 :].lstrip()
        output = f"{head}\n{section}\n{rest}"
    else:
        output = f"# Changelog\n\n{section}\n{existing}"

    target.write_text(output)
    print(f"Updated {target} with section for {args.chart_version}", file=sys.stderr)
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
