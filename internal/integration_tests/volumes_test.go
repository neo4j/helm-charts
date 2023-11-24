package integration_tests

import (
	"github.com/neo4j/helm-charts/internal/model"
	"github.com/neo4j/helm-charts/internal/resources"
	"github.com/stretchr/testify/assert"
)
import "testing"

func TestVolumesInGCloudK8s(t *testing.T) {
	t.Parallel()
	chart := model.StandaloneHelmChart
	releaseName := model.NewReleaseName("volumes-" + TestRunIdentifier)

	t.Logf("Starting setup of '%s'", t.Name())
	_, err := installNeo4j(t, releaseName, chart, resources.TestAntiAffinityRule.HelmArgs()...)
	t.Cleanup(standaloneCleanup(t, releaseName))

	if !assert.NoError(t, err) {
		return
	}

	t.Logf("Succeeded with setup of '%s'", t.Name())

	runSubTests(t, volumesTests(releaseName, chart))
}
