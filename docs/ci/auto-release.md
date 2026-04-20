# Auto-release on new Neo4j docker versions

The scheduled workflow at `.github/workflows/auto-release.yml` polls Docker
Hub four times a day (00:00, 06:00, 12:00, 18:00 UTC) and, when a new GA Neo4j
image is available, runs the full test suite against it and releases the
matching helm-chart version.

## How it decides to release

`bin/release/detect_new_version.py`:

1. Reads the current `appVersion` from `neo4j/Chart.yaml`.
2. Queries Docker Hub for `library/neo4j` tags and keeps only GA calver
   tags matching `^\d{4}\.\d{2}\.\d+$` (rejects `-enterprise`, `-community`,
   `-ubi9`, `nightly`, etc.).
3. Compares the highest GA tag to the pinned `appVersion` by
   `(year, month, patch)`.
4. Verifies that both `<tag>-enterprise` and `<tag>-enterprise-ubi9` variants
   exist on Docker Hub — those are `FROM`-referenced when building backup
   images in `tests.yml` / `package-and-release.yml`.
5. Skips if a GitHub release already exists for the derived helm-chart
   version.
6. Derives the helm chart version by stripping the month's leading zero
   (`2026.03.1` → `2026.3.1`). The docker image version keeps zero-padding.

If all checks pass, the detector writes
`should_release=true`/`docker_image_version=...`/`helm_chart_version=...` to
`$GITHUB_OUTPUT`.

## Changelog aggregation

At release time the packaging job runs `bin/changelog/generate.py`, which:

* finds PRs merged since the last git tag,
* keeps only PRs carrying the `changelog` GitHub label,
* extracts every `cl: <text>` line from each PR body (regex
  `^cl:\s*(.+)$`, case-insensitive).

The output is written to `tmp/release-notes.md` and used both as the
GitHub release body (via `softprops/action-gh-release`'s `body_path:`) and
prepended to `CHANGELOG.md` by `bin/changelog/update_file.py`. The
`CHANGELOG.md` change rides into the "Update chart versions…" commit that
`bin/gcloud/index_yaml_update` creates.

PRs without the `changelog` label are silently skipped, so chore/CI-only
PRs don't clutter release notes.

## Scope

* **Train:** CalVer on the `dev` branch only (e.g., `2026.3.2`). The `5.26`
  and `4.4` trains remain manual.
* **Version bumps:** patch, minor, and major (covered by the single
  "highest GA tag wins" comparison).
* **Failure handling:** abort the release and post to the failures Slack
  channel. No auto-opened issue.

## One-time setup

The workflow assumes the following repo configuration. This only needs to be
done once.

1. **Create the `changelog` label** so PR authors can opt in:

   ```bash
   gh label create changelog \
     --color 0366d6 \
     --description "Include this PR in the next helm-charts release notes"
   ```

2. **Add secrets and repo variables** (repo Settings → Secrets and variables
   → Actions):

   Secrets:

   | Name | Purpose |
   | --- | --- |
   | `RELEASE_AUTOMATION_PAT` | Fine-grained PAT owned by a user in `CI_ALLOWED_ACTORS` (currently `bfeshti` / `riggi-alekaj`). Needs `actions: read/write`, `contents: read` on this repo. The scheduled workflow uses it to `gh workflow run tests.yml`, and the dispatched run's `github.actor` becomes the PAT owner — which is why it must be allow-listed. |
   | `SLACK_BOT_TOKEN` | Bot user OAuth token (`xoxb-…`) for the Slack app that posts notifications. The app must have the `chat:write` scope and be invited to both channels below (`/invite @<bot>` in each channel). |

   Repo variables:

   | Name | Purpose |
   | --- | --- |
   | `SLACK_CHANNEL_FAILURES` | Slack channel ID (e.g. `C0123456789`) for failure alerts. Grab from Slack → channel → "View channel details" → bottom of "About" tab. |
   | `SLACK_CHANNEL_RELEASES` | Slack channel ID for successful-release announcements. |

   `DOCKERHUB_USERNAME` / `DOCKERHUB_TOKEN` already exist (used by
   `package-and-release.yml`); the detector reuses them to lift Docker Hub's
   anonymous rate limit.

3. **Verify `CI_ALLOWED_ACTORS`** repo variable (optional — defaults to
   `["bfeshti","riggi-alekaj"]` via the authz-gate composite action's
   fallback) includes the PAT owner.

## Manual operations

| Task | Command |
| --- | --- |
| Trigger an out-of-schedule check | `gh workflow run auto-release.yml` |
| Re-release a version after fixing a broken one | Delete the GitHub release + tag for that version, then `gh workflow run auto-release.yml` |
| Disable auto-release temporarily | Comment out the `schedule:` block in `auto-release.yml`; `workflow_dispatch:` remains for manual triggers |
| Inspect what the detector would do | `GITHUB_TOKEN=<pat> DOCKERHUB_USERNAME=<u> DOCKERHUB_TOKEN=<t> python3 bin/release/detect_new_version.py` (from the repo root) |

## Failure modes

| Signal | Cause | Action |
| --- | --- | --- |
| `notify_failure` Slack ping with "Dispatched run: …" | The `tests.yml` run that auto-release triggered failed. | Open the linked run, investigate, fix, re-trigger via `gh workflow run auto-release.yml`. |
| `notify_failure` Slack ping with empty run URL | The detector or dispatch step failed (before a run was located). | Open the orchestrator run linked in the message. |
| Orchestrator succeeds but no release appears | Detector decided `should_release=false`. Orchestrator run logs show the reason (no newer tag, variants missing, release already exists, etc.). | Expected quiet behavior. |
