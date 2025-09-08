package unit_tests

import (
	"strings"
	"testing"

	"github.com/neo4j/helm-charts/internal/model"
	"github.com/neo4j/helm-charts/internal/resources"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
)

// TestSecretMountsValidation tests that invalid secretMounts configurations fail validation
func TestSecretMountsValidation(t *testing.T) {
	t.Parallel()

	doTestCase := func(t *testing.T, chart model.Neo4jHelmChartBuilder, edition string) {
		// Test invalid secretMounts configuration should fail
		args := []string{"--set", "volumes.data.mode=defaultStorageClass", "--set", "disableLookups=true"}
		if edition == "enterprise" {
			args = append(args, "--set", "neo4j.acceptLicenseAgreement=eval")
		}

		_, err := model.HelmTemplateFromYamlFile(t, chart, resources.InvalidSecretMounts, args...)
		assert.Error(t, err, "Invalid secretMounts configuration should fail validation")

		// Check that error contains expected validation messages
		errorMsg := err.Error()
		assert.Contains(t, errorMsg, "secretMounts validation failed")
		assert.Contains(t, errorMsg, "secretName is required")
		assert.Contains(t, errorMsg, "mountPath is required")
		assert.Contains(t, errorMsg, "key is required")
	}

	forEachPrimaryChart(t, andEachSupportedEdition(doTestCase))
}

// TestSecretMountsGeneration tests that valid secretMounts configurations generate correct volumes and volume mounts
func TestSecretMountsGeneration(t *testing.T) {
	t.Parallel()

	doTestCase := func(t *testing.T, chart model.Neo4jHelmChartBuilder, edition string) {
		args := []string{"--set", "volumes.data.mode=defaultStorageClass", "--set", "disableLookups=true"}
		if edition == "enterprise" {
			args = append(args, "--set", "neo4j.acceptLicenseAgreement=eval")
		}

		manifest, err := model.HelmTemplateFromYamlFile(t, chart, resources.SecretMounts, args...)
		if !assert.NoError(t, err) {
			return
		}

		// Get the StatefulSet
		statefulSets := manifest.OfType(&appsv1.StatefulSet{})
		assert.Len(t, statefulSets, 1)

		statefulSet := statefulSets[0].(*appsv1.StatefulSet)
		neo4jContainer := &statefulSet.Spec.Template.Spec.Containers[0]

		// Verify volume mounts are present
		expectedVolumeMounts := map[string]string{
			"secret-mount-s3-credentials":   "/var/secrets/s3",
			"secret-mount-tls-certificates": "/var/secrets/tls",
		}

		for volumeName, mountPath := range expectedVolumeMounts {
			found := false
			for _, volumeMount := range neo4jContainer.VolumeMounts {
				if volumeMount.Name == volumeName {
					assert.Equal(t, mountPath, volumeMount.MountPath)
					assert.True(t, volumeMount.ReadOnly)
					found = true
					break
				}
			}
			assert.True(t, found, "Expected volume mount %s not found", volumeName)
		}

		// Verify volumes are present
		expectedVolumes := map[string]map[string]interface{}{
			"secret-mount-s3-credentials": {
				"secretName":  "my-s3-secret",
				"defaultMode": int32(384), // 0600 in decimal
				"itemsCount":  2,
			},
			"secret-mount-tls-certificates": {
				"secretName": "my-tls-certs",
				"itemsCount": 0, // No items specified
			},
		}

		for volumeName, expectedProps := range expectedVolumes {
			found := false
			for _, volume := range statefulSet.Spec.Template.Spec.Volumes {
				if volume.Name == volumeName {
					assert.NotNil(t, volume.Secret)
					assert.Equal(t, expectedProps["secretName"], volume.Secret.SecretName)

					if expectedMode, hasMode := expectedProps["defaultMode"]; hasMode {
						assert.Equal(t, expectedMode, *volume.Secret.DefaultMode)
					}

					expectedItemsCount := expectedProps["itemsCount"].(int)
					if expectedItemsCount > 0 {
						assert.Len(t, volume.Secret.Items, expectedItemsCount)
					} else {
						assert.Nil(t, volume.Secret.Items)
					}

					found = true
					break
				}
			}
			assert.True(t, found, "Expected volume %s not found", volumeName)
		}
	}

	forEachPrimaryChart(t, andEachSupportedEdition(doTestCase))
}

// TestEmptySecretMounts tests that empty secretMounts configuration works correctly
func TestEmptySecretMounts(t *testing.T) {
	t.Parallel()

	doTestCase := func(t *testing.T, chart model.Neo4jHelmChartBuilder, edition string) {
		args := []string{"--set", "volumes.data.mode=defaultStorageClass", "--set", "disableLookups=true"}
		if edition == "enterprise" {
			args = append(args, "--set", "neo4j.acceptLicenseAgreement=eval")
		}

		manifest, err := model.HelmTemplateFromYamlFile(t, chart, resources.EmptySecretMounts, args...)
		if !assert.NoError(t, err) {
			return
		}

		// Get the StatefulSet
		statefulSets := manifest.OfType(&appsv1.StatefulSet{})
		assert.Len(t, statefulSets, 1)

		statefulSet := statefulSets[0].(*appsv1.StatefulSet)
		neo4jContainer := &statefulSet.Spec.Template.Spec.Containers[0]

		// Verify no secret mount volumes are present
		for _, volumeMount := range neo4jContainer.VolumeMounts {
			assert.False(t, strings.HasPrefix(volumeMount.Name, "secret-mount-"),
				"No secret mount volumes should be present with empty secretMounts")
		}

		for _, volume := range statefulSet.Spec.Template.Spec.Volumes {
			assert.False(t, strings.HasPrefix(volume.Name, "secret-mount-"),
				"No secret mount volumes should be present with empty secretMounts")
		}
	}

	forEachPrimaryChart(t, andEachSupportedEdition(doTestCase))
}

// TestSeeduriS3SecretMounts tests the specific use case for seedURI S3 credentials
func TestSeeduriS3SecretMounts(t *testing.T) {
	t.Parallel()

	doTestCase := func(t *testing.T, chart model.Neo4jHelmChartBuilder, edition string) {
		// This test is specifically for enterprise edition (seedURI requires enterprise)
		if edition != "enterprise" {
			t.Skip("seedURI is only available in enterprise edition")
		}

		args := []string{"--set", "disableLookups=true"}
		manifest, err := model.HelmTemplateFromYamlFile(t, chart, resources.SeeduriS3SecretMounts, args...)
		if !assert.NoError(t, err) {
			return
		}

		// Get the StatefulSet
		statefulSets := manifest.OfType(&appsv1.StatefulSet{})
		assert.Len(t, statefulSets, 1)

		statefulSet := statefulSets[0].(*appsv1.StatefulSet)
		neo4jContainer := &statefulSet.Spec.Template.Spec.Containers[0]

		// Verify S3 credentials volume mount
		var s3VolumeMount *v1.VolumeMount
		for i, volumeMount := range neo4jContainer.VolumeMounts {
			if volumeMount.Name == "secret-mount-s3-credentials" {
				s3VolumeMount = &neo4jContainer.VolumeMounts[i]
				break
			}
		}

		assert.NotNil(t, s3VolumeMount, "S3 credentials volume mount should be present")
		assert.Equal(t, "/var/secrets/s3", s3VolumeMount.MountPath)
		assert.True(t, s3VolumeMount.ReadOnly)

		// Verify S3 credentials volume
		var s3Volume *v1.Volume
		for i, volume := range statefulSet.Spec.Template.Spec.Volumes {
			if volume.Name == "secret-mount-s3-credentials" {
				s3Volume = &statefulSet.Spec.Template.Spec.Volumes[i]
				break
			}
		}

		assert.NotNil(t, s3Volume, "S3 credentials volume should be present")
		assert.NotNil(t, s3Volume.Secret, "Volume should be a secret volume")
		assert.Equal(t, "cloudian-s3-credentials", s3Volume.Secret.SecretName)
		assert.Equal(t, int32(384), *s3Volume.Secret.DefaultMode) // 0600 in decimal
		assert.Len(t, s3Volume.Secret.Items, 4)                   // access-key-id, secret-access-key, endpoint, region

		// Verify individual secret items
		expectedItems := map[string]string{
			"access-key-id":     "access-key",
			"secret-access-key": "secret-key",
			"endpoint":          "endpoint",
			"region":            "region",
		}

		for _, item := range s3Volume.Secret.Items {
			expectedPath, exists := expectedItems[item.Key]
			assert.True(t, exists, "Unexpected secret item key: %s", item.Key)
			assert.Equal(t, expectedPath, item.Path)
		}
	}

	forEachPrimaryChart(t, andEachSupportedEdition(doTestCase))
}

// TestSecretMountsWithoutDisableLookups tests that validation fails when secrets don't exist (if lookups are enabled)
func TestSecretMountsWithoutDisableLookups(t *testing.T) {
	t.Parallel()

	doTestCase := func(t *testing.T, chart model.Neo4jHelmChartBuilder, edition string) {
		args := []string{"--set", "volumes.data.mode=defaultStorageClass"}
		if edition == "enterprise" {
			args = append(args, "--set", "neo4j.acceptLicenseAgreement=eval")
		}

		// This should fail because the secrets referenced in secretMounts don't exist
		_, err := model.HelmTemplateFromYamlFile(t, chart, resources.SecretMounts, args...)
		assert.Error(t, err, "Should fail when referenced secrets don't exist and lookups are enabled")

		errorMsg := err.Error()
		assert.Contains(t, errorMsg, "not found")
	}

	forEachPrimaryChart(t, andEachSupportedEdition(doTestCase))
}
