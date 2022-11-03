# Example - Install Neo4j Cluster using manually created disks and a Persistent Volume selector

This example uses manually provisioned cloud disks for the Neo4j storage volumes and installs a Neo4j 3 server cluster.
The `neo4j-persistent-volume` chart is used to configure 3 persistent volumes for each disk and manual storage class.
The `neo4j` chart then configures the StatefulSet to use a selector based volume claim template. 
A PVC will be dynamically provisioned for each PV.

## Install in AWS
```shell
export AWS_ZONE=us-east-1a
./install-example-aws.sh $AWS_ZONE
```

## Cleanup AWS
```shell
./cleanup-example-aws.sh
```

## Install in GCP
```shell
export CLOUDSDK_CORE_PROJECT=my-gcp-project
export CLOUDSDK_COMPUTE_ZONE=my-zone
./install-example-gcp.sh
```

## Cleanup GCP
```shell
./cleanup-example-gcp.sh
```

## Install in Azure
```shell
export AKS_CLUSTER_NAME=my-neo4j-cluster
export AZURE_RESOURCE_GROUP=myResourceGroup
export AZURE_LOCATION=mylocation
./install-example-azure.sh $AKS_CLUSTER_NAME $AZURE_RESOURCE_GROUP $AZURE_LOCATION
```

## Cleanup Azure
```shell
./cleanup-example-azure.sh
```
