package integration_tests

import (
	"github.com/neo4j/helm-charts/internal/model"
	"github.com/neo4j/helm-charts/internal/resources"
	"github.com/stretchr/testify/assert"
)
import "testing"

// Install Neo4j on the provided GKE K8s cluster and then run the tests from the table above using it
func TestInstallStandaloneOnGCloudK8s(t *testing.T) {
	releaseName := model.NewReleaseName("install-" + TestRunIdentifier)
	chart := model.StandaloneHelmChart

	t.Parallel()
	t.Logf("Starting setup of '%s'", t.Name())

	defaultHelmArgs := []string{}
	defaultHelmArgs = append(defaultHelmArgs, resources.TestAntiAffinityRule.HelmArgs()...)
	defaultHelmArgs = append(defaultHelmArgs, resources.BloomStandaloneTest.HelmArgs()...)
	_, err := installNeo4j(t, releaseName, chart, defaultHelmArgs...)
	t.Cleanup(standaloneCleanup(t, releaseName))

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
