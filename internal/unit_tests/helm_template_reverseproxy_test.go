package unit_tests

import (
	"fmt"
	"testing"

	"github.com/neo4j/helm-charts/internal/model"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/networking/v1"
)

// TestReverseProxyIngressWithAnnotations checks whether ingress has the provided annotations or not
func TestReverseProxyIngressWithAnnotations(t *testing.T) {
	t.Parallel()

	annotations := make(map[string]string, 2)
	annotations["demo1"] = "value1"
	annotations["demo2"] = "value2"
	helmValues := model.DefaultNeo4jReverseProxyValues
	helmValues.ReverseProxy.Ingress.Annotations = annotations

	manifests, err := model.HelmTemplateFromStruct(t, model.ReverseProxyHelmChart, helmValues)
	assert.NoError(t, err, "error seen while testing annotations with reverse proxy helm chart")
	ingressList := manifests.OfType(&v1.Ingress{})
	assert.Len(t, ingressList, 1, fmt.Sprintf("number of ingress should be 1 , not equal with %d", len(ingressList)))
	ingressAnnotations := ingressList[0].(*v1.Ingress).Annotations
	assert.Equal(t, ingressAnnotations, annotations, "ingress annotations are not matching")
}

// TestReverseProxyIngressWhenDisabled checks for no presence of ingress when disabled
func TestReverseProxyIngressWhenDisabled(t *testing.T) {
	t.Parallel()

	helmValues := model.DefaultNeo4jReverseProxyValues
	helmValues.ReverseProxy.Ingress.Enabled = false

	manifests, err := model.HelmTemplateFromStruct(t, model.ReverseProxyHelmChart, helmValues)
	assert.NoError(t, err, "error seen while testing ingress disabled with reverse proxy helm chart")
	ingressList := manifests.OfType(&v1.Ingress{})
	assert.Nil(t, ingressList, "ingress is not nil")
}

// TestReverseProxyIngressWhenTLSDisabled checks for no presence of tls configs when tls is disabled
func TestReverseProxyIngressWhenTLSDisabled(t *testing.T) {
	t.Parallel()

	helmValues := model.DefaultNeo4jReverseProxyValues
	helmValues.ReverseProxy.Ingress.TLS.Enabled = false

	manifests, err := model.HelmTemplateFromStruct(t, model.ReverseProxyHelmChart, helmValues)
	assert.NoError(t, err, "error seen while testing tls disabled with reverse proxy helm chart")
	ingressList := manifests.OfType(&v1.Ingress{})
	assert.Len(t, ingressList, 1, fmt.Sprintf("number of ingress should be 1 , not equal with %d", len(ingressList)))
	ingressTLS := ingressList[0].(*v1.Ingress).Spec.TLS
	assert.Nil(t, ingressTLS, "tls config is not nil")
}

// TestReverseProxyIngressWhenTLSEnabled checks for presence of tls configs when tls is enabled
func TestReverseProxyIngressWhenTLSEnabled(t *testing.T) {
	t.Parallel()

	helmValues := model.DefaultNeo4jReverseProxyValues
	secretName := "demo-secret"
	config := model.Config{
		SecretName: "demo-secret",
	}
	helmValues.ReverseProxy.Ingress.TLS.Config = []model.Config{config}
	manifests, err := model.HelmTemplateFromStruct(t, model.ReverseProxyHelmChart, helmValues)
	assert.NoError(t, err, "error seen while testing tls enabled with reverse proxy helm chart")
	ingressList := manifests.OfType(&v1.Ingress{})
	assert.Len(t, ingressList, 1, fmt.Sprintf("number of ingress should be 1 , not equal with %d", len(ingressList)))
	ingressTLS := ingressList[0].(*v1.Ingress).Spec.TLS
	assert.NotNil(t, ingressTLS, "tls config is nil")
	assert.Equal(t, ingressTLS[0].SecretName, secretName, fmt.Sprintf("TLS config secret name %s not matching with %s", secretName, ingressTLS[0].SecretName))
}

// TestReverseProxyIngressEmptyConfigWhenTLSEnabled checks when tls config is not present
func TestReverseProxyIngressEmptyConfigWhenTLSEnabled(t *testing.T) {
	t.Parallel()

	helmValues := model.DefaultNeo4jReverseProxyValues
	helmValues.ReverseProxy.Ingress.TLS.Config = nil
	_, err := model.HelmTemplateFromStruct(t, model.ReverseProxyHelmChart, helmValues)
	assert.Error(t, err, "no error found")
	assert.Contains(t, err.Error(), "Empty tls config")
}

// TestReverseProxyIngressEmptySecretName checks if error is seen when no secretname is provided
func TestReverseProxyIngressEmptySecretName(t *testing.T) {
	t.Parallel()

	helmValues := model.DefaultNeo4jReverseProxyValues
	config := model.Config{
		SecretName: "  ",
	}
	helmValues.ReverseProxy.Ingress.TLS.Config = []model.Config{config}
	_, err := model.HelmTemplateFromStruct(t, model.ReverseProxyHelmChart, helmValues)
	assert.Error(t, err, "no error found")
	assert.Contains(t, err.Error(), "Empty secretName for tls config")
}

// TestReverseProxyIngressHostName checks if hostname is populated in the ingress definition or not
func TestReverseProxyIngressHostName(t *testing.T) {
	t.Parallel()

	helmValues := model.DefaultNeo4jReverseProxyValues
	helmValues.ReverseProxy.Ingress.Host = "demo.com"
	manifests, err := model.HelmTemplateFromStruct(t, model.ReverseProxyHelmChart, helmValues)
	assert.NoError(t, err, fmt.Errorf("error seen while testing ingress hostname with reverse proxy helm chart \n %v \n helm values %v", err, helmValues))
	ingressList := manifests.OfType(&v1.Ingress{})
	assert.Len(t, ingressList, 1, fmt.Sprintf("number of ingress should be 1 , not equal with %d", len(ingressList)))
	ingressHostName := ingressList[0].(*v1.Ingress).Spec.Rules[0].Host
	assert.Equal(t, ingressHostName, "demo.com", "ingress hostname not matching")
}

// TestReverseProxyNamespaceParameter checks if namespace parameter is correctly passed to the deployment
func TestReverseProxyNamespaceParameter(t *testing.T) {
	t.Parallel()

	helmValues := model.DefaultNeo4jReverseProxyValues
	helmValues.ReverseProxy.Namespace = "neo4j-production"
	helmValues.ReverseProxy.ServiceName = "neo4j-service"
	manifests, err := model.HelmTemplateFromStruct(t, model.ReverseProxyHelmChart, helmValues)
	assert.NoError(t, err, "error seen while testing namespace parameter with reverse proxy helm chart")

	deploymentList := manifests.OfType(&appsv1.Deployment{})
	assert.Len(t, deploymentList, 1, fmt.Sprintf("number of deployments should be 1, not %d", len(deploymentList)))

	deployment := deploymentList[0].(*appsv1.Deployment)
	containers := deployment.Spec.Template.Spec.Containers
	assert.Len(t, containers, 1, "should have exactly one container")

	// Check if NAMESPACE env var is set to the custom namespace
	var namespaceEnv *corev1.EnvVar
	for _, env := range containers[0].Env {
		if env.Name == "NAMESPACE" {
			namespaceEnv = &env
			break
		}
	}

	assert.NotNil(t, namespaceEnv, "NAMESPACE environment variable should be set")
	assert.Equal(t, "neo4j-production", namespaceEnv.Value, "NAMESPACE should be set to the custom namespace")
}

// TestReverseProxyNamespaceDefaultsToReleaseNamespace checks backward compatibility when namespace is not specified
func TestReverseProxyNamespaceDefaultsToReleaseNamespace(t *testing.T) {
	t.Parallel()

	helmValues := model.DefaultNeo4jReverseProxyValues
	helmValues.ReverseProxy.Namespace = "" // explicitly empty to test default behavior
	helmValues.ReverseProxy.ServiceName = "neo4j-service"
	manifests, err := model.HelmTemplateFromStruct(t, model.ReverseProxyHelmChart, helmValues, "--namespace", "reverse-proxy-ns")
	assert.NoError(t, err, "error seen while testing default namespace with reverse proxy helm chart")

	deploymentList := manifests.OfType(&appsv1.Deployment{})
	assert.Len(t, deploymentList, 1, fmt.Sprintf("number of deployments should be 1, not %d", len(deploymentList)))

	deployment := deploymentList[0].(*appsv1.Deployment)
	containers := deployment.Spec.Template.Spec.Containers
	assert.Len(t, containers, 1, "should have exactly one container")

	// Check if NAMESPACE env var defaults to the release namespace
	var namespaceEnv *corev1.EnvVar
	for _, env := range containers[0].Env {
		if env.Name == "NAMESPACE" {
			namespaceEnv = &env
			break
		}
	}

	assert.NotNil(t, namespaceEnv, "NAMESPACE environment variable should be set")
	assert.Equal(t, "reverse-proxy-ns", namespaceEnv.Value, "NAMESPACE should default to the release namespace")
}

// TestReverseProxyPodLabels checks if pod labels are correctly set on the deployment pod template
func TestReverseProxyPodLabels(t *testing.T) {
	t.Parallel()

	podLabels := map[string]string{
		"app":         "reverse-proxy",
		"environment": "production",
		"team":        "platform",
	}
	helmValues := model.DefaultNeo4jReverseProxyValues
	helmValues.ReverseProxy.ServiceName = "neo4j-service"
	helmValues.ReverseProxy.PodLabels = podLabels

	manifests, err := model.HelmTemplateFromStruct(t, model.ReverseProxyHelmChart, helmValues)
	assert.NoError(t, err, "error seen while testing pod labels with reverse proxy helm chart")

	deploymentList := manifests.OfType(&appsv1.Deployment{})
	assert.Len(t, deploymentList, 1, fmt.Sprintf("number of deployments should be 1, not %d", len(deploymentList)))

	deployment := deploymentList[0].(*appsv1.Deployment)
	podTemplateLabels := deployment.Spec.Template.ObjectMeta.Labels

	// Check that all provided pod labels are present
	for key, expectedValue := range podLabels {
		assert.Contains(t, podTemplateLabels, key, fmt.Sprintf("pod label %s should be present", key))
		assert.Equal(t, expectedValue, podTemplateLabels[key], fmt.Sprintf("pod label %s value should match", key))
	}
}

// TestReverseProxyNodeSelector checks if nodeSelector is correctly set on the deployment pod spec
func TestReverseProxyNodeSelector(t *testing.T) {
	t.Parallel()

	nodeSelectorLabels := map[string]string{
		"label1": "value1",
		"label2": "value2",
	}
	helmValues := model.DefaultNeo4jReverseProxyValues
	helmValues.ReverseProxy.ServiceName = "neo4j-service"
	helmValues.ReverseProxy.NodeSelector = nodeSelectorLabels

	manifests, err := model.HelmTemplateFromStruct(t, model.ReverseProxyHelmChart, helmValues)
	assert.NoError(t, err, "error seen while testing nodeSelector with reverse proxy helm chart")

	deploymentList := manifests.OfType(&appsv1.Deployment{})
	assert.Len(t, deploymentList, 1, fmt.Sprintf("number of deployments should be 1, not %d", len(deploymentList)))

	deployment := deploymentList[0].(*appsv1.Deployment)
	nodeSelector := deployment.Spec.Template.Spec.NodeSelector

	assert.NotNil(t, nodeSelector, "nodeSelector should be present")
	assert.Equal(t, nodeSelectorLabels, nodeSelector, "nodeSelector labels should match")
}

// TestReverseProxyPodLabelsAndNodeSelectorTogether checks if both podLabels and nodeSelector work together
func TestReverseProxyPodLabelsAndNodeSelectorTogether(t *testing.T) {
	t.Parallel()

	podLabels := map[string]string{
		"app":         "reverse-proxy",
		"environment": "production",
	}
	nodeSelectorLabels := map[string]string{
		"workload-type": "reverse-proxy",
		"zone":          "us-east-1",
	}
	helmValues := model.DefaultNeo4jReverseProxyValues
	helmValues.ReverseProxy.ServiceName = "neo4j-service"
	helmValues.ReverseProxy.PodLabels = podLabels
	helmValues.ReverseProxy.NodeSelector = nodeSelectorLabels

	manifests, err := model.HelmTemplateFromStruct(t, model.ReverseProxyHelmChart, helmValues)
	assert.NoError(t, err, "error seen while testing podLabels and nodeSelector together with reverse proxy helm chart")

	deploymentList := manifests.OfType(&appsv1.Deployment{})
	assert.Len(t, deploymentList, 1, fmt.Sprintf("number of deployments should be 1, not %d", len(deploymentList)))

	deployment := deploymentList[0].(*appsv1.Deployment)

	// Check pod labels
	podTemplateLabels := deployment.Spec.Template.ObjectMeta.Labels
	for key, expectedValue := range podLabels {
		assert.Contains(t, podTemplateLabels, key, fmt.Sprintf("pod label %s should be present", key))
		assert.Equal(t, expectedValue, podTemplateLabels[key], fmt.Sprintf("pod label %s value should match", key))
	}

	// Check nodeSelector
	nodeSelector := deployment.Spec.Template.Spec.NodeSelector
	assert.NotNil(t, nodeSelector, "nodeSelector should be present")
	assert.Equal(t, nodeSelectorLabels, nodeSelector, "nodeSelector labels should match")
}

// TestReverseProxyEmptyPodLabelsAndNodeSelector checks that empty values don't break the template
func TestReverseProxyEmptyPodLabelsAndNodeSelector(t *testing.T) {
	t.Parallel()

	helmValues := model.DefaultNeo4jReverseProxyValues
	helmValues.ReverseProxy.ServiceName = "neo4j-service"
	helmValues.ReverseProxy.PodLabels = map[string]string{}
	helmValues.ReverseProxy.NodeSelector = map[string]string{}

	manifests, err := model.HelmTemplateFromStruct(t, model.ReverseProxyHelmChart, helmValues)
	assert.NoError(t, err, "error seen while testing empty podLabels and nodeSelector with reverse proxy helm chart")

	deploymentList := manifests.OfType(&appsv1.Deployment{})
	assert.Len(t, deploymentList, 1, fmt.Sprintf("number of deployments should be 1, not %d", len(deploymentList)))

	deployment := deploymentList[0].(*appsv1.Deployment)

	// Pod should still have the default label
	podTemplateLabels := deployment.Spec.Template.ObjectMeta.Labels
	assert.Contains(t, podTemplateLabels, "name", "pod should have default name label")

	// NodeSelector should not be present when empty
	nodeSelector := deployment.Spec.Template.Spec.NodeSelector
	assert.Nil(t, nodeSelector, "nodeSelector should not be present when empty")
}

// TestReverseProxyTolerations checks if tolerations are correctly set on the deployment pod spec
func TestReverseProxyTolerations(t *testing.T) {
	t.Parallel()

	dummyToleration := model.Toleration{
		Key:      "dedicated",
		Operator: "Equal",
		Value:    "reverse-proxy",
		Effect:   "NoSchedule",
	}
	helmValues := model.DefaultNeo4jReverseProxyValues
	helmValues.ReverseProxy.ServiceName = "neo4j-service"
	helmValues.ReverseProxy.Tolerations = []model.Toleration{dummyToleration}

	manifests, err := model.HelmTemplateFromStruct(t, model.ReverseProxyHelmChart, helmValues)
	assert.NoError(t, err, "error seen while testing tolerations with reverse proxy helm chart")

	deploymentList := manifests.OfType(&appsv1.Deployment{})
	assert.Len(t, deploymentList, 1, fmt.Sprintf("number of deployments should be 1, not %d", len(deploymentList)))

	deployment := deploymentList[0].(*appsv1.Deployment)
	tolerations := deployment.Spec.Template.Spec.Tolerations
	assert.Len(t, tolerations, 1, "should have exactly one toleration")
	assert.Equal(t, dummyToleration.Key, tolerations[0].Key, "toleration key should match")
	assert.Equal(t, dummyToleration.Operator, string(tolerations[0].Operator), "toleration operator should match")
	assert.Equal(t, dummyToleration.Value, tolerations[0].Value, "toleration value should match")
	assert.Equal(t, dummyToleration.Effect, string(tolerations[0].Effect), "toleration effect should match")
}

// TestReverseProxyEmptyTolerations checks that empty tolerations don't break the template
func TestReverseProxyEmptyTolerations(t *testing.T) {
	t.Parallel()

	helmValues := model.DefaultNeo4jReverseProxyValues
	helmValues.ReverseProxy.ServiceName = "neo4j-service"
	helmValues.ReverseProxy.Tolerations = []model.Toleration{}

	manifests, err := model.HelmTemplateFromStruct(t, model.ReverseProxyHelmChart, helmValues)
	assert.NoError(t, err, "error seen while testing empty tolerations with reverse proxy helm chart")

	deploymentList := manifests.OfType(&appsv1.Deployment{})
	assert.Len(t, deploymentList, 1, fmt.Sprintf("number of deployments should be 1, not %d", len(deploymentList)))

	deployment := deploymentList[0].(*appsv1.Deployment)
	tolerations := deployment.Spec.Template.Spec.Tolerations
	assert.Nil(t, tolerations, "tolerations should not be present when empty")
}

// TestReverseProxyTolerationsWithNodeSelector checks if both tolerations and nodeSelector work together
func TestReverseProxyTolerationsWithNodeSelector(t *testing.T) {
	t.Parallel()

	dummyToleration := model.Toleration{
		Key:      "dedicated",
		Operator: "Equal",
		Value:    "reverse-proxy",
		Effect:   "NoSchedule",
	}
	nodeSelectorLabels := map[string]string{
		"workload-type": "reverse-proxy",
	}
	helmValues := model.DefaultNeo4jReverseProxyValues
	helmValues.ReverseProxy.ServiceName = "neo4j-service"
	helmValues.ReverseProxy.Tolerations = []model.Toleration{dummyToleration}
	helmValues.ReverseProxy.NodeSelector = nodeSelectorLabels

	manifests, err := model.HelmTemplateFromStruct(t, model.ReverseProxyHelmChart, helmValues)
	assert.NoError(t, err, "error seen while testing tolerations with nodeSelector on reverse proxy helm chart")

	deploymentList := manifests.OfType(&appsv1.Deployment{})
	assert.Len(t, deploymentList, 1, fmt.Sprintf("number of deployments should be 1, not %d", len(deploymentList)))

	deployment := deploymentList[0].(*appsv1.Deployment)
	tolerations := deployment.Spec.Template.Spec.Tolerations
	assert.Len(t, tolerations, 1, "should have exactly one toleration")

	nodeSelector := deployment.Spec.Template.Spec.NodeSelector
	assert.NotNil(t, nodeSelector, "nodeSelector should be present")
	assert.Equal(t, nodeSelectorLabels, nodeSelector, "nodeSelector labels should match")
}

// TestReverseProxyImagePullSecrets tests that imagePullSecrets are correctly set in the reverse proxy deployment
func TestReverseProxyImagePullSecrets(t *testing.T) {
	t.Parallel()

	helmValues := model.DefaultNeo4jReverseProxyValues
	helmValues.ReverseProxy.ImagePullSecrets = []string{"my-pull-secret", "another-secret"}
	helmValues.ReverseProxy.ServiceName = "neo4j-service"

	manifests, err := model.HelmTemplateFromStruct(t, model.ReverseProxyHelmChart, helmValues)
	assert.NoError(t, err, "error seen while testing imagePullSecrets with reverse proxy helm chart")

	deploymentList := manifests.OfType(&appsv1.Deployment{})
	assert.Len(t, deploymentList, 1, fmt.Sprintf("number of deployments should be 1, not %d", len(deploymentList)))

	deployment := deploymentList[0].(*appsv1.Deployment)
	pullSecrets := deployment.Spec.Template.Spec.ImagePullSecrets
	assert.Len(t, pullSecrets, 2, "should have 2 imagePullSecrets")
	assert.Equal(t, "my-pull-secret", pullSecrets[0].Name)
	assert.Equal(t, "another-secret", pullSecrets[1].Name)
}

// TestReverseProxyImagePullSecretsEmpty tests that empty imagePullSecrets are handled correctly
func TestReverseProxyImagePullSecretsEmpty(t *testing.T) {
	t.Parallel()

	helmValues := model.DefaultNeo4jReverseProxyValues
	helmValues.ReverseProxy.ImagePullSecrets = []string{}
	helmValues.ReverseProxy.ServiceName = "neo4j-service"

	manifests, err := model.HelmTemplateFromStruct(t, model.ReverseProxyHelmChart, helmValues)
	assert.NoError(t, err, "error seen while testing empty imagePullSecrets with reverse proxy helm chart")

	deploymentList := manifests.OfType(&appsv1.Deployment{})
	assert.Len(t, deploymentList, 1, fmt.Sprintf("number of deployments should be 1, not %d", len(deploymentList)))

	deployment := deploymentList[0].(*appsv1.Deployment)
	pullSecrets := deployment.Spec.Template.Spec.ImagePullSecrets
	assert.Nil(t, pullSecrets, "imagePullSecrets should be nil when empty")
}

// TestReverseProxyExtraEnvVars tests that extraEnvVars are correctly injected into the reverse proxy container
func TestReverseProxyExtraEnvVars(t *testing.T) {
	t.Parallel()

	helmValues := model.DefaultNeo4jReverseProxyValues
	helmValues.ReverseProxy.ServiceName = "neo4j-service"
	helmValues.ExtraEnvVars = []map[string]interface{}{
		{"name": "CUSTOM_VAR_1", "value": "value1"},
		{"name": "CUSTOM_VAR_2", "value": "value2"},
	}

	manifests, err := model.HelmTemplateFromStruct(t, model.ReverseProxyHelmChart, helmValues)
	assert.NoError(t, err, "error seen while testing extraEnvVars with reverse proxy helm chart")

	deploymentList := manifests.OfType(&appsv1.Deployment{})
	assert.Len(t, deploymentList, 1, fmt.Sprintf("number of deployments should be 1, not %d", len(deploymentList)))

	deployment := deploymentList[0].(*appsv1.Deployment)
	container := deployment.Spec.Template.Spec.Containers[0]
	envVariables := container.Env

	assert.Contains(t, envVariables, corev1.EnvVar{Name: "CUSTOM_VAR_1", Value: "value1"},
		"CUSTOM_VAR_1 should be present in container env vars")
	assert.Contains(t, envVariables, corev1.EnvVar{Name: "CUSTOM_VAR_2", Value: "value2"},
		"CUSTOM_VAR_2 should be present in container env vars")
}

// TestReverseProxyExtraVolumesAndMounts tests that extraVolumes and extraVolumeMounts are correctly applied
func TestReverseProxyExtraVolumesAndMounts(t *testing.T) {
	t.Parallel()

	helmValues := model.DefaultNeo4jReverseProxyValues
	helmValues.ReverseProxy.ServiceName = "neo4j-service"
	helmValues.ExtraVolumes = []map[string]interface{}{
		{
			"name": "custom-config",
			"configMap": map[string]interface{}{
				"name": "my-configmap",
			},
		},
	}
	helmValues.ExtraVolumeMounts = []map[string]interface{}{
		{
			"name":      "custom-config",
			"mountPath": "/config",
			"readOnly":  true,
		},
	}

	manifests, err := model.HelmTemplateFromStruct(t, model.ReverseProxyHelmChart, helmValues)
	assert.NoError(t, err, "error seen while testing extraVolumes and extraVolumeMounts with reverse proxy helm chart")

	deploymentList := manifests.OfType(&appsv1.Deployment{})
	assert.Len(t, deploymentList, 1, fmt.Sprintf("number of deployments should be 1, not %d", len(deploymentList)))

	deployment := deploymentList[0].(*appsv1.Deployment)
	container := deployment.Spec.Template.Spec.Containers[0]

	var foundMount bool
	for _, vm := range container.VolumeMounts {
		if vm.Name == "custom-config" {
			foundMount = true
			assert.Equal(t, "/config", vm.MountPath)
			assert.True(t, vm.ReadOnly)
			break
		}
	}
	assert.True(t, foundMount, "custom-config volume mount should be present")

	volumes := deployment.Spec.Template.Spec.Volumes
	var foundVolume bool
	for _, v := range volumes {
		if v.Name == "custom-config" {
			foundVolume = true
			assert.NotNil(t, v.ConfigMap, "custom-config volume should be a configMap volume")
			assert.Equal(t, "my-configmap", v.ConfigMap.Name)
			break
		}
	}
	assert.True(t, foundVolume, "custom-config volume should be present")
}

// TestReverseProxyNoExtraEnvVars tests that the template works correctly when no extras are specified
func TestReverseProxyNoExtraEnvVars(t *testing.T) {
	t.Parallel()

	helmValues := model.DefaultNeo4jReverseProxyValues
	helmValues.ReverseProxy.ServiceName = "neo4j-service"

	manifests, err := model.HelmTemplateFromStruct(t, model.ReverseProxyHelmChart, helmValues)
	assert.NoError(t, err, "error seen while testing reverse proxy without extras")

	deploymentList := manifests.OfType(&appsv1.Deployment{})
	assert.Len(t, deploymentList, 1)

	deployment := deploymentList[0].(*appsv1.Deployment)
	container := deployment.Spec.Template.Spec.Containers[0]

	assert.Nil(t, container.VolumeMounts, "volumeMounts should be nil when no extras specified")
	assert.Nil(t, deployment.Spec.Template.Spec.Volumes, "volumes should be nil when no extras specified")
}

func TestReverseProxyResources(t *testing.T) {
	t.Parallel()

	helmValues := model.DefaultNeo4jReverseProxyValues
	helmValues.ReverseProxy.ServiceName = "neo4j-service"
	helmValues.ReverseProxy.Resources = model.ReverseProxyResources{
		Requests: model.ReverseProxyResourceValues{
			CPU:    "200m",
			Memory: "256Mi",
		},
		Limits: model.ReverseProxyResourceValues{
			CPU:    "1000m",
			Memory: "512Mi",
		},
	}

	manifests, err := model.HelmTemplateFromStruct(t, model.ReverseProxyHelmChart, helmValues)
	assert.NoError(t, err, "error seen while testing resources with reverse proxy helm chart")

	deploymentList := manifests.OfType(&appsv1.Deployment{})
	assert.Len(t, deploymentList, 1, fmt.Sprintf("number of deployments should be 1, not %d", len(deploymentList)))

	deployment := deploymentList[0].(*appsv1.Deployment)
	containers := deployment.Spec.Template.Spec.Containers
	assert.Len(t, containers, 1, "should have exactly one container")

	resources := containers[0].Resources
	assert.Equal(t, "200m", resources.Requests.Cpu().String(), "CPU request should match")
	assert.Equal(t, "256Mi", resources.Requests.Memory().String(), "Memory request should match")
	assert.Equal(t, "1", resources.Limits.Cpu().String(), "CPU limit should match")
	assert.Equal(t, "512Mi", resources.Limits.Memory().String(), "Memory limit should match")
}

func TestReverseProxyDefaultResources(t *testing.T) {
	t.Parallel()

	helmValues := model.DefaultNeo4jReverseProxyValues
	helmValues.ReverseProxy.ServiceName = "neo4j-service"

	manifests, err := model.HelmTemplateFromStruct(t, model.ReverseProxyHelmChart, helmValues)
	assert.NoError(t, err, "error seen while testing default resources with reverse proxy helm chart")

	deploymentList := manifests.OfType(&appsv1.Deployment{})
	assert.Len(t, deploymentList, 1, fmt.Sprintf("number of deployments should be 1, not %d", len(deploymentList)))

	deployment := deploymentList[0].(*appsv1.Deployment)
	containers := deployment.Spec.Template.Spec.Containers
	assert.Len(t, containers, 1, "should have exactly one container")

	resources := containers[0].Resources
	assert.False(t, resources.Requests.Cpu().IsZero(), "default CPU request should not be zero")
	assert.False(t, resources.Requests.Memory().IsZero(), "default memory request should not be zero")
	assert.False(t, resources.Limits.Cpu().IsZero(), "default CPU limit should not be zero")
	assert.False(t, resources.Limits.Memory().IsZero(), "default memory limit should not be zero")
}
