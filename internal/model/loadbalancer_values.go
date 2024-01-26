package model

type Neo4jLoadBalancerValues struct {
	Neo4j         Neo4jLoadBalancer `yaml:"neo4j"`
	Annotations   map[string]string `yaml:"annotations"`
	Ports         Ports             `yaml:"ports"`
	Selector      map[string]string `yaml:"selector"`
	Spec          Spec              `yaml:"spec"`
	ClusterDomain string            `yaml:"clusterDomain"`
	MultiCluster  bool              `yaml:"multiCluster"`
}
type Neo4jLoadBalancer struct {
	Name    string `yaml:"name"`
	Edition string `yaml:"edition"`
}
