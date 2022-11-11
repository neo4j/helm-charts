# Example - Install Neo4j Cluster using a dedicated storage class

This example uses a dynamically provisioned volumes using a dedicated storage class. 
A `neo4j-data` StorageClass is created, then PVCs are dynamically created for each cluster server using the storage class.

## Install in AWS
```shell
./install-example-aws.sh
```

## Install in GCP
```shell
./install-example-gcp.sh
```

## Install in Azure
```shell

./install-example-azure.sh
```

