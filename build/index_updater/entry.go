package index_updater

import "fmt"

type Entry struct {
	Created     string   `yaml:"created"`
	Description string   `yaml:"description"`
	Digest      string   `yaml:"digest"`
	Home        string   `yaml:"home"`
	Name        string   `yaml:"name"`
	Sources     []string `yaml:"sources"`
	Urls        []string `yaml:"urls"`
	Version     string   `yaml:"version"`
}

// NewEntry creates a new entry type with the provided info
func NewEntry(sha string, chartName string, branchName string) *Entry {
	e := &Entry{
		Created: getCurrentDateAndTime(),
		Digest:  sha,
		Home:    "https://github.com/neo4j/helm-charts",
		Name:    chartName,
		Sources: []string{
			fmt.Sprintf("https://github.com/neo4j/helm-charts/tree/%s/%s", chartName, branchName),
		},
		Urls: []string{
			fmt.Sprintf("https://github.com/neo4j/helm-charts/releases/download/%s/%s-%s.tgz", version, chartName, version),
		},
		Version: version,
	}
	e.setDescription()
	return e
}

func (e *Entry) setDescription() {
	if e.Name == "neo4j-standalone" {
		e.Description = fmt.Sprintf("Neo4j Standalone %s", version)
	} else if e.Name == "neo4j-cluster-core" {
		e.Description = fmt.Sprintf("Neo4j Cluster Core %s", version)
	} else if e.Name == "neo4j-cluster-read-replica" {
		e.Description = fmt.Sprintf("Neo4j Cluster Read Replica %s", version)
	} else if e.Name == "neo4j-cluster-loadbalancer" {
		e.Description = fmt.Sprintf("Neo4j Cluster LoadBalancer %s", version)
	} else if e.Name == "neo4j-cluster-headless-service" {
		e.Description = fmt.Sprintf("Neo4j Cluster Headless Service %s", version)
	} else if e.Name == "neo4j" {
		e.Description = fmt.Sprintf("Neo4j %s", version)
	} else if e.Name == "neo4j-persistent-volume" {
		e.Description = fmt.Sprintf("Neo4j Persistent Volume %s", version)
	} else if e.Name == "neo4j-headless-service" {
		e.Description = fmt.Sprintf("Neo4j Headless Service %s", version)
	}

}
