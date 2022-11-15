package index_updater

import (
	"fmt"
	"log"
	"os"
	"strings"
)

var helmRepo = "neo4j"

var chartsv4 = []string{
	"neo4j-standalone",
	"neo4j-cluster-core",
	"neo4j-cluster-headless-service",
	"neo4j-cluster-loadbalancer",
	"neo4j-cluster-read-replica",
}

var chartsv5 = []string{
	"neo4j",
	"neo4j-persistent-volume",
	"neo4j-headless-service",
}

var version, branch string
var chartsList = chartsv4

func init() {
	value, present := os.LookupEnv("NEO4JVERSION")
	if !present {
		fmt.Println("Please set NEO4JVERSION env variable !!")
		log.Fatal("missing NEO4JVERSION")
	}
	version = value

	value, present = os.LookupEnv("BRANCH")
	if !present {
		fmt.Println("Please set BRANCH env variable !!")
		log.Fatal("missing BRANCH")
	}
	branch = value

	if strings.HasPrefix(branch, "4.") && strings.HasPrefix(version, "5.") {
		log.Fatalf("Invalid combination of branch %s and version %s", branch, version)
	}
	if strings.HasPrefix(branch, "5.") && strings.HasPrefix(version, "4.") {
		log.Fatalf("Invalid combination of branch %s and version %s", branch, version)
	}

	if strings.HasPrefix(branch, "5.") {
		chartsList = chartsv5
	}
}
