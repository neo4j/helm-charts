# build/

This directory contains scripts that are expected to run in TeamCity

## Assumptions

These scripts may assume:

 - Scripts in `<git root>/bin` are available on the path
 - TeamCity-related environment variables are set

## Rules

Scripts that are specific to a particular Cloud / Kubernetes provider should be prefixed with the provider name.

- `gcloud-` for Google cloud
- `kind-` for Kubernetes IN Docker
- `aws-` for AWS

etc.

N.b. if there is a risk of name-collisions between scripts in this directory and scripts in the `bin/` directory then append `-tc` to the name of the script in this directory

For example `gcloud-create-gke-cluster` becomes `gcloud-create-gke-cluster-tc`
