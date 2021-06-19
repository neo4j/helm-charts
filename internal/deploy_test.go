package internal

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"k8s.io/utils/env"
	"log"
)
import "testing"

type SubTest struct {
	name string
	test func(*testing.T)
}

var neo4jEdition = env.GetString("NEO4J_EDITION", "")
var neo4jConfFile = fmt.Sprintf("neo4j/neo4j-%s.conf", neo4jEdition)

var k8sTests = []SubTest{
	{name: "Check Neo4j Configuration", test: func(t *testing.T) { assert.NoError(t, CheckNeo4jConfiguration(t, (&Neo4jConfiguration{}).PopulateFromFile(neo4jConfFile)), "Neo4j Config check should succeed") }},
	{name: "Create Node", test: func(t *testing.T) { assert.NoError(t, CreateNode(), "Create Node should succeed") }},
	{name: "Delete Resources", test: func(t *testing.T) { assert.NoError(t, ResourcesCleanup(), "Cleanup Resources should succeed") }},
	{name: "Reinstall Resources", test: func(t *testing.T) { assert.NoError(t, ResourcesReinstall(), "Reinstall Resources should succeed") }},
	{name: "Count Nodes", test: func(t *testing.T) { assert.NoError(t, CheckNodeCount(t), "Count Nodes should succeed") }},
	{name: "Check Probes", test: func(t *testing.T) { assert.NoError(t, CheckProbes(t), "Probes Matching should succeed") }},
	{name: "Check Service Annotations", test: func(t *testing.T) { assert.NoError(t, CheckServiceAnnotations(t), "Services should have annotations") }},
	{name: "Check RunAsNonRoot", test: func(t *testing.T) { assert.NoError(t, RunAsNonRoot(t), "RunAsNonRoot check should succeed") }},
	{name: "Exec in Pod", test: func(t *testing.T) { assert.NoError(t, ExecInPod(t), "Exec in Pod should succeed") }},
}

// Install neo4j on the provided GKE K8s cluster and then run the tests from the table above using it
func TestGCloudK8s(t *testing.T) {

	t.Logf("Starting setup of '%s'", t.Name())
	cleanup := installNeo4j(t)
	defer cleanup()

	if err := configureNeo4j(); err != nil {
		t.Fatal(err)
	}
	t.Logf("Succeeded with setup of '%s'", t.Name())

	runSubTests(t)
}

func runSubTests(t *testing.T) {
	defer t.Logf("Finished running all tests in '%s'", t.Name())

	for _, test := range k8sTests {

		t.Run(test.name, func(t *testing.T) {
			defer t.Logf("Finished running subtest '%s'", t.Name())

			test.test(t)
		})
	}
}

func installNeo4j(t *testing.T) func() {
	cleanup := InstallNeo4j(CurrentZone(), CurrentProject())

	return func() {
		t.Logf("Beginning cleanup of '%s'", t.Name())
		defer t.Logf("Finished cleanup of '%s'", t.Name())

		err := cleanup()
		if err != nil {
			log.Panicf("Error during cleanup: %s", err)
		}
	}
}

func configureNeo4j() error {
	return SetPassword()
}
