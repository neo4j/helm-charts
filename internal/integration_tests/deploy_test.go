package integration_tests

import (
	. "github.com/neo4j/helm-charts/internal/helpers"
	"github.com/neo4j/helm-charts/internal/integration_tests/gcloud"
	"github.com/neo4j/helm-charts/internal/model"
	"github.com/neo4j/helm-charts/internal/resources"
	"github.com/stretchr/testify/assert"
)
import "testing"

type SubTest struct {
	name string
	test func(*testing.T)
}

func k8sTests(name model.ReleaseName, chart model.Neo4jHelmChart) ([]SubTest, error) {
	expectedConfiguration, err := (&model.Neo4jConfiguration{}).PopulateFromFile(Neo4jConfFile)
	if err != nil {
		return nil, err
	}

	return []SubTest{
		{name: "Check Neo4j Logs For Any Errors", test: func(t *testing.T) {
			t.Parallel()
			assert.NoError(t, CheckNeo4jLogsForAnyErrors(t, name), "Neo4j Logs check should succeed")
		}},
		{name: "Check Neo4j Configuration", test: func(t *testing.T) {
			assert.NoError(t, CheckNeo4jConfiguration(t, name, expectedConfiguration), "Neo4j Config check should succeed")
		}},
		{name: "Create Node", test: func(t *testing.T) { assert.NoError(t, CreateNode(t, name), "Create Node should succeed") }},
		{name: "Delete Resources", test: func(t *testing.T) { assert.NoError(t, ResourcesCleanup(t, name), "Cleanup Resources should succeed") }},
		{name: "Reinstall Resources", test: func(t *testing.T) {
			assert.NoError(t, ResourcesReinstall(t, name, chart), "Reinstall Resources should succeed")
		}},
		{name: "Count Nodes", test: func(t *testing.T) { assert.NoError(t, CheckNodeCount(t, name), "Count Nodes should succeed") }},
		{name: "Check Probes", test: func(t *testing.T) { assert.NoError(t, CheckProbes(t, name), "Probes Matching should succeed") }},
		{name: "Check Service Annotations", test: func(t *testing.T) {
			assert.NoError(t, CheckServiceAnnotations(t, name, chart), "Services should have annotations")
		}},
		{name: "Check RunAsNonRoot", test: func(t *testing.T) { assert.NoError(t, RunAsNonRoot(t, name), "RunAsNonRoot check should succeed") }},
		{name: "Exec in Pod", test: func(t *testing.T) { assert.NoError(t, CheckExecInPod(t, name), "Exec in Pod should succeed") }},
	}, err
}

// Install Neo4j on the provided GKE K8s cluster and then run the tests from the table above using it
func TestInstallStandaloneOnGCloudK8s(t *testing.T) {
	releaseName := model.NewReleaseName("install-" + TestRunIdentifier)
	chart := model.StandaloneHelmChart

	t.Parallel()
	t.Logf("Starting setup of '%s'", t.Name())

	cleanup, err := installNeo4j(t, releaseName, chart, resources.TestAntiAffinityRule.HelmArgs()...)
	t.Cleanup(func() { cleanupTest(t, cleanup) })

	if !assert.NoError(t, err) {
		t.Logf("%#v", err)
		return
	}

	t.Logf("Succeeded with setup of '%s'", t.Name())

	subTests, err := k8sTests(releaseName, chart)
	if !assert.NoError(t, err) {
		return
	}
	runSubTests(t, subTests)
}

func runSubTests(t *testing.T, subTests []SubTest) {
	t.Cleanup(func() { t.Logf("Finished running all tests in '%s'", t.Name()) })

	for _, test := range subTests {

		t.Run(test.name, func(t *testing.T) {
			t.Logf("Started running subtest '%s'", t.Name())
			t.Cleanup(func() { t.Logf("Finished running subtest '%s'", t.Name()) })
			test.test(t)
		})
	}
}

func installNeo4j(t *testing.T, releaseName model.ReleaseName, chart model.Neo4jHelmChart, extraHelmInstallArgs ...string) (Closeable, error) {
	closeables := []Closeable{}
	addCloseable := func(closeable Closeable) {
		closeables = append([]Closeable{closeable}, closeables...)
	}

	closeable, err := prepareK8s(t, releaseName)
	addCloseable(closeable)
	if err != nil {
		return AsCloseable(closeables), err
	}

	closeable, err = InstallNeo4jInGcloud(t, gcloud.CurrentZone(), gcloud.CurrentProject(), releaseName, chart, extraHelmInstallArgs...)
	addCloseable(closeable)
	if err != nil {
		return AsCloseable(closeables), err
	}

	err = run(t, "kubectl", "--namespace", string(releaseName.Namespace()), "rollout", "status", "--watch", "--timeout=120s", "statefulset/"+releaseName.String())
	return AsCloseable(closeables), err
}
