package internal

import (
	"github.com/stretchr/testify/assert"
	"log"
)
import "testing"

type SubTest struct {
	name string
	test func(*testing.T)
}

var k8sTests = []SubTest{
	{name: "Create Node", test: func(t *testing.T) { assert.NoError(t, CreateNode(), "Create Node should succeed") }},
	{name: "Count Nodes ", test: func(t *testing.T) { assert.NoError(t, CheckNodeCount(t), "Count Nodes should succeed") }},
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
	cleanup := InstallNeo4j(CurrentZone, CurrentProject)

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
