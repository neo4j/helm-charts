package unit_tests

import (
	"github.com/neo4j/helm-charts/internal/model"
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

//TestClusterCoreInternalPorts checks if the internals services for cluster core contains the expected ports
func TestClusterCoreInternalPorts(t *testing.T) {
	t.Parallel()

	expectedPorts := map[int32]int32{
		6362: 6362,
		7687: 7687,
		7474: 7474,
		7473: 7473,
		7688: 7688,
		6000: 6000,
		5000: 5000,
		7000: 7000,
	}

	core := model.NewReleaseName("foo")

	readReplicaManifest, err := model.HelmTemplateForRelease(t, core, model.ClusterCoreHelmChart, useDataModeAndAcceptLicense)
	if !assert.NoError(t, err) {
		return
	}

	internalService := readReplicaManifest.OfTypeWithName(&v1.Service{}, core.InternalServiceName()).(*v1.Service)

	checkPortsMatchExpected(t, expectedPorts, internalService)
}

//TestReadReplicaServerGroups checks if the configMap data has an entry called causal_clustering.server_groups
// It also checks if the key causal_clustering.server_groups contains a value "cores" or not
func TestClusterCoreServerGroups(t *testing.T) {
	t.Parallel()

	clusterCore := model.NewReleaseName("foo")

	clusterCoreManifest, err := model.HelmTemplateForRelease(t, clusterCore, model.ClusterCoreHelmChart, useDataModeAndAcceptLicense)
	if !assert.NoError(t, err) {
		return
	}

	defaultConfigMap := clusterCoreManifest.OfTypeWithName(&v1.ConfigMap{}, clusterCore.DefaultConfigMapName()).(*v1.ConfigMap)
	assert.Contains(t, defaultConfigMap.Data, "causal_clustering.server_groups")
	assert.Contains(t, defaultConfigMap.Data["causal_clustering.server_groups"], "cores")
	assert.NotContains(t, defaultConfigMap.Data["causal_clustering.server_groups"], "read-replicas")
}

//TestClusterCorePanicOnShutDownConfig checks whether the dbms.panic.shutdown_on_panic attribut is set to the default value true or not
func TestClusterCorePanicOnShutDownConfig(t *testing.T) {
	t.Parallel()

	clusterCore := model.NewReleaseName("foo")

	clusterCoreManifest, err := model.HelmTemplateForRelease(t, clusterCore, model.ClusterCoreHelmChart, useDataModeAndAcceptLicense)
	if !assert.NoError(t, err) {
		return
	}

	defaultConfigMap := clusterCoreManifest.OfTypeWithName(&v1.ConfigMap{}, clusterCore.DefaultConfigMapName()).(*v1.ConfigMap)
	assert.Contains(t, defaultConfigMap.Data, "dbms.panic.shutdown_on_panic")
	assert.Contains(t, defaultConfigMap.Data["dbms.panic.shutdown_on_panic"], "true")
}

//TODO : This is to be enabled in 5.0
//TestClusterCoreDefaultLogFormat checks whether the dbms.logs.default_format value is set to JSON or not
//func TestClusterCoreDefaultLogFormat(t *testing.T) {
//	t.Parallel()
//
//	clusterCore := model.NewReleaseName("foo")
//
//	clusterCoreManifest, err := model.HelmTemplateForRelease(t, clusterCore, model.ClusterCoreHelmChart, useDataModeAndAcceptLicense)
//	if !assert.NoError(t, err) {
//		return
//	}
//
//	defaultConfigMap := clusterCoreManifest.OfTypeWithName(&v1.ConfigMap{}, clusterCore.DefaultConfigMapName()).(*v1.ConfigMap)
//	assert.Contains(t, defaultConfigMap.Data, "dbms.logs.default_format")
//	assert.Contains(t, defaultConfigMap.Data["dbms.logs.default_format"], "JSON")
//}
