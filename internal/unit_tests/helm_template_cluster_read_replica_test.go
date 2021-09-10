package unit_tests

import (
	"github.com/neo-technology/neo4j-helm-charts/internal/model"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"testing"
)

func TestDefaultNeo4jNameClusterReadReplica(t *testing.T) {
	t.Parallel()

	readReplica := model.NewReleaseName("foo")

	readReplicaManifest, err := model.HelmTemplateForRelease(t, readReplica, model.ClusterReadReplicaHelmChart, useDataModeAndAcceptLicense)
	if !assert.NoError(t, err) {
		return
	}

	statefulSet := readReplicaManifest.Only(t, &appsv1.StatefulSet{}).(*appsv1.StatefulSet)
	selector := statefulSet.Labels

	assert.Contains(t, selector, "app")
	assert.Equal(t, selector["app"], "neo4j-cluster")
}

//TODO : assert dbms mode to be READ_REPLICA
//TestReadReplicaInternalPorts checks if the internals services for read replica contains the expected ports
func TestReadReplicaInternalPorts(t *testing.T) {
	t.Parallel()

	expectedPorts := map[int32]int32{
		6362: 6362,
		7687: 7687,
		7474: 7474,
		7473: 7473,
		7688: 7688,
		6000: 6000,
	}

	readReplica := model.NewReleaseName("foo")

	readReplicaManifest, err := model.HelmTemplateForRelease(t, readReplica, model.ClusterReadReplicaHelmChart, useDataModeAndAcceptLicense)
	if !assert.NoError(t, err) {
		return
	}

	internalService := readReplicaManifest.OfTypeWithName(&v1.Service{}, readReplica.InternalServiceName()).(*v1.Service)

	checkPortsMatchExpected(t, expectedPorts, internalService)
}

//TestReadReplicaServerGroups checks if the configMap data has an entry called causal_clustering.server_groups
// It also checks if the key causal_clustering.server_groups contains a value "read-replicas" or not
func TestReadReplicaServerGroups(t *testing.T) {
	t.Parallel()

	readReplica := model.NewReleaseName("foo")

	readReplicaManifest, err := model.HelmTemplateForRelease(t, readReplica, model.ClusterReadReplicaHelmChart, useDataModeAndAcceptLicense)
	if !assert.NoError(t, err) {
		return
	}

	defaultConfigMap := readReplicaManifest.OfTypeWithName(&v1.ConfigMap{}, readReplica.DefaultConfigMapName()).(*v1.ConfigMap)
	assert.Contains(t, defaultConfigMap.Data, "causal_clustering.server_groups")
	assert.Contains(t, defaultConfigMap.Data["causal_clustering.server_groups"], "read-replicas")
}

//TestReadReplicaAntiAffinityRule checks if the podSpec.podAntiAffinity rule exists under statefulset or not
func TestReadReplicaAntiAffinityRuleExists(t *testing.T) {
	t.Parallel()

	readReplica := model.NewReleaseName("foo")

	readReplicaManifest, err := model.HelmTemplateForRelease(t, readReplica, model.ClusterReadReplicaHelmChart, useDataModeAndAcceptLicense)
	if !assert.NoError(t, err) {
		return
	}

	statefulSet := readReplicaManifest.Only(t, &appsv1.StatefulSet{}).(*appsv1.StatefulSet)
	assert.NotEqual(t, statefulSet.Spec.Template.Spec.Affinity.PodAntiAffinity, nil)
}

//TestReadReplicaAntiAffinityRule checks that podAntiAffinity rule should not exist when --set podSpec.podAntiAffinity=false is passed
func TestReadReplicaAntiAffinityRuleDoesNotExists(t *testing.T) {
	t.Parallel()

	readReplica := model.NewReleaseName("foo")

	readReplicaManifest, err := model.HelmTemplateForRelease(
		t,
		readReplica,
		model.ClusterReadReplicaHelmChart,
		useDataModeAndAcceptLicense,
		[]string{"--set", "podSpec.podAntiAffinity=false"}...,
	)
	if !assert.NoError(t, err) {
		return
	}

	statefulSet := readReplicaManifest.Only(t, &appsv1.StatefulSet{}).(*appsv1.StatefulSet)
	assert.Empty(t, statefulSet.Spec.Template.Spec.Affinity)
}
