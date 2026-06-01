package unit_tests

import (
	"fmt"
	"testing"

	"github.com/neo4j/helm-charts/internal/model"
	"github.com/stretchr/testify/assert"
	batchv1 "k8s.io/api/batch/v1"
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

	operationsJob := manifest.OfTypeWithName(
		&batchv1.Job{},
		fmt.Sprintf("%s-operations", model.DefaultHelmTemplateReleaseName.String()),
	).(*batchv1.Job)
	assert.NotNil(t, operationsJob, "operations job not found")
	podSpec := operationsJob.Spec.Template.Spec
	assert.Equal(t, podSpec.RestartPolicy, v1.RestartPolicyNever)
	assert.Len(t, podSpec.Containers, 1)
	assert.Len(t, podSpec.Containers[0].Env, 4)
	for _, envVar := range podSpec.Containers[0].Env {
		assert.Contains(t, []string{"RELEASE_NAME", "NAMESPACE", "SECRETNAME", "PROTOCOL"}, envVar.Name)
		switch envVar.Name {
		case "RELEASE_NAME":
			assert.Equal(t, envVar.Value, model.DefaultHelmTemplateReleaseName.String())
			continue
		case "NAMESPACE":
			assert.Equal(t, envVar.Value, string(model.DefaultHelmTemplateReleaseName.Namespace()))
			continue
		case "SECRETNAME":
			assert.Equal(t, envVar.Value, fmt.Sprintf("%s-auth", helmValues.Neo4J.Name))
			continue
		case "PROTOCOL":
			assert.Equal(t, envVar.Value, "neo4j")
			continue
		default:
			break
		}
	}
	assert.Contains(t, operationsJob.ObjectMeta.Labels, "testkey")
	assert.Equal(t, "neo4j-operations", operationsJob.ObjectMeta.Labels["app"])
	assert.Equal(t, helmValues.Neo4J.Name, operationsJob.ObjectMeta.Labels["helm.neo4j.com/neo4j.name"])
	assert.Equal(t, "true", operationsJob.ObjectMeta.Labels["helm.neo4j.com/clustering"])
	assert.Equal(t, model.DefaultHelmTemplateReleaseName.String(), operationsJob.ObjectMeta.Labels["helm.neo4j.com/instance"])

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
