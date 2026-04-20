# ADR 0001: Runner topology for the CI test pipeline

*Status: Accepted. Author: CI overhaul, Phase 4.*

## Context

The helm-charts repository has two distinct CI needs that were historically served by a single 75-minute GKE-backed pipeline:

1. **Full integration coverage.** Real LoadBalancer provisioning, GKE workload identity, disk provisioning, and ingress behavior — things that require a real cloud Kubernetes distribution and genuine network infrastructure.
2. **PR-level feedback.** Did this change break helm templating? Did a chart start producing invalid Kubernetes manifests? Did a values-file default regress?

Running (2) on the same pipeline as (1) means every trivial chart change has to wait for a ~75-minute end-to-end run before a reviewer gets a "build passed" signal. Waiting that long for feedback tempts reviewers to merge on vibes.

## Decision

Two runner topologies, used for different purposes:

- **Full suite (existing) stays on GKE**, invoked by `tests.yml` and the release workflow. This is the authority for correctness on the cloud surface area.
- **New `pr-fast` smoke suite on kind**, invoked on every PR, targets <5 minutes end-to-end. It runs helm-template unit tests plus a minimal "install the chart on a real, local Kubernetes and make sure the pod comes up" smoke test.

Both remain on GitHub-hosted runners. Self-hosted runners were considered and rejected: the test suite's wall-clock is dominated by GKE spin-up (8-12 minutes per cluster), not by compute, so a self-hosted runner fleet would need maintenance, security patching, and capacity planning without meaningfully accelerating the critical path.

## Consequences

### Positive

- PR feedback drops from 45-75 minutes to <5 minutes for the majority of changes.
- `pr-fast` catches the class of bugs — invalid YAML, broken `{{ if }}` branches, missing required values, reference to undefined template — that a full GKE run would eventually flag but very slowly.
- Failures in `pr-fast` are cheap to reproduce locally (kind runs everywhere) so contributors don't need cloud access to debug most chart-level regressions.
- GKE capacity stays available for the cases that actually need it: merges to `dev`, scheduled auto-releases, and manual dispatches.

### Negative

- Two CI systems to maintain. The smoke suite must stay in sync with the chart structure — a new chart added to the repo must be added to `pr-fast` too, or it goes untested at the PR level.
- kind cannot faithfully emulate everything: no LoadBalancer IPs, no workload identity, no PD disk behavior. Tests that depend on those must stay on GKE. This is documented in the `pr-fast` workflow comments.

### Neutral

- Some tests may duplicate between kind and GKE (a chart install smoke test runs in both). That duplication is intentional — fast feedback first, authoritative verification second.

## Alternatives considered

- **Minikube instead of kind**: both work. kind is faster to start (~30s vs ~90s) and is the Kubernetes project's official testing tool, so picked for recognizability and startup speed.
- **Self-hosted runner fleet**: rejected (see above).
- **`pr-fast` against a persistent pre-warmed GKE cluster**: rejected because cross-PR state contamination is the #1 source of flakes in the current GKE pipeline and we don't want to reintroduce it in a faster envelope.

## Follow-ups tracked outside this ADR

- Gating rules: eventually `pr-fast` should be a required check on `dev`, with the full suite remaining required before release. For now, `pr-fast` is advisory.
- JUnit artifacts + a flakiness dashboard are out of scope for this CI overhaul cycle but worth revisiting if `pr-fast` introduces new flakes.
