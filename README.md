# Neo4j Helm Charts

This repository contains work on official Neo4j Helm Charts


# Usage

The helm chart is in the `neo4j/` directory.

## Installation

```
# Currently we require you to have a pre-existing persistent disk. To create a suitable one use:
gcloud-create-persistent-disk "my-release-name"

cd neo4j
helm install "my-release-name" . [-f <values.yaml>]

# Find the external IP address of the created service and then connect using browser <IP>:7474 
kubectl get svc 
```

### Using a pre-existing Neo4j Configuration

This requires checking out the helm chart and modifying it. Unfortunately for packaged charts helm does not permit reading files from the filesystem. 
```
cd neo4j

# modify or replace the neo4j.conf file as required
# e.g. echo "dbms.allow_upgrade=true" >> neo4j.conf

helm install "my-release-name" .
```

### Passing Additional Neo4j Configuration

To set additional config (or override) to the "standard" neo4j.conf that is packaged with the version of neo4j being used. Set the `config` property in values.yaml 

Or you can use `--set` if values are quoted and the dots in neo4j config keys are escaped with `\`
```
helm install "my-release-name" . --set 'config.dbms\.tx_state\.memory_allocation=OFF_HEAP'
```

## Upgrade

```
helm upgrade "MyReleaseName" .
```


# Development: Working with different Cloud / K8s Provider Specific Instructions

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

This assumes you have a running GKE cluster (or you can use `gcloud-create-gke-cluster` to create a GKE cluster if you don't have one already)

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