package unit_tests

import (
	"fmt"
	"github.com/neo4j/helm-charts/internal/model"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/networking/v1"
	"testing"
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
	helmValues.ReverseProxy.Ingress.TLS.Config[0].SecretName = secretName
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
	assert.Contains(t, err.Error(), "Empty secretName for tls config")
}

// TestReverseProxyIngressEmptySecretName checks if error is seen when no secretname is provided
func TestReverseProxyIngressEmptySecretName(t *testing.T) {
	t.Parallel()

	helmValues := model.DefaultNeo4jReverseProxyValues
	helmValues.ReverseProxy.Ingress.TLS.Config[0].SecretName = "   "
	_, err := model.HelmTemplateFromStruct(t, model.ReverseProxyHelmChart, helmValues)
	assert.Error(t, err, "no error found")
	assert.Contains(t, err.Error(), "Empty secretName for tls config")
}
