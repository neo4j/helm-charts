# Auto-release for Neo4j 5.26 patch versions

The scheduled workflow at `.github/workflows/auto-release.yml` polls Docker
Hub four times a day (00:00, 06:00, 12:00, 18:00 UTC). When a new `5.26.x`
Neo4j image is available, it dispatches the full `5.26` test/release pipeline.

## How it decides to release

`bin/release/detect_new_version.py`:

1. Reads the current `appVersion` from `neo4j/Chart.yaml`.
2. Queries Docker Hub for `library/neo4j` tags and keeps only tags matching
   `^5\.26\.\d+$`.
3. Compares the highest discovered patch version to the pinned `appVersion`.
4. Verifies that both `<tag>-enterprise` and `<tag>-enterprise-ubi10` variants
   exist on Docker Hub. The backup image builds use these as source images.
5. Skips if a GitHub release already exists for the same `5.26.x` tag.
6. Emits the same value for `helm_chart_version` and `docker_image_version`,
   for example `5.26.26`.

If all checks pass, the detector writes
`should_release=true`/`docker_image_version=...`/`helm_chart_version=...` to
`$GITHUB_OUTPUT`.

## Changelog aggregation

At release time the packaging job runs `bin/changelog/generate.py`, which:

* finds PRs merged since the last git tag,
* keeps only PRs carrying the `changelog` GitHub label,
* extracts every `cl: <text>` line from each PR body.

The output is written to `tmp/release-notes.md` and used both as the GitHub
release body and as a new section prepended to `CHANGELOG.md`.

PRs without the `changelog` label are skipped, so maintenance-only changes do
not clutter release notes.

## Scope

* **Train:** Neo4j `5.26.x` on the `5.26` branch only.
* **Release type:** patch releases only.
* **Images:** Debian release images use `<tag>-enterprise`; RedHat release
  images use `<tag>-enterprise-ubi10`.
* **Failure handling:** abort the release and post to the failures Slack
  channel. No issue is opened automatically.

## One-time setup

The workflow assumes the following repo configuration. This only needs to be
done once.

1. **Create the `changelog` label** so PR authors can opt in:

   ```bash
   gh label create changelog \
     --color 0366d6 \
     --description "Include this PR in the next helm-charts release notes"
   ```

2. **Add secrets and repo variables** under repo Settings -> Secrets and
   variables -> Actions.

   Secrets:

   | Name | Purpose |
   | --- | --- |
   | `GH_TOKEN` | Token used by the orchestrator to dispatch `tests.yml` and by release jobs to update contents/releases. The token owner must be allowed by `CI_ALLOWED_ACTORS`. |
   | `SLACK_BOT_TOKEN` | Bot user OAuth token for Slack notifications. The app must have `chat:write` and be invited to both configured channels. |
   | `DOCKERHUB_USERNAME` / `DOCKERHUB_TOKEN` | Optional Docker Hub credentials used by the detector and backup builds to avoid anonymous rate limits. |

   Repo variables:

   | Name | Purpose |
   | --- | --- |
   | `CI_ALLOWED_ACTORS` | JSON array of GitHub usernames allowed to run release workflows. Defaults to `["bfeshti","riggi-alekaj"]` inside the auth gate if unset. |
   | `SLACK_CHANNEL_FAILURES` | Slack channel ID for failed release alerts. |
   | `SLACK_CHANNEL_RELEASES` | Slack channel ID for successful release announcements. |

## Manual operations

| Task | Command |
| --- | --- |
| Trigger an out-of-schedule check | `gh workflow run auto-release.yml --ref 5.26` |
| Re-release a version after fixing a broken one | Delete the GitHub release and tag for that version, then run `gh workflow run auto-release.yml --ref 5.26` |
| Disable auto-release temporarily | Comment out the `schedule:` block in `auto-release.yml`; `workflow_dispatch:` remains available |
| Inspect what the detector would do | `GITHUB_TOKEN=<pat> DOCKERHUB_USERNAME=<u> DOCKERHUB_TOKEN=<t> python3 bin/release/detect_new_version.py` |

## Failure modes

| Signal | Cause | Action |
| --- | --- | --- |
| Failure Slack ping with "Dispatched run: ..." | The dispatched `tests.yml` run failed. | Open the linked run, fix the failing test or release step, then re-trigger `auto-release.yml`. |
| Failure Slack ping with an empty run URL | Detection or dispatch failed before a run was located. | Open the orchestrator run linked in the message. |
| Orchestrator succeeds but no release appears | Detector emitted `should_release=false`. | Expected when no newer `5.26.x` image exists, a variant is missing, or the release already exists. |
