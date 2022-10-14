package unit_tests

import (
	"github.com/neo4j/helm-charts/internal/model"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"testing"
)

// Tests the "default" behaviour that you get if you don't pass in *any* other values and the helm chart defaults are used
func TestDefaultCommunityHelmTemplate(t *testing.T) {
	t.Parallel()

	helmValues := model.HelmValues{
		Neo4J: model.Neo4J{
			Name: "test",
		},
		Volumes: model.Volumes{
			Data: model.Data{
				Mode: "selector",
			},
		},
	}
	manifest, err := model.HelmTemplateFromStruct(t, model.HelmChart, helmValues)
	if !assert.NoError(t, err) {
		return
	}
	checkNeo4jManifest(t, manifest, 2)

	neo4jStatefulSet := manifest.First(&appsv1.StatefulSet{}).(*appsv1.StatefulSet)
	neo4jStatefulSet.GetName()
	assert.NotEmpty(t, neo4jStatefulSet.Spec.Template.Spec.Containers)
	for _, container := range neo4jStatefulSet.Spec.Template.Spec.Containers {
		assert.NotContains(t, container.Image, "enterprise")
		assert.Equal(t, "1", container.Resources.Requests.Cpu().String())
		assert.Equal(t, "2Gi", container.Resources.Requests.Memory().String())
	}
	for _, container := range neo4jStatefulSet.Spec.Template.Spec.InitContainers {
		assert.NotContains(t, container.Image, "enterprise")
	}

	envConfigMap := manifest.OfTypeWithName(&v1.ConfigMap{}, model.DefaultHelmTemplateReleaseName.EnvConfigMapName()).(*v1.ConfigMap)
	assert.Equal(t, envConfigMap.Data["NEO4J_EDITION"], "COMMUNITY_K8S")
}

func TestExplicitCommunityHelmTemplate(t *testing.T) {
	t.Parallel()
	helmValues := model.DefaultCommunityValues
	manifest, err := model.HelmTemplateFromStruct(t, model.HelmChart, helmValues)
	if !assert.NoError(t, err) {
		return
	}

	checkNeo4jManifest(t, manifest, 2)

	neo4jStatefulSet := manifest.First(&appsv1.StatefulSet{}).(*appsv1.StatefulSet)
	neo4jStatefulSet.GetName()
	for _, container := range neo4jStatefulSet.Spec.Template.Spec.Containers {
		assert.NotContains(t, container.Image, "enterprise")
	}
	for _, container := range neo4jStatefulSet.Spec.Template.Spec.InitContainers {
		assert.NotContains(t, container.Image, "enterprise")
	}

	envConfigMap := manifest.OfTypeWithName(&v1.ConfigMap{}, model.DefaultHelmTemplateReleaseName.EnvConfigMapName()).(*v1.ConfigMap)
	assert.Equal(t, envConfigMap.Data["NEO4J_EDITION"], "COMMUNITY_K8S")
}
