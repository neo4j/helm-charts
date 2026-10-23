# Changelog

## [2026.10.1] - 2026-10-23

*Neo4j docker image: `2026.10.1`*

- See [Neo4j release notes](https://neo4j.com/release-notes/database/neo4j-2026-10/) for upstream DB changes

### Helm-charts changes

- Patch networking, cryptography, gRPC, OpenTelemetry, and cloud SDK dependencies in repository-managed Go modules. ([#562](https://github.com/neo4j/helm-charts/pull/562))
- Upgrade helper image Go toolchains to 1.26.10 and Alpine bases to 3.24, including the reverse-proxy runtime to 3.24.3. ([#562](https://github.com/neo4j/helm-charts/pull/562))
- Upgrade Helm build and CI tooling to 3.22.1 and update the GKE authentication plugin package name in the CI image. ([#562](https://github.com/neo4j/helm-charts/pull/562))

## [2026.10.0] - 2026-10-20

*Neo4j docker image: `2026.10.0`*

- See [Neo4j release notes](https://neo4j.com/release-notes/database/neo4j-2026-10/) for upstream DB changes

### Helm-charts changes

- Add configurable retries for the Neo4j operations Job through neo4j.operations.backoffLimit, defaulting to 0. ([#560](https://github.com/neo4j/helm-charts/pull/560))

## [2026.9.0] - 2026-09-22

*Neo4j docker image: `2026.09.0`*

- See [Neo4j release notes](https://neo4j.com/release-notes/database/neo4j-2026-09/) for upstream DB changes
- No user-facing helm-charts changes (maintenance release).

## [2026.8.1] - 2026-09-14

*Neo4j docker image: `2026.08.1`*

- See [Neo4j release notes](https://neo4j.com/release-notes/database/neo4j-2026-08/) for upstream DB changes
- No user-facing helm-charts changes (maintenance release).

## [2026.7.1] - 2026-08-11

*Neo4j docker image: `2026.07.1`*

- See [Neo4j release notes](https://neo4j.com/release-notes/database/neo4j-2026-07/) for upstream DB changes
- No user-facing helm-charts changes (maintenance release).

## [2026.6.0] - 2026-07-15

*Neo4j docker image: `2026.06.0`*

- See [Neo4j release notes](https://neo4j.com/release-notes/database/neo4j-2026-06/) for upstream DB changes
- No user-facing helm-charts changes (maintenance release).

## [2026.5.0] - 2026-06-02

*Neo4j docker image: `2026.05.0`*

- See [Neo4j release notes](https://neo4j.com/release-notes/database/neo4j-2026-05/) for upstream DB changes
- No user-facing helm-charts changes (maintenance release).

## [2026.4.0] - 2026-04-24

*Neo4j docker image: `2026.04.0`*

- See [Neo4j release notes](https://neo4j.com/release-notes/database/neo4j-2026-04/) for upstream DB changes
- No user-facing helm-charts changes (maintenance release).
