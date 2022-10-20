package integration_tests

import (
	"github.com/neo4j/helm-charts/internal/model"
	"github.com/neo4j/helm-charts/internal/resources"
	"github.com/stretchr/testify/assert"
)
import "testing"

func TestVolumesInGCloudK8s(t *testing.T) {
	chart := model.Neo4jHelmChartCommunityAndEnterprise
	releaseName := model.NewReleaseName("volumes-" + TestRunIdentifier)
	t.Parallel()

	t.Logf("Starting setup of '%s'", t.Name())
	defaultHelmArgs := []string{}
	defaultHelmArgs = append(defaultHelmArgs, model.DefaultNeo4jNameArg...)
	defaultHelmArgs = append(defaultHelmArgs, resources.TestAntiAffinityRule.HelmArgs()...)
	_, err := installNeo4j(t, releaseName, chart, defaultHelmArgs...)
	t.Cleanup(standaloneCleanup(t, releaseName))

	if !assert.NoError(t, err) {
		return
	}

	t.Logf("Succeeded with setup of '%s'", t.Name())

	runSubTests(t, volumesTests(releaseName, chart))
}
