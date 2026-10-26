# Changelog

## [5.26.32-1] - 2026-10-26

*Neo4j docker image: `5.26.32`*

- See [Neo4j release notes](https://neo4j.com/release-notes/database/neo4j-5/) for upstream DB changes

### Helm-charts changes

- Patch networking, cryptography, gRPC, OpenTelemetry, AWS SDK v2, and protobuf dependencies in repository-managed Go modules. ([#563](https://github.com/neo4j/helm-charts/pull/563))
- Upgrade helper image Go toolchains to 1.26.10 and Alpine bases to 3.24, including the reverse-proxy runtime to 3.24.3. ([#563](https://github.com/neo4j/helm-charts/pull/563))
- Upgrade Helm build and CI tooling to 3.22.1 and update the GKE authentication plugin package name in the CI image. ([#563](https://github.com/neo4j/helm-charts/pull/563))
- Add configurable retries for the Neo4j operations Job through neo4j.operations.backoffLimit, defaulting to 0. ([#561](https://github.com/neo4j/helm-charts/pull/561))

## [5.26.32] - 2026-10-12

*Neo4j docker image: `5.26.32`*

- See [Neo4j release notes](https://neo4j.com/release-notes/database/neo4j-5/) for upstream DB changes
- No user-facing helm-charts changes (maintenance release).

## [5.26.31] - 2026-09-22

*Neo4j docker image: `5.26.31`*

- See [Neo4j release notes](https://neo4j.com/release-notes/database/neo4j-5/) for upstream DB changes
- No user-facing helm-charts changes (maintenance release).

## [5.26.30] - 2026-08-25

*Neo4j docker image: `5.26.30`*

- See [Neo4j release notes](https://neo4j.com/release-notes/database/neo4j-5/) for upstream DB changes
- No user-facing helm-charts changes (maintenance release).

## [5.26.29] - 2026-08-05

*Neo4j docker image: `5.26.29`*

- See [Neo4j release notes](https://neo4j.com/release-notes/database/neo4j-5/) for upstream DB changes
- No user-facing helm-charts changes (maintenance release).

## [5.26.28] - 2026-07-02

*Neo4j docker image: `5.26.28`*

- See [Neo4j release notes](https://neo4j.com/release-notes/database/neo4j-5/) for upstream DB changes
- No user-facing helm-charts changes (maintenance release).

## [5.26.27] - 2026-06-09

*Neo4j docker image: `5.26.27`*

- See [Neo4j release notes](https://neo4j.com/release-notes/database/neo4j-5/) for upstream DB changes
- No user-facing helm-charts changes (maintenance release).

## [5.26.26] - 2026-05-12

*Neo4j docker image: `5.26.26`*

- See [Neo4j release notes](https://neo4j.com/release-notes/database/neo4j-5/) for upstream DB changes
- No user-facing helm-charts changes (maintenance release).

