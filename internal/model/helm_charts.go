package model

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
)

var _, thisFile, _, _ = runtime.Caller(0)
var modelDir = path.Dir(thisFile)

var LoadBalancerHelmChart = newHelmChart("neo4j-cluster-loadbalancer")

var HeadlessServiceHelmChart = newHelmChart("neo4j-cluster-headless-service")

var StandaloneHelmChart = newNeo4jHelmChart("neo4j-standalone", []string{"community", "enterprise"})

var BackupHelmChart = newHelmChart("neo4j-admin")

var ClusterCoreHelmChart = newNeo4jHelmChart("neo4j-cluster-core", []string{"enterprise"})

var ClusterReadReplicaHelmChart = newNeo4jHelmChart("neo4j-cluster-read-replica", []string{"enterprise"})

var PrimaryHelmCharts = []Neo4jHelmChartBuilder{StandaloneHelmChart, ClusterCoreHelmChart}

type helmChart struct {
	path     string
	editions []string
}

type HelmChartBuilder interface {
	getPath() string
	Name() string
}

type Neo4jHelmChartBuilder interface {
	HelmChartBuilder
	GetEditions() []string
	SupportsEdition(edition string) bool
}

func (h *helmChart) getPath() string {
	return h.path
}

func (h *helmChart) Name() string {
	dir, file := filepath.Split(h.path)
	if file != "" {
		return file
	} else {
		return dir
	}
}

func (h *helmChart) GetEditions() []string {
	return h.editions
}

func (h *helmChart) SupportsEdition(edition string) bool {
	for _, supportedEdition := range h.editions {
		if edition == supportedEdition {
			return true
		}
	}
	return false
}

func chartExistsAt(path string) (bool, error) {
	if fileInfo, err := os.Stat(path); err == nil {
		if filepath.Ext(path) == ".yaml" && !fileInfo.IsDir() {
			return true, nil
		}
		if fileInfo.IsDir() {
			return chartExistsAt(filepath.Join(path, "Chart.yaml"))
		}
		return false, fmt.Errorf("unexpected error occured. File %s returned fileInfo: %v", path, fileInfo)
	} else {
		return false, err
	}
}

func newHelmChart(helmChartName string) HelmChartBuilder {
	filepath := path.Join(path.Join(modelDir, "../.."), helmChartName)
	if exists, err := chartExistsAt(filepath); err != nil || !exists {
		panic(err)
	}
	return &helmChart{filepath, nil}
}

func newNeo4jHelmChart(helmChartName string, editions []string) Neo4jHelmChartBuilder {
	filepath := path.Join(path.Join(modelDir, "../.."), helmChartName)
	if exists, err := chartExistsAt(filepath); err != nil || !exists {
		panic(err)
	}
	return &helmChart{filepath, editions}
}
