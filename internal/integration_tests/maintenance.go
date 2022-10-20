package integration_tests

import (
	"github.com/neo4j/helm-charts/internal/model"
	"github.com/stretchr/testify/assert"
	"strings"
	"time"
)
import "testing"

func exitMaintenanceMode(t *testing.T, releaseName model.ReleaseName, chart model.Neo4jHelmChartBuilder, extraArgs ...string) error {
	err := run(
		t, "helm", model.BaseHelmCommand("upgrade", releaseName, chart, model.Neo4jEdition, append(extraArgs, "--set", "neo4j.offlineMaintenanceModeEnabled=false", "--set", "neo4j.name="+model.DefaultNeo4jName, "--wait", "--timeout", "300s")...)...,
	)
	if !assert.NoError(t, err) {
		return err
	}

	err = run(t, "kubectl", "--namespace", string(releaseName.Namespace()), "rollout", "status", "--watch", "--timeout=120s", "statefulset/"+releaseName.String())
	if !assert.NoError(t, err) {
		return err
	}
	return err
}

func enterMaintenanceMode(t *testing.T, releaseName model.ReleaseName, chart model.Neo4jHelmChartBuilder) error {
	err := run(t, "helm", model.BaseHelmCommand("upgrade", releaseName, chart, model.Neo4jEdition, "--set", "neo4j.offlineMaintenanceModeEnabled=true", "--set", "neo4j.name="+model.DefaultNeo4jName)...)

	if !assert.NoError(t, err) {
		return err
	}

	time.Sleep(30 * time.Second)
	err = run(t, "kubectl", "--namespace", string(releaseName.Namespace()), "wait", "--for=condition=Initialized", "--timeout=300s", "pod/"+releaseName.PodName())

	if !assert.NoError(t, err) {
		return err
	}
	time.Sleep(30 * time.Second)

	return err
}

func checkNeo4jNotRunning(t *testing.T, releaseName model.ReleaseName) error {
	cmd := []string{
		"jps",
	}
	stdout, stderr, err := ExecInPod(releaseName, cmd)
	assert.Len(t, strings.Split(stdout, "\n"), 1)
	assert.NotContains(t, stdout, "neo4j")
	assert.Empty(t, stderr)
	assert.NoError(t, err)
	return err
}
