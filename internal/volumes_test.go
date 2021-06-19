package internal

import (
	"fmt"
	"github.com/hashicorp/go-multierror"
	"github.com/stretchr/testify/assert"
	"strings"
)
import "testing"

func volumesTests(name *ReleaseName) []SubTest {
	return []SubTest{
		{name: "Create Node", test: func(t *testing.T) { assert.NoError(t, CreateNode(t, name), "Create Node should succeed") }},
		{name: "Check Volumes", test: func(t *testing.T) { assert.NoError(t, CheckVolumes(t, name), "Check volumes") }},
		{name: "Enter maintenance mode", test: func(t *testing.T) { assert.NoError(t, EnterMaintenanceMode(t, name), "Enter maintenance mode") }},
		{name: "Check Volumes", test: func(t *testing.T) { assert.NoError(t, CheckVolumes(t, name), "Check volumes") }},
	}
}

func checkVolume(t *testing.T, releaseName *ReleaseName, volumePath string, sem chan error) {
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

func CheckVolumes(t *testing.T, releaseName *ReleaseName) error {
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

func TestVolumesInGCloudK8s(t *testing.T) {
	releaseName := ReleaseName("volumes")
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

	runSubTests(t, volumesTests(&releaseName))
}
