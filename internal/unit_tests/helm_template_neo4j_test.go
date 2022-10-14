package unit_tests

import (
	"github.com/neo4j/helm-charts/internal/model"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"testing"
)

func TestNeo4jNameWithLoadBalancerSelector(t *testing.T) {
	t.Parallel()

	core1 := model.NewReleaseName("foo")
	args := append(useEnterprise, useDataModeAndAcceptLicense...)
	coreOneManifest, err := model.HelmTemplateForRelease(t, core1, model.HelmChart, args, useNeo4jClusterName...)
	if !assert.NoError(t, err) {
		return
	}

	service := coreOneManifest.OfTypeWithName(&v1.Service{}, "neo4j-cluster-lb-neo4j").(*v1.Service)
	selector := service.Spec.Selector

	checkCoreManifestHasPodMatchingSelector(t, coreOneManifest, selector)

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

// TestClusterCoreInternalPorts checks if the internals services for cluster core contains the expected ports
func TestNeo4jInternalPorts(t *testing.T) {
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

	neo4j := model.NewReleaseName("foo")
	desiredFeatures := [][]string{
		useEnterprise,
		useDataModeAndAcceptLicense,
		useNeo4jClusterName,
		enableCluster,
	}
	var args []string
	for _, a := range desiredFeatures {
		args = append(args, a...)
	}
	neo4jManifest, err := model.HelmTemplateForRelease(t, neo4j, model.HelmChart, args)
	if !assert.NoError(t, err) {
		return
	}

	internalService := neo4jManifest.OfTypeWithName(&v1.Service{}, neo4j.InternalServiceName()).(*v1.Service)

	checkPortsMatchExpected(t, expectedPorts, internalService)
}

// TestNeo4jPanicOnShutDownConfig checks whether the server.panic.shutdown_on_panic attribute is set to the default value true or not
func TestNeo4jPanicOnShutDownConfig(t *testing.T) {
	t.Parallel()

	neo4j := model.NewReleaseName("foo")
	desiredFeatures := [][]string{
		useEnterprise,
		useDataModeAndAcceptLicense,
		useNeo4jClusterName,
		enableCluster,
	}
	var args []string
	for _, a := range desiredFeatures {
		args = append(args, a...)
	}
	neo4jManifest, err := model.HelmTemplateForRelease(t, neo4j, model.HelmChart, args)
	if !assert.NoError(t, err) {
		return
	}

	defaultConfigMap := neo4jManifest.OfTypeWithName(&v1.ConfigMap{}, neo4j.DefaultConfigMapName()).(*v1.ConfigMap)
	assert.Contains(t, defaultConfigMap.Data, "server.panic.shutdown_on_panic")
	assert.Contains(t, defaultConfigMap.Data["server.panic.shutdown_on_panic"], "true")
}

// TestNeo4jServerAndUserLogsConfig checks whether the server and user logs are set in the respective conf files.
func TestNeo4jServerAndUserLogsConfig(t *testing.T) {
	t.Parallel()
	serverLogsXml := []string{"--set", "logging.serverLogsXml=\"unit test case to test it\""}
	userLogsXml := []string{"--set", "logging.userLogsXml=\"unit test case to test it\""}
	neo4j := model.NewReleaseName("foo")
	desiredFeatures := [][]string{
		useEnterprise,
		useDataModeAndAcceptLicense,
		useNeo4jClusterName,
		enableCluster,
		serverLogsXml,
		userLogsXml,
	}
	var args []string
	for _, a := range desiredFeatures {
		args = append(args, a...)
	}

	neo4jManifest, err := model.HelmTemplateForRelease(t, neo4j, model.HelmChart, useDataModeAndAcceptLicense, args...)
	if !assert.NoError(t, err) {
		return
	}

	defaultConfigMap := neo4jManifest.OfTypeWithName(&v1.ConfigMap{}, neo4j.DefaultConfigMapName()).(*v1.ConfigMap)
	userLogsConfigMap := neo4jManifest.OfTypeWithName(&v1.ConfigMap{}, neo4j.UserLogsConfigMapName()).(*v1.ConfigMap)
	serverLogsConfigMap := neo4jManifest.OfTypeWithName(&v1.ConfigMap{}, neo4j.ServerLogsConfigMapName()).(*v1.ConfigMap)
	assert.NotNil(t, userLogsConfigMap, " userLogs Config Map cannot be null")
	assert.NotNil(t, serverLogsConfigMap, " serverLogs Config Map cannot be null")
	assert.Contains(t, defaultConfigMap.Data, "server.user.config")
	assert.Contains(t, defaultConfigMap.Data, "server.logs.config")
	assert.Contains(t, defaultConfigMap.Data["server.user.config"], "/config/user-logs.xml/user-logs.xml")
	assert.Contains(t, defaultConfigMap.Data["server.logs.config"], "/config/server-logs.xml/server-logs.xml")
	assert.Contains(t, serverLogsConfigMap.Data["server-logs.xml"], "unit test case to test it")
	assert.Contains(t, userLogsConfigMap.Data["user-logs.xml"], "unit test case to test it")
}
