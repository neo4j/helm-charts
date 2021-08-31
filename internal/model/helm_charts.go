package model

import (
	"fmt"
	"os"
	"path/filepath"
)

var LoadBalancerHelmChart = newHelmChart("./neo4j-loadbalancer")

var StandaloneHelmChart = newNeo4jHelmChart("./neo4j-standalone", []string{"community", "enterprise"})

var ClusterCoreHelmChart = newNeo4jHelmChart("./neo4j-cluster-core", []string{"enterprise"})

var PrimaryHelmCharts = []Neo4jHelmChart{StandaloneHelmChart, ClusterCoreHelmChart}

type helmChart struct {
	path     string
	editions []string
}

type HelmChart interface {
	getPath() string
	Name() string
}

type Neo4jHelmChart interface {
	HelmChart
	GetEditions() []string
}

func (h *helmChart) getPath() string {
	return h.path
}

func (h *helmChart) GetEditions() []string {
	return h.editions
}

func (h *helmChart) Name() string {
	dir, file := filepath.Split(h.path)
	if file != "" {
		return file
	} else {
		return dir
	}
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

func newHelmChart(path string) HelmChart {
	if exists, err := chartExistsAt(path); err != nil || !exists {
		panic(err)
	}
	return &helmChart{path, nil}
}

func newNeo4jHelmChart(path string, editions []string) Neo4jHelmChart {
	if exists, err := chartExistsAt(path); err != nil || !exists {
		panic(err)
	}
	return &helmChart{path, editions}
}
