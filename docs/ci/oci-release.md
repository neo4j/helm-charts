# OCI Helm chart releases

Release workflows publish the same signed chart archives to three destinations:

* Google Artifact Registry as OCI artifacts
* The classic S3 Helm repository
* The GitHub release

The OCI base path is:

```text
oci://europe-west2-docker.pkg.dev/neo4j-helm/helm-charts
```

The six published charts are `neo4j`, `neo4j-admin`,
`neo4j-headless-service`, `neo4j-persistent-volume`,
`neo4j-reverse-proxy`, and `neo4j-loadbalancer`.

## Release flow

`bin/release/package_charts` creates every signed `.tgz` and `.tgz.prov` file
once. The release then runs these publishers in order:

1. `bin/release/publish_oci` pushes and pulls every OCI chart, verifies its GPG
   provenance, and compares the stored bytes.
2. `bin/release/publish_classic` copies those verified package files to S3 and
   rebuilds the classic repository index.
3. `bin/gcloud/index_yaml_update` updates the repository index and release tag.
4. The GitHub release attaches the same packages, provenance files, and OCI
   digest list.

Artifact Registry tags are immutable. On a retry, `publish_oci` verifies an
existing chart and compares its extracted content with the new local package.
When the content matches, the publisher restores the exact remote package and
provenance bytes locally before the S3 and GitHub publication steps. Different
chart content for an existing version stops the release.

## GitHub configuration

The `helm-production` environment must allow releases from `dev` and `5.26`.
The release job requires these repository variables:

| Variable | Value |
| --- | --- |
| `GCP_PROJECT_ID` | `neo4j-helm` |
| `HELM_OCI_LOCATION` | `europe-west2` |
| `HELM_OCI_REPOSITORY` | `helm-charts` |
| `GCP_WORKLOAD_IDENTITY_PROVIDER` | Full Google workload identity provider name |
| `GCP_HELM_RELEASE_SERVICE_ACCOUNT` | Dedicated Artifact Registry writer service account |

The job requests `id-token: write` only for short-lived Google authentication.
The release service account needs `roles/artifactregistry.writer` on the
`helm-charts` repository and no project-wide role.

## Consumer commands

OCI repositories do not use `helm repo add`.

```bash
helm show chart \
  oci://europe-west2-docker.pkg.dev/neo4j-helm/helm-charts/neo4j \
  --version 2026.7.1

helm install my-neo4j \
  oci://europe-west2-docker.pkg.dev/neo4j-helm/helm-charts/neo4j \
  --version 2026.7.1 \
  --namespace neo4j \
  --create-namespace \
  --values values.yaml
```

See the [Helm OCI registry documentation](https://helm.sh/docs/topics/registries/)
for more client commands.
