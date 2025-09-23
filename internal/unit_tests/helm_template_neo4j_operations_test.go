package unit_tests

import (
	"fmt"
	"testing"

	"github.com/neo4j/helm-charts/internal/model"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	v12 "k8s.io/api/rbac/v1"
)

// TestNeo4jOperationsEnableServer tests for the neo4j.operations.enableServer flag
func TestNeo4jOperationsEnableServer(t *testing.T) {
	t.Parallel()

	clusterSize := 3
	helmValues := model.DefaultEnterpriseValues
	helmValues.Neo4J.MinimumClusterSize = clusterSize
	operations := model.Operations{
		EnableServer: true,
		Image:        "demo:123",
		Protocol:     "neo4j",
		Labels: map[string]string{
			"testkey": "demo",
		},
	}
	helmValues.Neo4J.Operations = operations

	manifest, err := model.HelmTemplateFromStruct(t, model.HelmChart, helmValues)
	if !assert.NoError(t, err) {
		return
	}

	operationsPod := manifest.OfTypeWithName(
		&v1.Pod{},
		fmt.Sprintf("%s-operations", model.DefaultHelmTemplateReleaseName.String()),
	).(*v1.Pod)
	assert.NotNil(t, operationsPod, "operations pod not found")
	assert.Equal(t, operationsPod.Spec.RestartPolicy, v1.RestartPolicyNever)
	assert.Len(t, operationsPod.Spec.Containers, 1)
	envVarNames := make(map[string]bool)
	for _, envVar := range operationsPod.Spec.Containers[0].Env {
		envVarNames[envVar.Name] = true
	}

	// Check for required env variables
	requiredEnvVars := []string{"RELEASE_NAME", "NAMESPACE", "SECRETNAME", "PROTOCOL"}
	for _, required := range requiredEnvVars {
		assert.True(t, envVarNames[required], "Required environment variable %s not found", required)
	}

	for _, envVar := range operationsPod.Spec.Containers[0].Env {
		switch envVar.Name {
		case "RELEASE_NAME", "NAMESPACE", "SECRETNAME", "PROTOCOL":
		case "SSL_DISABLE_HOSTNAME_VERIFICATION", "SSL_INSECURE_SKIP_VERIFY":
		default:
			t.Errorf("Unexpected environment variable: %s", envVar.Name)
		}
	}

	for _, envVar := range operationsPod.Spec.Containers[0].Env {
		switch envVar.Name {
		case "RELEASE_NAME":
			assert.Equal(t, envVar.Value, model.DefaultHelmTemplateReleaseName.String())
		case "NAMESPACE":
			assert.Equal(t, envVar.Value, string(model.DefaultHelmTemplateReleaseName.Namespace()))
		case "SECRETNAME":
			assert.Equal(t, envVar.Value, fmt.Sprintf("%s-auth", helmValues.Neo4J.Name))
		case "PROTOCOL":
			assert.Equal(t, envVar.Value, "neo4j")
		}
	}
	assert.Contains(t, operationsPod.ObjectMeta.Labels, "testkey")

	operationsRole := manifest.OfTypeWithName(
		&v12.Role{},
		fmt.Sprintf("%s-secrets-reader", model.DefaultHelmTemplateReleaseName.String()),
	).(*v12.Role)
	assert.NotNil(t, operationsRole, "operations role not found")
	assert.Len(t, operationsRole.Rules, 1)
	assert.Equal(t, operationsRole.Rules[0].Verbs, []string{"get", "watch", "list"})
	assert.Equal(t, operationsRole.Rules[0].Resources, []string{"secrets"})

	serviceAccount := manifest.OfTypeWithName(
		&v1.ServiceAccount{},
		model.DefaultHelmTemplateReleaseName.String(),
	).(*v1.ServiceAccount)
	assert.NotNil(t, serviceAccount, "serviceaccount not found")

	operationsRoleBinding := manifest.OfTypeWithName(
		&v12.RoleBinding{},
		fmt.Sprintf("%s-secrets-binding", model.DefaultHelmTemplateReleaseName.String()),
	).(*v12.RoleBinding)
	assert.NotNil(t, operationsRoleBinding, "operations role binding not found")
	assert.Equal(t, operationsRoleBinding.RoleRef.Name, operationsRole.Name)
	assert.Len(t, operationsRoleBinding.Subjects, 1)
	assert.Equal(t, operationsRoleBinding.Subjects[0].Kind, "ServiceAccount")
	assert.Equal(t, operationsRoleBinding.Subjects[0].Name, serviceAccount.Name)

}

// TestNeo4jOperationsWithSSLConfiguration tests SSL configuration for operations pod
func TestNeo4jOperationsWithSSLConfiguration(t *testing.T) {
	t.Parallel()

	clusterSize := 3
	helmValues := model.DefaultEnterpriseValues
	helmValues.Neo4J.MinimumClusterSize = clusterSize

	// Configure operations with SSL settings
	operations := model.Operations{
		EnableServer: true,
		Image:        "demo:123",
		Protocol:     "neo4j+s",
		SSL: &model.OperationsSSL{
			DisableHostnameVerification: true,
			InsecureSkipVerify:          false,
		},
	}
	helmValues.Neo4J.Operations = operations

	manifest, err := model.HelmTemplateFromStruct(t, model.HelmChart, helmValues)
	if !assert.NoError(t, err) {
		return
	}

	operationsPod := manifest.OfTypeWithName(
		&v1.Pod{},
		fmt.Sprintf("%s-operations", model.DefaultHelmTemplateReleaseName.String()),
	).(*v1.Pod)
	assert.NotNil(t, operationsPod, "operations pod not found")

	// Check for SSL environment variables
	envVars := make(map[string]string)
	for _, envVar := range operationsPod.Spec.Containers[0].Env {
		envVars[envVar.Name] = envVar.Value
	}

	assert.Equal(t, "true", envVars["SSL_DISABLE_HOSTNAME_VERIFICATION"])
	assert.Equal(t, "false", envVars["SSL_INSECURE_SKIP_VERIFY"])
	assert.Equal(t, "neo4j+s", envVars["PROTOCOL"])
}

// TestNeo4jOperationsEnableServerForStandalone tests for the neo4j.operations.enableServer flag is enabled for standalone
// EnableServer works only for clusters, not required for standalone
func TestNeo4jOperationsEnableServerForStandalone(t *testing.T) {
	t.Parallel()

	helmValues := model.DefaultEnterpriseValues
	operations := model.Operations{
		EnableServer: true,
		Image:        "demo:123",
	}
	helmValues.Neo4J.Operations = operations

	manifest, err := model.HelmTemplateFromStruct(t, model.HelmChart, helmValues)
	if !assert.NoError(t, err) {
		return
	}

	operationsPod := manifest.OfTypeWithName(
		&v1.Pod{},
		fmt.Sprintf("%s-operations", model.DefaultHelmTemplateReleaseName.String()),
	)
	assert.Nil(t, operationsPod, "operations pod should not be present for standalone")

	operationsRole := manifest.OfTypeWithName(
		&v12.Role{},
		fmt.Sprintf("%s-secrets-reader", model.DefaultHelmTemplateReleaseName.String()),
	)
	assert.Nil(t, operationsRole, "operations role should not be present for standalone")

}
