# bin/

This directory contains scripts that are useful for the development and testing of neo4j-helm charts.
These scripts ARE NOT intended to be shipped with the helm chart or depended on at runtime in any way.
These scripts ARE NOT permitted to assume that they are running in TeamCity (such scripts should go in the `build/` directory)

## Assumptions

These scripts may assume:

 - `source devenv` has been run
 - All scripts in this directory are available on the path

## Dependencies

 - `kubectl`
 - `go` SDK
 - For gcloud development the `gcloud` cli tool

## Rules

Scripts that are specific to a particular Cloud / Kubernetes provider should be prefixed with the provider name.

 - `gcloud-` for Google cloud
 - `kind-` for Kubernetes IN Docker
 - `aws-` for AWS 

etc.