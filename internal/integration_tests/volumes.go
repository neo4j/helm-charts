package integration_tests

import (
	"fmt"
	"github.com/hashicorp/go-multierror"
	"github.com/neo4j/helm-charts/internal/model"
	"github.com/neo4j/helm-charts/internal/resources"
	"github.com/stretchr/testify/assert"
	"strings"
)
import "testing"

func volumesTests(name model.ReleaseName, chart model.Neo4jHelmChartBuilder) []SubTest {
	return []SubTest{
		{name: "Create Node", test: func(t *testing.T) { assert.NoError(t, createNode(t, name), "Create Node should succeed") }},
		{name: "Check Volumes", test: func(t *testing.T) { assert.NoError(t, checkVolumes(t, name), "Check volumes") }},
		{name: "Enter maintenance mode", test: func(t *testing.T) { assert.NoError(t, enterMaintenanceMode(t, name, chart), "Enter maintenance mode") }},
		{name: "Check Volumes", test: func(t *testing.T) { assert.NoError(t, checkVolumes(t, name), "Check volumes") }},
		{name: "Exit maintenance mode and install plugins", test: func(t *testing.T) {
			assert.NoError(t, exitMaintenanceMode(t, name, chart, resources.PluginsInitContainer.HelmArgs()...), "Exit maintenance mode and install plugins")
		}},
		{name: "Check Apoc", test: func(t *testing.T) { assert.NoError(t, checkApoc(t, name), "Check APOC") }},
	}
}

func checkApoc(t *testing.T, releaseName model.ReleaseName) error {
	results, err := runQuery(t, releaseName, "CALL apoc.help('apoc')", nil, false)
	if !assert.NoError(t, err) {
		return err
	}
	assert.Greater(t, len(results), 2, "Apoc help returned %s", results)
	return err
}

func checkVolume(t *testing.T, releaseName model.ReleaseName, volumePath string, sem chan error) {
	cmd := []string{"ls", "-1a", volumePath}

	stdout, stderr, err := ExecInPod(releaseName, cmd)
	assert.GreaterOrEqual(t, len(strings.Split(stdout, "\n")), 2, "Insufficient content in %s: %s", volumePath, stdout)
	assert.Empty(t, stderr)
	if !assert.NoError(t, err) {
		sem <- fmt.Errorf("Error checking volume %s", volumePath)
	} else {
		sem <- nil
	}

}

func checkVolumes(t *testing.T, releaseName model.ReleaseName) error {
	volumePathsThatShouldContainFiles := []string{
		"/logs",
		"/data",
		"/backups",
		"/metrics",
	}

	volumePathsThatShouldExist := append(
		volumePathsThatShouldContainFiles,
		"/licenses",
		"/import",
	)

	cmd := []string{"ls", "-1a", "/"}

	stdout, stderr, err := ExecInPod(releaseName, cmd)
	if !assert.NoError(t, err) {
		return err
	}
	assert.Empty(t, stderr)
	lsResult := strings.Split(stdout, "\n")
	for _, pathThatShouldExist := range volumePathsThatShouldExist {
		assert.Contains(t, lsResult, strings.TrimPrefix(pathThatShouldExist, "/"), "%s missing from root directory. ls result: %s", pathThatShouldExist, stdout)
	}

	// semaphore
	sem := make(chan error, len(volumePathsThatShouldContainFiles))

	for _, volumePath := range volumePathsThatShouldContainFiles {
		go checkVolume(t, releaseName, volumePath, sem)
	}

	for i := 0; i < len(volumePathsThatShouldContainFiles); i++ {
		errInGoRoutine := <-sem
		if errInGoRoutine != nil {
			err = multierror.Append(err, errInGoRoutine)
		}
	}

	return err
}
