package integration_tests

import (
	"github.com/neo4j/helm-charts/internal/model"
	"github.com/neo4j/helm-charts/internal/resources"
	"github.com/stretchr/testify/assert"
	"strings"
	"time"
)
import "testing"

func maintenanceTests(name model.ReleaseName, chart model.Neo4jHelmChart) []SubTest {
	return []SubTest{
		{name: "Create Node", test: func(t *testing.T) { assert.NoError(t, CreateNode(t, name), "Create Node should succeed") }},
		{name: "Maintenance Mode", test: func(t *testing.T) { assert.NoError(t, CheckMaintenanceMode(t, name, chart), "Check maintenance mode") }},
		{name: "Count Nodes", test: func(t *testing.T) { assert.NoError(t, CheckNodeCount(t, name), "Count Nodes should succeed") }},
	}
}

func CheckMaintenanceMode(t *testing.T, releaseName model.ReleaseName, chart model.Neo4jHelmChart) error {
	err := checkNeo4jRunning(t, releaseName)
	if err != nil {
		return err
	}

	err = EnterMaintenanceMode(t, releaseName, chart)
	if !assert.NoError(t, err) {
		return err
	}

	err = checkNeo4jNotRunning(t, releaseName)
	if !assert.NoError(t, err) {
		return err
	}

	err = ExitMaintenanceMode(t, releaseName, chart)
	if !assert.NoError(t, err) {
		return err
	}

	return err
}

func ExitMaintenanceMode(t *testing.T, releaseName model.ReleaseName, chart model.Neo4jHelmChart, extraArgs ...string) error {
	diskName := releaseName.DiskName()
	err := run(
		t, "helm", model.BaseHelmCommand("upgrade", releaseName, chart, model.Neo4jEdition, &diskName,
			append(extraArgs, "--set", "neo4j.offlineMaintenanceModeEnabled=false", "--wait", "--timeout", "300s")...,
		)...,
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

func EnterMaintenanceMode(t *testing.T, releaseName model.ReleaseName, chart model.Neo4jHelmChart) error {
	diskName := releaseName.DiskName()
	err := run(t, "helm", model.BaseHelmCommand("upgrade", releaseName, chart, model.Neo4jEdition, &diskName, "--set", "neo4j.offlineMaintenanceModeEnabled=true")...)

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

func UninstallRelease(t *testing.T, releaseName model.ReleaseName) error {
	return run(t, "helm", "uninstall", releaseName.String(), "--namespace", string(releaseName.Namespace()))
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

func checkNeo4jRunning(t *testing.T, releaseName model.ReleaseName) error {
	cmd := []string{
		"jps",
	}

	checkPod := func() (keepTrying bool, err error) {
		stdout, stderr, err := ExecInPod(releaseName, cmd)
		if err != nil {
			return false, err
		}

		if len(strings.Split(stdout, "\n")) == 0 {
			return true, nil
		}

		checksPass := assert.Len(t, strings.Split(stdout, "\n"), 2) &&
			assert.Contains(t, stdout, "EntryPoint") &&
			assert.Empty(t, stderr) &&
			assert.NoError(t, err)

		return !checksPass, err
	}

	timeout := time.After(1 * time.Minute)
	for {
		select {
		case <-timeout:
			_, err := checkPod()
			return err
		default:
			if keepGoing, err := checkPod(); !keepGoing {
				return err
			}
		}
	}
}

func TestMaintenanceInGCloudK8s(t *testing.T) {
	chart := model.StandaloneHelmChart
	releaseName := model.NewReleaseName("maintenance-" + TestRunIdentifier)
	t.Parallel()

	t.Logf("Starting setup of '%s'", t.Name())
	cleanup, err := installNeo4j(t, releaseName, chart, resources.TestAntiAffinityRule.HelmArgs()...)
	t.Cleanup(func() { cleanupTest(t, cleanup) })

	if !assert.NoError(t, err) {
		return
	}

	t.Logf("Succeeded with setup of '%s'", t.Name())

	runSubTests(t, maintenanceTests(releaseName, chart))
}
