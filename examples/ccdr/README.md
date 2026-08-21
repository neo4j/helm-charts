# Cross-cluster database replication (CCDR)

CCDR keeps a read-only replica database in an independent Neo4j cluster. It is an Enterprise Edition feature available from Neo4j 2026.08. The upstream and replica clusters must run compatible Neo4j versions; follow the Neo4j Operations Manual for the release-specific compatibility rules and Cypher syntax.

## Network replication

Install every member of each Neo4j cluster as a separate Helm release. Start with the common values in [source-cluster-values.yaml](source-cluster-values.yaml) and [replica-cluster-values.yaml](replica-cluster-values.yaml). Allow a new cluster to become healthy using the chart's default internal addresses, then roll the members one at a time while setting a unique externally reachable address on every member:

```yaml
config:
  server.cluster.advertised_address: source-1.example.com:6000

services:
  ccdr:
    enabled: true
    annotations:
      # Add provider-specific annotations for an internal load balancer here.
    spec:
      type: LoadBalancer
```

`services.ccdr` creates a service selecting only that Helm release's pod and exposes only TCP 6000. Enable it only for members that must accept CCDR traffic. Prefer private addresses and restrict ingress to the replica cluster's network. If both clusters share routable Kubernetes DNS and networking, `ClusterIP` is sufficient.

For GKE clusters in different regions, [gke-network-values.yaml](gke-network-values.yaml) creates private internal load balancers with global access enabled so clients in other regions of the same VPC, or connected VPC networks, can reach them.

The advertised address is not inferred from a `LoadBalancer` because its address may be allocated asynchronously. Configure `server.cluster.advertised_address` explicitly to the stable DNS name or IP assigned to that member's CCDR service.

Do not change all advertised addresses simultaneously, and do not use the external addresses during first-ever system-database bootstrap. Wait for the cluster to become healthy, update one member, wait for it to become ready again, and then continue with the next member.

For encrypted inter-cluster traffic, configure `ssl.cluster` in both clusters. Each cluster must trust its own CA and the remote cluster's CA. The certificate SANs must cover the member's advertised address. The values files show the expected secret mounts; create the referenced secrets before installing the releases.

After all members are reachable, create the source database and the replica database using the syntax documented for your Neo4j version. Supply the three upstream `host:6000` endpoints in `replicaConfig.addresses`.

## Backup-pull replication

Backup-pull CCDR does not require `services.ccdr`. Configure a `neo4j-admin` backup job to continuously extend and upload an unbroken backup chain, then create the replica with a `replicaConfig.pullURI` pointing at that object-storage location. [backup-pull-job-gcp-values.yaml](backup-pull-job-gcp-values.yaml) and [backup-pull-replica-values.yaml](backup-pull-replica-values.yaml) provide GKE-oriented starting points.

When `backup.remoteAddressResolution` is enabled, every source member must advertise a backup address that the backup job can resolve. Set this per Helm release, for example `server.backup.advertised_address=source-1-admin.source.svc.cluster.local:6362`. Leaving the default `localhost:6362` causes discovery to succeed but the subsequent backup connection to fail.

The recovery point objective is the time between uploaded differential backups plus the replica pull interval. `db.cluster.backup.pull_interval` controls polling (default `1m` in Neo4j 2026.06 and later). Broken or delayed backup chains increase the possible data-loss window, so monitor both the backup job and replica transaction lag.

The `neo4j/helm-charts-backup` image tags use non-zero-padded release components, for example `2026.6.0` and `2026.8.0`.
