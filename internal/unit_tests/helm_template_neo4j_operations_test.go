package unit_tests

import (
	"fmt"
	"testing"

	"github.com/neo4j/helm-charts/internal/model"
	"github.com/stretchr/testify/assert"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	v12 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
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

	operationsJob := manifest.OfTypeWithName(
		&batchv1.Job{},
		fmt.Sprintf("%s-operations", model.DefaultHelmTemplateReleaseName.String()),
	).(*batchv1.Job)
	assert.NotNil(t, operationsJob, "operations job not found")
	podSpec := operationsJob.Spec.Template.Spec
	assert.Equal(t, podSpec.RestartPolicy, v1.RestartPolicyNever)
	assert.Len(t, podSpec.Containers, 1)
	envVarNames := make(map[string]bool)
	for _, envVar := range podSpec.Containers[0].Env {
		envVarNames[envVar.Name] = true
	}

	// Check for required env variables
	requiredEnvVars := []string{"RELEASE_NAME", "NAMESPACE", "SECRETNAME", "PROTOCOL"}
	for _, required := range requiredEnvVars {
		assert.True(t, envVarNames[required], "Required environment variable %s not found", required)
	}

	for _, envVar := range podSpec.Containers[0].Env {
		switch envVar.Name {
		case "RELEASE_NAME", "NAMESPACE", "SECRETNAME", "PROTOCOL":
		case "SSL_DISABLE_HOSTNAME_VERIFICATION", "SSL_INSECURE_SKIP_VERIFY":
		default:
			t.Errorf("Unexpected environment variable: %s", envVar.Name)
		}
	}

	for _, envVar := range podSpec.Containers[0].Env {
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
	assert.Contains(t, operationsJob.ObjectMeta.Labels, "testkey")

	operationsRole := manifest.OfTypeWithName(
		&v12.Role{},
		fmt.Sprintf("%s-operations", model.DefaultHelmTemplateReleaseName.String()),
	).(*v12.Role)
	assert.NotNil(t, operationsRole, "operations role not found")
	assert.Len(t, operationsRole.Rules, 1)
	assert.Equal(t, operationsRole.Rules[0].Verbs, []string{"get"})
	assert.Equal(t, operationsRole.Rules[0].Resources, []string{"secrets"})
	assert.NotEmpty(t, operationsRole.Rules[0].ResourceNames, "operations role should have resourceNames for least-privilege")

	operationsServiceAccount := manifest.OfTypeWithName(
		&v1.ServiceAccount{},
		fmt.Sprintf("%s-operations", model.DefaultHelmTemplateReleaseName.String()),
	).(*v1.ServiceAccount)
	assert.NotNil(t, operationsServiceAccount, "operations serviceaccount not found")

	operationsRoleBinding := manifest.OfTypeWithName(
		&v12.RoleBinding{},
		fmt.Sprintf("%s-operations", model.DefaultHelmTemplateReleaseName.String()),
	).(*v12.RoleBinding)
	assert.NotNil(t, operationsRoleBinding, "operations role binding not found")
	assert.Equal(t, operationsRoleBinding.RoleRef.Name, operationsRole.Name)
	assert.Len(t, operationsRoleBinding.Subjects, 1)
	assert.Equal(t, operationsRoleBinding.Subjects[0].Kind, "ServiceAccount")
	assert.Equal(t, operationsRoleBinding.Subjects[0].Name, operationsServiceAccount.Name)

}

// TestNeo4jOperationsResourcesDefaultShape verifies the operations Job renders a
// valid resources block (requests/limits) when values.yaml supplies the flat
// `{cpu, memory}` shape. Guards against the schema error reported in issue #542
// where `resources.cpu` / `resources.memory` were emitted directly on the container.
func TestNeo4jOperationsResourcesDefaultShape(t *testing.T) {
	t.Parallel()

	clusterSize := 3
	helmValues := model.DefaultEnterpriseValues
	helmValues.Neo4J.MinimumClusterSize = clusterSize
	helmValues.Neo4J.Operations = model.Operations{
		EnableServer: true,
		Image:        "demo:123",
		Protocol:     "neo4j",
	}

	manifest, err := model.HelmTemplateFromStruct(t, model.HelmChart, helmValues)
	if !assert.NoError(t, err) {
		return
	}

	operationsJob := manifest.OfTypeWithName(
		&batchv1.Job{},
		fmt.Sprintf("%s-operations", model.DefaultHelmTemplateReleaseName.String()),
	).(*batchv1.Job)
	assert.NotNil(t, operationsJob, "operations job not found")

	resources := operationsJob.Spec.Template.Spec.Containers[0].Resources
	assert.Equal(t, resource.MustParse("100m"), resources.Requests[v1.ResourceCPU])
	assert.Equal(t, resource.MustParse("128Mi"), resources.Requests[v1.ResourceMemory])
	assert.Equal(t, resource.MustParse("100m"), resources.Limits[v1.ResourceCPU])
	assert.Equal(t, resource.MustParse("128Mi"), resources.Limits[v1.ResourceMemory])
}

// TestNeo4jOperationsResourcesRequestsLimits verifies the operations Job
// passes through explicit requests/limits when the values use the standard
// Kubernetes shape.
func TestNeo4jOperationsResourcesRequestsLimits(t *testing.T) {
	t.Parallel()

	clusterSize := 3
	helmValues := model.DefaultEnterpriseValues
	helmValues.Neo4J.MinimumClusterSize = clusterSize
	helmValues.Neo4J.Operations = model.Operations{
		EnableServer: true,
		Image:        "demo:123",
		Protocol:     "neo4j",
	}

	extraArgs := []string{
		"--set", "neo4j.operations.resources.requests.cpu=200m",
		"--set", "neo4j.operations.resources.requests.memory=256Mi",
		"--set", "neo4j.operations.resources.limits.cpu=500m",
		"--set", "neo4j.operations.resources.limits.memory=512Mi",
	}

	manifest, err := model.HelmTemplateFromStruct(t, model.HelmChart, helmValues, extraArgs...)
	if !assert.NoError(t, err) {
		return
	}

	operationsJob := manifest.OfTypeWithName(
		&batchv1.Job{},
		fmt.Sprintf("%s-operations", model.DefaultHelmTemplateReleaseName.String()),
	).(*batchv1.Job)
	assert.NotNil(t, operationsJob, "operations job not found")

	resources := operationsJob.Spec.Template.Spec.Containers[0].Resources
	assert.Equal(t, resource.MustParse("200m"), resources.Requests[v1.ResourceCPU])
	assert.Equal(t, resource.MustParse("256Mi"), resources.Requests[v1.ResourceMemory])
	assert.Equal(t, resource.MustParse("500m"), resources.Limits[v1.ResourceCPU])
	assert.Equal(t, resource.MustParse("512Mi"), resources.Limits[v1.ResourceMemory])
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

	operationsJob := manifest.OfTypeWithName(
		&batchv1.Job{},
		fmt.Sprintf("%s-operations", model.DefaultHelmTemplateReleaseName.String()),
	).(*batchv1.Job)
	assert.NotNil(t, operationsJob, "operations job not found")

	// Check for SSL environment variables
	envVars := make(map[string]string)
	for _, envVar := range operationsJob.Spec.Template.Spec.Containers[0].Env {
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

	operationsJob := manifest.OfTypeWithName(
		&batchv1.Job{},
		fmt.Sprintf("%s-operations", model.DefaultHelmTemplateReleaseName.String()),
	)
	assert.Nil(t, operationsJob, "operations job should not be present for standalone")

	operationsRole := manifest.OfTypeWithName(
		&v12.Role{},
		fmt.Sprintf("%s-operations", model.DefaultHelmTemplateReleaseName.String()),
	)
	assert.Nil(t, operationsRole, "operations role should not be present for standalone")

}

// TestNeo4jOperationsImagePullSecrets tests that imagePullSecrets are correctly set in the operations pod
func TestNeo4jOperationsImagePullSecrets(t *testing.T) {
	t.Parallel()

	clusterSize := 3
	helmValues := model.DefaultEnterpriseValues
	helmValues.DisableLookups = true
	helmValues.Neo4J.MinimumClusterSize = clusterSize
	helmValues.Image.ImagePullSecrets = []string{"my-pull-secret", "another-secret"}
	operations := model.Operations{
		EnableServer: true,
		Image:        "demo:123",
		Protocol:     "neo4j",
	}
	helmValues.Neo4J.Operations = operations

	manifest, err := model.HelmTemplateFromStruct(t, model.HelmChart, helmValues)
	if !assert.NoError(t, err) {
		return
	}

	operationsJob := manifest.OfTypeWithName(
		&batchv1.Job{},
		fmt.Sprintf("%s-operations", model.DefaultHelmTemplateReleaseName.String()),
	).(*batchv1.Job)
	assert.NotNil(t, operationsJob, "operations job not found")

	pullSecrets := operationsJob.Spec.Template.Spec.ImagePullSecrets
	assert.Len(t, pullSecrets, 2, "should have 2 imagePullSecrets")
	assert.Equal(t, "my-pull-secret", pullSecrets[0].Name)
	assert.Equal(t, "another-secret", pullSecrets[1].Name)
}

// TestNeo4jOperationsImagePullSecretsEmpty tests that empty imagePullSecrets are handled correctly
func TestNeo4jOperationsImagePullSecretsEmpty(t *testing.T) {
	t.Parallel()

	clusterSize := 3
	helmValues := model.DefaultEnterpriseValues
	helmValues.DisableLookups = true
	helmValues.Neo4J.MinimumClusterSize = clusterSize
	helmValues.Image.ImagePullSecrets = []string{}
	operations := model.Operations{
		EnableServer: true,
		Image:        "demo:123",
		Protocol:     "neo4j",
	}
	helmValues.Neo4J.Operations = operations

	manifest, err := model.HelmTemplateFromStruct(t, model.HelmChart, helmValues)
	if !assert.NoError(t, err) {
		return
	}

	operationsJob := manifest.OfTypeWithName(
		&batchv1.Job{},
		fmt.Sprintf("%s-operations", model.DefaultHelmTemplateReleaseName.String()),
	).(*batchv1.Job)
	assert.NotNil(t, operationsJob, "operations job not found")

	pullSecrets := operationsJob.Spec.Template.Spec.ImagePullSecrets
	assert.Nil(t, pullSecrets, "imagePullSecrets should be nil when empty")
}
