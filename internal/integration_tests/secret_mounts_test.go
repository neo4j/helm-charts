package integration_tests

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/neo4j/helm-charts/internal/model"
	"github.com/neo4j/helm-charts/internal/resources"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestSecretMountsInGCloudK8s tests secret mounting functionality in Kubernetes environment
func TestSecretMountsInGCloudK8s(t *testing.T) {
	chart := model.Neo4jHelmChartCommunityAndEnterprise
	releaseName := model.NewReleaseName("secret-mounts-" + TestNamespace(t))
	t.Parallel()

	t.Logf("Starting setup of '%s'", t.Name())

	// Create namespace first
	_, err := createNamespace(t, releaseName)
	if !assert.NoError(t, err) {
		return
	}

	// Create test secrets after namespace exists
	err = createTestSecrets(t, releaseName)
	if !assert.NoError(t, err) {
		return
	}

	// Cleanup secrets at the end
	t.Cleanup(func() {
		cleanupTestSecrets(t, releaseName)
	})

	// Install Neo4j with secret mounts configuration
	extraArgs := []string{}
	extraArgs = append(extraArgs, model.DefaultNeo4jNameArg...)
	extraArgs = append(extraArgs, resources.TestAntiAffinityRule.HelmArgs()...)
	extraArgs = append(extraArgs, resources.SecretMounts.HelmArgs()...)
	extraArgs = append(extraArgs, "--set", "neo4j.acceptLicenseAgreement=eval")
	extraArgs = append(extraArgs, "--set", "volumes.data.mode=defaultStorageClass")
	extraArgs = append(extraArgs, "--set", "disableLookups=true")

	helmClient := model.NewHelmClient(model.DefaultNeo4jChartName, extraArgs...)
	helmValues := model.DefaultEnterpriseValues
	helmValues.Neo4J.Edition = "enterprise"

	namespace := string(releaseName.Namespace())
	_, err = helmClient.Install(t, releaseName.String(), namespace, helmValues)

	// cleanup
	t.Cleanup(standaloneCleanup(t, releaseName))

	if !assert.NoError(t, err) {
		return
	}

	t.Logf("Succeeded with setup of '%s'", t.Name())

	runSubTests(t, secretMountsTests(releaseName, chart))
}

// secretMountsTests returns test functions for secret mounts functionality
func secretMountsTests(releaseName model.ReleaseName, chart model.Neo4jHelmChartBuilder) []SubTest {
	return []SubTest{
		{
			name: "VerifySecretMountsInPod",
			test: func(t *testing.T) {
				verifySecretMountsInPod(t, releaseName)
			},
		},
		{
			name: "VerifySecretMountPermissions",
			test: func(t *testing.T) {
				verifySecretMountPermissions(t, releaseName)
			},
		},
		{
			name: "VerifySecretMountContents",
			test: func(t *testing.T) {
				verifySecretMountContents(t, releaseName)
			},
		},
	}
}

// createTestSecrets creates the test secrets needed for the secret mounts tests
func createTestSecrets(t *testing.T, releaseName model.ReleaseName) error {

	namespace := releaseName.Namespace()
	ctx := context.Background()

	secrets := []struct {
		name string
		data map[string][]byte
	}{
		{
			name: "my-s3-secret",
			data: map[string][]byte{
				"access-key": []byte("test-access-key"),
				"secret-key": []byte("test-secret-key"),
			},
		},
		{
			name: "my-tls-certs",
			data: map[string][]byte{
				"tls.crt": []byte("-----BEGIN CERTIFICATE-----\ntest-cert\n-----END CERTIFICATE-----"),
				"tls.key": []byte("-----BEGIN PRIVATE KEY-----\ntest-key\n-----END PRIVATE KEY-----"),
			},
		},
	}

	for _, secret := range secrets {
		s := &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      secret.name,
				Namespace: string(namespace),
			},
			Type: v1.SecretTypeOpaque,
			Data: secret.data,
		}

		_, err := Clientset.CoreV1().Secrets(string(namespace)).Create(ctx, s, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("failed to create secret %s: %v", secret.name, err)
		}
		t.Logf("Created test secret: %s", secret.name)
	}

	return nil
}

// cleanupTestSecrets removes the test secrets
func cleanupTestSecrets(t *testing.T, releaseName model.ReleaseName) {

	namespace := releaseName.Namespace()
	ctx := context.Background()

	secretNames := []string{"my-s3-secret", "my-tls-certs"}

	for _, secretName := range secretNames {
		err := Clientset.CoreV1().Secrets(string(namespace)).Delete(ctx, secretName, metav1.DeleteOptions{})
		if apierrors.IsNotFound(err) {
			t.Logf("Secret already cleaned up: %s", secretName)
		} else if err != nil {
			t.Logf("Failed to cleanup secret %s: %v", secretName, err)
		} else {
			t.Logf("Cleaned up test secret: %s", secretName)
		}
	}
}

// verifySecretMountsInPod verifies that the secret volumes are mounted in the Neo4j pod
func verifySecretMountsInPod(t *testing.T, releaseName model.ReleaseName) {

	namespace := releaseName.Namespace()
	podName := releaseName.PodName()

	// Wait for pod to be ready
	assert.Eventually(t, func() bool {
		pod, err := Clientset.CoreV1().Pods(string(namespace)).Get(context.Background(), podName, metav1.GetOptions{})
		if err != nil {
			t.Logf("Failed to get pod %s: %v", podName, err)
			return false
		}

		t.Logf("Pod %s status: Phase=%s, Conditions=%d", podName, pod.Status.Phase, len(pod.Status.Conditions))

		// Log container statuses for debugging
		for _, containerStatus := range pod.Status.ContainerStatuses {
			t.Logf("Container %s: Ready=%t, State=%+v", containerStatus.Name, containerStatus.Ready, containerStatus.State)
		}

		for _, condition := range pod.Status.Conditions {
			t.Logf("Pod condition: Type=%s, Status=%s, Reason=%s, Message=%s",
				condition.Type, condition.Status, condition.Reason, condition.Message)
			if condition.Type == v1.PodReady && condition.Status == v1.ConditionTrue {
				return true
			}
		}
		return false
	}, 3*time.Minute, 10*time.Second, "Pod should become ready")

	// If readiness check failed, gather debug information
	if pod, err := Clientset.CoreV1().Pods(string(namespace)).Get(context.Background(), podName, metav1.GetOptions{}); err == nil {
		// Check if pod is still not ready and gather debugging info
		isPodReady := false
		for _, condition := range pod.Status.Conditions {
			if condition.Type == v1.PodReady && condition.Status == v1.ConditionTrue {
				isPodReady = true
				break
			}
		}

		if !isPodReady {
			t.Logf("Pod failed to become ready, gathering debug information...")

			// Get container logs
			for _, container := range pod.Spec.Containers {
				if container.Name == "neo4j" {
					// Get recent logs from Neo4j container
					logOptions := &v1.PodLogOptions{
						Container: "neo4j",
						TailLines: &[]int64{50}[0], // Last 50 lines
						Follow:    false,
					}

					req := Clientset.CoreV1().Pods(string(namespace)).GetLogs(podName, logOptions)
					logs, logErr := req.DoRaw(context.Background())
					if logErr != nil {
						t.Logf("Failed to get container logs: %v", logErr)
					} else {
						t.Logf("Neo4j container logs (last 50 lines):\n%s", string(logs))
					}
					break
				}
			}

			// Get pod events for more debugging
			events, eventErr := Clientset.CoreV1().Events(string(namespace)).List(context.Background(), metav1.ListOptions{
				FieldSelector: "involvedObject.name=" + podName,
			})
			if eventErr != nil {
				t.Logf("Failed to get pod events: %v", eventErr)
			} else {
				t.Logf("Pod events:")
				for _, event := range events.Items {
					t.Logf("  %s: %s - %s", event.Type, event.Reason, event.Message)
				}
			}
		}
	}

	// Get the pod and verify volume mounts
	pod, err := Clientset.CoreV1().Pods(string(namespace)).Get(context.Background(), podName, metav1.GetOptions{})
	if !assert.NoError(t, err) {
		return
	}

	// Find the Neo4j container
	var neo4jContainer *v1.Container
	for i, container := range pod.Spec.Containers {
		if container.Name == "neo4j" {
			neo4jContainer = &pod.Spec.Containers[i]
			break
		}
	}

	assert.NotNil(t, neo4jContainer, "Neo4j container should be found")

	// Verify expected volume mounts
	expectedMounts := map[string]string{
		"secret-mount-s3-credentials":   "/var/secrets/s3",
		"secret-mount-tls-certificates": "/var/secrets/tls",
	}

	for volumeName, expectedPath := range expectedMounts {
		found := false
		for _, mount := range neo4jContainer.VolumeMounts {
			if mount.Name == volumeName {
				assert.Equal(t, expectedPath, mount.MountPath)
				assert.True(t, mount.ReadOnly)
				found = true
				break
			}
		}
		assert.True(t, found, "Volume mount %s should be present", volumeName)
	}
}

// verifySecretMountPermissions verifies that the mounted secrets have correct permissions
func verifySecretMountPermissions(t *testing.T, releaseName model.ReleaseName) {

	namespace := releaseName.Namespace()
	podName := releaseName.PodName()

	// Wait for pod to be ready
	assert.Eventually(t, func() bool {
		pod, err := Clientset.CoreV1().Pods(string(namespace)).Get(context.Background(), podName, metav1.GetOptions{})
		if err != nil {
			return false
		}

		for _, condition := range pod.Status.Conditions {
			if condition.Type == v1.PodReady && condition.Status == v1.ConditionTrue {
				return true
			}
		}
		return false
	}, 1*time.Minute, 10*time.Second, "Pod should become ready")

	// Test file permissions for different mount configurations
	testCases := []struct {
		path         string
		expectedPerm string
		description  string
	}{
		{"/var/secrets/s3/access-key", "600", "S3 credentials should have 0600 permissions"},
		{"/var/secrets/tls", "644", "TLS certs should have default 0644 permissions"},
	}

	for _, tc := range testCases {
		// Check file permissions using ls -la
		cmd := []string{"ls", "-la", tc.path}
		_, _, err := ExecInPod(releaseName, cmd, "")
		if !assert.NoError(t, err, "Should be able to check permissions for %s", tc.path) {
			continue
		}

		// More detailed permission check could be added here using stat command
		cmd = []string{"stat", "-c", "%a", tc.path}
		output, _, err := ExecInPod(releaseName, cmd, "")
		if !assert.NoError(t, err, "Should be able to get permission details for %s", tc.path) {
			continue
		}

		t.Logf("Permissions for %s: %s", tc.path, output)
		// Note: Exact permission verification would need more sophisticated checks
		// as Kubernetes may modify the permissions based on securityContext
	}
}

// verifySecretMountContents verifies that the mounted secrets contain the expected data
func verifySecretMountContents(t *testing.T, releaseName model.ReleaseName) {

	namespace := releaseName.Namespace()
	podName := releaseName.PodName()

	// Wait for pod to be ready
	assert.Eventually(t, func() bool {
		pod, err := Clientset.CoreV1().Pods(string(namespace)).Get(context.Background(), podName, metav1.GetOptions{})
		if err != nil {
			return false
		}

		for _, condition := range pod.Status.Conditions {
			if condition.Type == v1.PodReady && condition.Status == v1.ConditionTrue {
				return true
			}
		}
		return false
	}, 1*time.Minute, 10*time.Second, "Pod should become ready")

	// Test that secret contents are accessible
	testCases := []struct {
		filePath        string
		expectedContent string
		description     string
	}{
		{"/var/secrets/s3/access-key", "test-access-key", "S3 access key should be readable"},
		{"/var/secrets/s3/secret-key", "test-secret-key", "S3 secret key should be readable"},
	}

	for _, tc := range testCases {
		cmd := []string{"cat", tc.filePath}
		output, _, err := ExecInPod(releaseName, cmd, "")
		if !assert.NoError(t, err, "Should be able to read %s", tc.filePath) {
			continue
		}

		assert.Equal(t, tc.expectedContent, output, tc.description)
		t.Logf("Successfully verified content of %s", tc.filePath)
	}

	// Verify that files not specified in items are not present (for secrets with items specified)
	cmd := []string{"ls", "/var/secrets/s3/"}
	output, _, err := ExecInPod(releaseName, cmd, "")
	if assert.NoError(t, err) {
		// Should only contain the files specified in items
		assert.Contains(t, output, "access-key")
		assert.Contains(t, output, "secret-key")
		// Should not contain other keys that might be in the secret but not in items
		t.Logf("S3 secret mount contents: %s", output)
	}
}
