# Deploy Neo4j cluster multiple Kubernetes Clusters

See docs for details https://neo4j.com/docs/operations-manual/4.4/kubernetes/multi-dc-cluster/aks/

This is not a common deployment scenario, this example only serves to demonstrate the configuration for customers
that require this feature.

Example requires `Microsoft.Authorization/roleAssignments/write` and `Microsoft.Authorization/roleAssignments/delete`
permissions in Azure.

The example will 

* deploy 3 AKS clusters
* deploy a Neo4j cluster with a server on each AKS cluster.
  * The loadbalancer service will use a private IP that is available across all the AZ clusters. This is because the loadbalancer exposes internal clustering ports
* create an azure application gateway to access the cluster

# Deploy multi zone cluster
```shell
./multi-cluster-example-aks.sh
```

# Cleanup example
```shell
./multi-cluster-cleanup-aks.sh
```
