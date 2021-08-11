package internal_test

import (
	"github.com/sergi/go-diff/diffmatchpatch"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"testing"
)

var acceptableChartsDifferences = []string{
	"name: neo4j-standalone\n",
	"name: neo4j-cluster-core\n",
}

// The charts files for Standalone and Core installations must be kept in sync.
// The only permitted difference between them is the name
func TestCoreChartMatchesStandalone(t *testing.T) {

	standaloneChartFile := "neo4j-standalone/Chart.yaml"
	clusterChartFile := "neo4j-cluster-core/Chart.yaml"

	assertDiffIsAcceptable(t, standaloneChartFile, clusterChartFile, acceptableChartsDifferences)
}

var acceptableValuesDifferences = []string{
	`  # Neo4j Clustering requires Enterprise Edition`+"\n"+`  edition: "enterprise"` + "\n",
	`  # Neo4j Edition to use (community|enterprise)`+"\n"+`  edition: "community"`+"\n"+`  # set edition: "enterprise" to use Neo4j Enterprise Edition` + "\n",
	`  dbms.mode: "CORE"` + "\n",
	`  causal_clustering.middleware.akka.allow_any_core_to_bootstrap: "true"` + "\n",
	`    # if selectCluster is true load balancer will select any instance of the same dbms.mode`+"\n"+`    selectCluster: true`+"\n\n",
}

// The values files for Standalone and Core installations must be kept in sync.
// The only permitted difference between them is the default dbms.mode
func TestCoreValuesMatchesStandalone(t *testing.T) {

	standaloneValuesFile := "neo4j-standalone/values.yaml"
	clusterCoreValuesFile := "neo4j-cluster-core/values.yaml"

	assertDiffIsAcceptable(t, standaloneValuesFile, clusterCoreValuesFile, acceptableValuesDifferences)
}

// The Neo4j Enterprise default configuration files for Standalone and Core installations must be kept in sync.
func TestCoreEnterpriseConfMatchesStandalone(t *testing.T) {

	standaloneValuesFile := "neo4j-standalone/neo4j-enterprise.conf"
	clusterCoreValuesFile := "neo4j-cluster-core/neo4j-enterprise.conf"

	assertDiffIsAcceptable(t, standaloneValuesFile, clusterCoreValuesFile, nil)
}

func assertDiffIsAcceptable(t *testing.T, standaloneChartFile string, clusterChartFile string, acceptableDifferences []string) {
	if !assert.NotEqual(t, standaloneChartFile, clusterChartFile) {
		return
	}

	standaloneBytes, err := ioutil.ReadFile(standaloneChartFile)
	if !assert.NoError(t, err) {
		return
	}
	clusterCoreBytes, err := ioutil.ReadFile(clusterChartFile)
	if !assert.NoError(t, err) {
		return
	}

	dmp := diffmatchpatch.New()
	c1, c2, lineArray := dmp.DiffLinesToChars(string(standaloneBytes), string(clusterCoreBytes))

	diffs := dmp.DiffMain(c1, c2, true)
	result := dmp.DiffCharsToLines(diffs, lineArray)

	for _, diff := range dmp.DiffCleanupSemanticLossless(result) {
		if diff.Type != diffmatchpatch.DiffEqual {
			assert.Contains(t, acceptableDifferences, diff.Text)
		}
	}
}
