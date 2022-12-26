package unit_tests_test

import (
	"github.com/sergi/go-diff/diffmatchpatch"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"regexp"
	"strings"
	"testing"
)

var acceptableChartsDifferences = []string{
	"name: neo4j-standalone",
	"name: neo4j-cluster-core",
}

// The charts files for Standalone and Core installations must be kept in sync.
// The only permitted difference between them is the name
func TestCoreChartMatchesStandalone(t *testing.T) {

	standaloneChartFile := "neo4j-standalone/Chart.yaml"
	clusterChartFile := "neo4j-cluster-core/Chart.yaml"

	assertDiffIsAcceptable(t, standaloneChartFile, clusterChartFile, acceptableChartsDifferences)
}

var acceptableValuesDifferences = []string{
	`name: ""`,
	`name: "neo4j-cluster"`,
	`edition: "community"`,
	`edition: "enterprise"`,
	"enabled: true",
	"enabled: false\n\n\n    selectCluster: false",
	`dbms.mode: "CORE"`,
	`causal_clustering.middleware.akka.allow_any_core_to_bootstrap: "true"`,
	`cluster:
    privateKey:
      secretName:
      subPath:
    publicCertificate:
      secretName:
      subPath:
    trustedCerts:
      sources: [ ]
    revokedCerts:
      sources: [ ]`,
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

var blankLines = regexp.MustCompile(`(?m)^\s*$`)
var commentsOnly = regexp.MustCompile(`(?m)^\s*#.*$`)

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
	c1, c2, lineArray := dmp.DiffLinesToChars(clean(standaloneBytes), clean(clusterCoreBytes))

	diffs := dmp.DiffMain(c1, c2, true)
	result := dmp.DiffCharsToLines(diffs, lineArray)

	for _, diff := range dmp.DiffCleanupSemanticLossless(result) {
		text := strings.TrimSpace(diff.Text)
		if diff.Type != diffmatchpatch.DiffEqual && text != "" {
			assert.Contains(t, acceptableDifferences, text)
			t.Log("acceptable diff", text)
		}
	}
}

func clean(standaloneBytes []byte) string {
	cleaned := string(commentsOnly.ReplaceAllLiteral(blankLines.ReplaceAllLiteral(standaloneBytes, nil), nil))
	return cleaned
}
