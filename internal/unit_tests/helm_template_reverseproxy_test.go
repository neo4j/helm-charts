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
