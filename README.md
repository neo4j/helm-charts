# Neo4j Helm Charts

This repository contains work on official Neo4j Helm Charts


# Usage

The helm chart is in the `neo4j/` directory.

## Installation

```
# Currently we require you to have a pre-existing persistent disk. To create a suitable one uncomment the relevant script:
# gcloud-create-persistent-disk "my-release-name"
# docker-desktop-create-persistent-disk my-neo4j "my-release-name"

helm install "my-release-name" ./neo4j [-f <values.yaml>] [--set key=value]

# Find the external IP address of the created service and then connect using browser <EXTERNAL-IP>:7474
kubectl get svc 
```

## Upgrade

```
helm upgrade "MyReleaseName" . [-f <values.yaml>] [--set key=value]
```

## Configuring Neo4j

### Setting Neo4j Configuration in yaml

Additional Neo4j configuration can be added to the `config` property in values.yaml as a yaml-object. We recommend quoting all neo4j configuration values in yaml. Neo4j expects configuration values to be strings, certain values will be parsed as non-string types if they are not quoted for example: `true`, `false`, `yes`, `no` and any numeric values. This is the best way to manage Neo4j configuration in helm.

```
# values.yaml

# neo4j config as yaml object
config:
  dbms.tx_state.memory_allocation: "OFF_HEAP"
  causal_clustering.catchup_batch_size: "64"
  metrics.csv.enabled: "true"

```

Neo4j configuration values can also be passed to helm using the `--set` parameter but these must be quoted, and the dots in neo4j configuration keys should be escaped with backslashes:

```
helm upgrade "my-release-name" . --set 'config.dbms\.tx_state\.memory_allocation=OFF_HEAP'
```

### Using a pre-existing Neo4j Configuration

You can include existing neo4j.conf properties-file content in the `configImport` field of values.yaml. This has to be included as a [multi-line string](https://yaml-multiline.info) we recommend removing all comments and carefully checking the indentation of text included in this way to avoid yaml parsing errors.

```
# values.yaml

# multiline string using block scalar (requires indentation of multiline text)
configImport: |
  dbms.allow_upgrade=true
  dbms.connector.bolt.enabled=true
  dbms.connector.bolt.tls_level=DISABLED
  dbms.connector.bolt.listen_address=:7687
  dbms.connector.bolt.advertised_address=:7687

```

# Development: Working with different Cloud / K8s Provider Specific Instructions

We target different Kubernetes providers. This section contains instructions on working with each provider. 

## Docker Desktop

This requires that you have [enabled kubernetes on Docker Desktop](https://docs.docker.com/desktop/kubernetes/#enable-kubernetes)

Make sure that you have not already got any neo4j instances using the default neo4j ports (e.g. Neo4j Desktop)
```
source devenv

docker-desktop-configure-kubectl
docker-desktop-create-persistent-disk my-neo4j "my-release-name"

helm install "my-release-name" ./neo4j [-f <values.yaml>] [--set key=value]

# you may need to wait a minute or two for browser access to work
open http://localhost:7474
```

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