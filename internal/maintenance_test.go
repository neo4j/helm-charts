package internal

import (
	"github.com/stretchr/testify/assert"
	"strings"
	"time"
)
import "testing"

func maintenanceTests(name *ReleaseName) []SubTest {
	return []SubTest{
		{name: "Create Node", test: func(t *testing.T) { assert.NoError(t, CreateNode(name), "Create Node should succeed") }},
		{name: "Maintenance Mode", test: func(t *testing.T) { assert.NoError(t, CheckMaintenanceMode(t, name), "Check maintenance mode") }},
		{name: "Count Nodes", test: func(t *testing.T) { assert.NoError(t, CheckNodeCount(t, name), "Count Nodes should succeed") }},
	}
}

func CheckMaintenanceMode(t *testing.T, releaseName *ReleaseName) error {
	cmd := []string{
		"jps",
	}

	stdout, stderr, err := ExecInPod(releaseName, cmd)
	assert.Len(t, strings.Split(stdout, "\n"), 2)
	assert.Contains(t, stdout, "EntryPoint")
	assert.Empty(t, stderr)
	assert.NoError(t, err)

	if err != nil {
		return err
	}

	err = run("helm", baseHelmCommand("upgrade", releaseName, "--set", "neo4j.offlineMaintenanceModeEnabled=true")...)
	assert.NoError(t, err)
	if err != nil {
		return err
	}

	time.Sleep(30 * time.Second)
	err = run("kubectl", "--namespace", string(releaseName.namespace()), "wait", "--for=condition=Initialized", "pod/" + releaseName.podName())
	assert.NoError(t, err)
	if err != nil {
		return err
	}
	time.Sleep(30 * time.Second)

	stdout, stderr, err = ExecInPod(releaseName, cmd)
	assert.Len(t, strings.Split(stdout, "\n"), 1)
	assert.NotContains(t, stdout, "neo4j")
	assert.Empty(t, stderr)
	assert.NoError(t, err)

	err = run("helm", baseHelmCommand("upgrade", releaseName,
		"--set", "neo4j.offlineMaintenanceModeEnabled=false", "--wait", "--timeout", "300s")...)
	assert.NoError(t, err)
	if err != nil {
		return err
	}
	time.Sleep(30 * time.Second)
	err = run("kubectl", "--namespace", string(releaseName.namespace()), "rollout", "status", "--watch", "--timeout=120s", "statefulset/" + string(*releaseName))
	assert.NoError(t, err)
	if err != nil {
		return err
	}
	time.Sleep(30 * time.Second)

	return err
}

func TestMaintenanceInGCloudK8s(t *testing.T) {
	releaseName := ReleaseName("maintenance")
	t.Parallel()

	t.Logf("Starting setup of '%s'", t.Name())
	cleanup, err := installNeo4j(t, &releaseName)
	defer cleanup()

	if !assert.NoError(t, err) {
		return
	}

	if err := configureNeo4j(&releaseName); err != nil {
		assert.NoError(t, err)
		return
	}

	t.Logf("Succeeded with setup of '%s'", t.Name())

	runSubTests(t, maintenanceTests(&releaseName))
}
