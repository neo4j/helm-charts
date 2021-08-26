package integration_tests

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"k8s.io/utils/env"
	"neo4j.com/helm-charts-tests/internal/integration_tests/gcloud"
	"neo4j.com/helm-charts-tests/internal/model"
)
import "testing"

type SubTest struct {
	name string
	test func(*testing.T)
}

var neo4jEdition = env.GetString("NEO4J_EDITION", "enterprise")
var neo4jConfFile = fmt.Sprintf("neo4j-standalone/neo4j-%s.conf", neo4jEdition)

func k8sTests(name *model.ReleaseName) []SubTest {
	return []SubTest{
		{name: "Check Neo4j Configuration", test: func(t *testing.T) { assert.NoError(t, CheckNeo4jConfiguration(t, name, neo4jConfFile), "Neo4j Config check should succeed") }},
		{name: "Create Node", test: func(t *testing.T) { assert.NoError(t, CreateNode(t, name), "Create Node should succeed") }},
		{name: "Delete Resources", test: func(t *testing.T) { assert.NoError(t, ResourcesCleanup(t, name), "Cleanup Resources should succeed") }},
		{name: "Reinstall Resources", test: func(t *testing.T) { assert.NoError(t, ResourcesReinstall(t, name), "Reinstall Resources should succeed") }},
		{name: "Count Nodes", test: func(t *testing.T) { assert.NoError(t, CheckNodeCount(t, name), "Count Nodes should succeed") }},
		{name: "Check Probes", test: func(t *testing.T) { assert.NoError(t, CheckProbes(t, name), "Probes Matching should succeed") }},
		{name: "Check Service Annotations", test: func(t *testing.T) { assert.NoError(t, CheckServiceAnnotations(t, name), "Services should have annotations") }},
		{name: "Check RunAsNonRoot", test: func(t *testing.T) { assert.NoError(t, RunAsNonRoot(t, name), "RunAsNonRoot check should succeed") }},
		{name: "Exec in Pod", test: func(t *testing.T) { assert.NoError(t, CheckExecInPod(t, name), "Exec in Pod should succeed") }},
	}
}

// Install Neo4j on the provided GKE K8s cluster and then run the tests from the table above using it
func TestInstallOnGCloudK8s(t *testing.T) {
	releaseName := model.ReleaseName("install-" + TestRunIdentifier)
	t.Parallel()
	t.Logf("Starting setup of '%s'", t.Name())
	cleanup, err := installNeo4j(t, &releaseName)
	defer cleanup()

	if !assert.NoError(t, err) {
		t.Logf("%#v", err)
		return
	}

	if err := configureNeo4j(&releaseName); err != nil {
		assert.NoError(t, err)
		return
	}

	t.Logf("Succeeded with setup of '%s'", t.Name())

	runSubTests(t, k8sTests(&releaseName))
}

func runSubTests(t *testing.T, subTests []SubTest) {
	defer t.Logf("Finished running all tests in '%s'", t.Name())

	for _, test := range subTests {

		t.Run(test.name, func(t *testing.T) {
			t.Logf("Started running subtest '%s'", t.Name())
			defer t.Logf("Finished running subtest '%s'", t.Name())

			test.test(t)
		})
	}
}

func installNeo4j(t *testing.T, releaseName *model.ReleaseName) (func(), error) {
	cleanup, err := InstallNeo4jInGcloud(t, gcloud.CurrentZone(), gcloud.CurrentProject(), releaseName)

	return func() {
		t.Logf("Beginning cleanup of '%s'", t.Name())
		defer t.Logf("Finished cleanup of '%s'", t.Name())

		if cleanup != nil {
			err := cleanup()
			if err != nil {
				t.Logf("Error during cleanup: %s", err)
			}
		}
	}, err
}

func configureNeo4j(releaseName *model.ReleaseName) error {
	return nil
}
