package index_updater

import (
	"gopkg.in/yaml.v3"
)

type IndexYaml struct {
	APIVersion string `yaml:"apiVersion"`
	Generated  string `yaml:"generated"`
	Entries    struct {
		Neo4j                       []*Entry `yaml:"neo4j"`
		Neo4jPersistentVolume       []*Entry `yaml:"neo4j-persistent-volume"`
		Neo4jHeadlessService        []*Entry `yaml:"neo4j-headless-service"`
		Neo4jStandalone             []*Entry `yaml:"neo4j-standalone"`
		Neo4jClusterCore            []*Entry `yaml:"neo4j-cluster-core"`
		Neo4jClusterReadReplica     []*Entry `yaml:"neo4j-cluster-read-replica"`
		Neo4jClusterLoadbalancer    []*Entry `yaml:"neo4j-cluster-loadbalancer"`
		Neo4jClusterHeadlessService []*Entry `yaml:"neo4j-cluster-headless-service"`
	} `yaml:"entries"`
}

// NewIndexYaml reads the current index.yaml and returns the same
func NewIndexYaml() (IndexYaml, error) {
	fileBytes, err := readIndexYaml()
	if err != nil {
		return IndexYaml{}, err
	}

	indexYaml := IndexYaml{}
	err = yaml.Unmarshal(fileBytes, &indexYaml)
	if err != nil {
		return IndexYaml{}, err
	}
	return indexYaml, nil
}

func (i *IndexYaml) UpdateEntries(es []*Entry) {
	for _, e := range es {
		if e.Name == "neo4j-standalone" {
			i.Entries.Neo4jStandalone = append([]*Entry{e}, i.Entries.Neo4jStandalone...)
		} else if e.Name == "neo4j-cluster-core" {
			i.Entries.Neo4jClusterCore = append([]*Entry{e}, i.Entries.Neo4jClusterCore...)
		} else if e.Name == "neo4j-cluster-read-replica" {
			i.Entries.Neo4jClusterReadReplica = append([]*Entry{e}, i.Entries.Neo4jClusterReadReplica...)
		} else if e.Name == "neo4j-cluster-loadbalancer" {
			i.Entries.Neo4jClusterLoadbalancer = append([]*Entry{e}, i.Entries.Neo4jClusterLoadbalancer...)
		} else if e.Name == "neo4j-cluster-headless-service" {
			i.Entries.Neo4jClusterHeadlessService = append([]*Entry{e}, i.Entries.Neo4jClusterHeadlessService...)
		} else if e.Name == "neo4j" {
			i.Entries.Neo4j = append([]*Entry{e}, i.Entries.Neo4j...)
		} else if e.Name == "neo4j-persistent-volume" {
			i.Entries.Neo4jPersistentVolume = append([]*Entry{e}, i.Entries.Neo4jPersistentVolume...)
		} else if e.Name == "neo4j-headless-service" {
			i.Entries.Neo4jHeadlessService = append([]*Entry{e}, i.Entries.Neo4jHeadlessService...)
		}
	}
}
