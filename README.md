# Helm-Charts

This repository contains Helm charts that supports both Neo4j standalone and Neo4j clusters

Helm charts for Neo4j clusters are supported from version >= 4.4.0

Helm charts can be downloaded from [here](https://neo4j.com/download-center/#helm)

[Full Documentation can be found here](https://neo4j.com/docs/operations-manual/current/kubernetes/)

## Examples
See the `examples` directory for common usage patterns of this Helm Chart

* [Dynamic volumes with dedicated storage class](../dev/examples/dedicated-storage-class-cluster/README.md)
* [Using Bloom and GDS Plugins](../dev/examples/bloom-gds-license/README.md)
* [Manually created disks with a volume selector (Standalone)](../dev/examples/persistent-volume-selector-standalone/README.md)
* [Manually created disks with a volume selector (Cluster)](../dev/examples/persistent-volume-selector-cluster/README.md)
* [Manually created disks with a pre provisioned PVC](../dev/examples/persistent-volume-manual/README.md)
* [Multi AKS cluster](../dev/examples/multi-cluster/README.md)

 
