package internal

import (
	"github.com/stretchr/testify/assert"
	"strings"
	"time"
)
import "testing"

func maintenanceTests(name *ReleaseName) []SubTest {
	return []SubTest{
		{name: "Create Node", test: func(t *testing.T) { assert.NoError(t, CreateNode(t, name), "Create Node should succeed") }},
		{name: "Maintenance Mode", test: func(t *testing.T) { assert.NoError(t, CheckMaintenanceMode(t, name), "Check maintenance mode") }},
		{name: "Count Nodes", test: func(t *testing.T) { assert.NoError(t, CheckNodeCount(t, name), "Count Nodes should succeed") }},
	}
}

func CheckMaintenanceMode(t *testing.T, releaseName *ReleaseName) error {
	err := checkNeo4jRunning(t, releaseName)
	if err != nil {
		return err
	}

	err = EnterMaintenanceMode(t, releaseName)
	if !assert.NoError(t, err) {
		return err
	}

	err = checkNeo4jNotRunning(t, releaseName)
	if !assert.NoError(t, err) {
		return err
	}

	err = ExitMaintenanceMode(t, releaseName)
	if !assert.NoError(t, err) {
		return err
	}

	return err
}

func ExitMaintenanceMode(t *testing.T, releaseName *ReleaseName, extraArgs ...string) (error) {
	err := run(
		t, "helm", baseHelmCommand("upgrade", releaseName,
			append(extraArgs, "--set", "neo4j.offlineMaintenanceModeEnabled=false", "--wait", "--timeout", "300s")...
		)...
	)
	if !assert.NoError(t, err) {
		return err
	}

	err = run(t, "kubectl", "--namespace", string(releaseName.namespace()), "rollout", "status", "--watch", "--timeout=120s", "statefulset/"+string(*releaseName))
	if !assert.NoError(t, err) {
		return err
	}
	return err
}

func EnterMaintenanceMode(t *testing.T, releaseName *ReleaseName) error {
	err := run(t, "helm", baseHelmCommand("upgrade", releaseName, "--set", "neo4j.offlineMaintenanceModeEnabled=true")...)

	if !assert.NoError(t, err) {
		return err
	}

	time.Sleep(30 * time.Second)
	err = run(t, "kubectl", "--namespace", string(releaseName.namespace()), "wait", "--for=condition=Initialized", "pod/"+releaseName.podName())

	if !assert.NoError(t, err) {
		return err
	}
	time.Sleep(30 * time.Second)

	return err
}

func UninstallRelease(t *testing.T, releaseName *ReleaseName) error {
	return run(t, "helm", "uninstall", string(*releaseName), "--namespace", string(releaseName.namespace()))
}

func checkNeo4jNotRunning(t *testing.T, releaseName *ReleaseName) error {
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

func checkNeo4jRunning(t *testing.T, releaseName *ReleaseName) error {
	cmd := []string{
		"jps",
	}

	stdout, stderr, err := ExecInPod(releaseName, cmd)
	assert.Len(t, strings.Split(stdout, "\n"), 2)
	assert.Contains(t, stdout, "EntryPoint")
	assert.Empty(t, stderr)
	assert.NoError(t, err)

	return err
}

func TestMaintenanceInGCloudK8s(t *testing.T) {

	releaseName := ReleaseName("maintenance-"+TestRunIdentifier)
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
