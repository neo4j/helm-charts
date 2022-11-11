# Example - Install Neo4j Cluster using a dedicated storage class

This example uses a dynamically provisioned volumes using a dedicated storage class. 
A `neo4j-data` StorageClass is created, then PVCs are dynamically created for each cluster server using the storage class.

The example will use the following Helm values
```yaml
neo4j:
  name: dedicated-storage-class
  minimumClusterSize: 3
  acceptLicenseAgreement: "yes"
  edition: enterprise
volumes:
  data:
    mode: dynamic
    dynamic:
      storageClassName: "neo4j-data"
      accessModes:
        - ReadWriteOnce
      requests:
        storage: 100Gi
```

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

## Cleanup the example
```shell
./cleanup.sh
```
