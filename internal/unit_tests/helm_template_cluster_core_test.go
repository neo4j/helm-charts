package unit_tests

import (
	"github.com/neo-technology/neo4j-helm-charts/internal/model"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"testing"
)

func TestDefaultNeo4jNameClusterCoreAndLoadBalancer(t *testing.T) {
	t.Parallel()

	core1 := model.NewReleaseName("foo")
	core2 := model.NewReleaseName("bar")
	lb := model.NewReleaseName("lb")

	coreOneManifest, err := model.HelmTemplateForRelease(t, core1, model.ClusterCoreHelmChart, useDataModeAndAcceptLicense)
	if !assert.NoError(t, err) {
		return
	}
	coreTwoManifest, err := model.HelmTemplateForRelease(t, core2, model.ClusterCoreHelmChart, useDataModeAndAcceptLicense)
	if !assert.NoError(t, err) {
		return
	}
	lbManifest, err := model.HelmTemplateForRelease(t, lb, model.LoadBalancerHelmChart, nil)
	if !assert.NoError(t, err) {
		return
	}

	service := lbManifest.Only(t, &v1.Service{}).(*v1.Service)
	selector := service.Spec.Selector

	checkCoreManifestHasPodMatchingSelector(t, coreOneManifest, selector)
	checkCoreManifestHasPodMatchingSelector(t, coreTwoManifest, selector)

	assert.Contains(t, selector, "app")
	assert.Equal(t, selector["app"], "neo4j-cluster")
}

func checkCoreManifestHasPodMatchingSelector(t *testing.T, manifest *model.K8sResources, selector map[string]string) {
	for _, sts := range manifest.OfType(&appsv1.StatefulSet{}) {
		podLabels := sts.(*appsv1.StatefulSet).Spec.Template.Labels
		for key, expectedValue := range selector {
			assert.Contains(t, podLabels, key)
			assert.Equal(t, expectedValue, podLabels[key])
		}
	}
}
