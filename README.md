# Helm-Charts

This repository contains Helm charts that supports both Neo4j standalone and Neo4j clusters

Helm charts for Neo4j clusters are supported from version >= 4.4.0

Helm charts can be downloaded from [here](https://neo4j.com/download-center/#helm)

[Full Documentation can be found here](https://neo4j.com/docs/operations-manual/current/kubernetes/)

## Examples
See the `examples` directory for common usage patterns of this Helm Chart

* [Dynamic volumes with dedicated storage class](../blob/dev/dedicated-storage-class-cluster/README.md)
* [Using Bloom and GDS Plugins](../blob/dev/bloom-gds-license/README.md)
* [Manually created disks with a volume selector (Standalone)](../blob/dev/persistent-volume-selector-standalone/README.md)
* [Manually created disks with a volume selector (Cluster)](../blob/dev/persistent-volume-selector-cluster/README.md)
* [Manually created disks with a pre provisioned PVC](../blob/dev/persistent-volume-manual/README.md)
* [Multi AKS cluster](../blob/dev/multi-cluster/README.md)

 
