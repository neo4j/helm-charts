# Neo4j Helm Charts

This repository contains work on official Neo4j Helm Charts


# Working with different Cloud / K8s Provider Specific Instructions

We target different Kubernetes providers. This section contains instructions on working with each provider. 

## Kubernetes IN Docker (KIND)

TODO: Write instructions for kind

## Google cloud

### Create a GKE cluster

Set the `CLOUDSDK_` variables in `devenv.local.template`

```
source devenv
gcloud-auth
gcloud-create-gke-cluster
```

### Run tests

This assumes you have a running GKE cluster

```
source devenv
gcloud-auth
gcloud-configure-kubectl
run-go-tests
```

### Teardown a GKE cluster

This will delete the GKE cluster. It does not delete persistent disks associated with the cluster. 
If you have created persistent disks manually you need to remove them manually.
```
source devenv
gcloud-auth
gcloud-configure-kubectl
gcloud-delete-gke-cluster
```

### ADVANCED: cleanup leaked Google Cloud resources

We cannot guarantee that every developer or teamcity run cleans up after itself (TC runs on spot instances after all).

This will cleanup the _typical_ resources that are created in the course of testing and development of neo4j-helm-chart IF they are more than 24 hours old.


This deletes Google Cloud persistent disks and GKE kubernetes clusters without confirmation!!!


This runs against all resources in the currently configured project - ENSURE THAT YOU ARE IN THE CORRECT GOOGLE CLOUD PROJECT BEFORE RUNNING THIS 

```
source devenv
gcloud-auth
./build/gcloud-cleanup-leaked-resources
```