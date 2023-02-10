package unit_tests

import (
	"fmt"
	"github.com/neo4j/helm-charts/internal/model"
	"github.com/neo4j/helm-charts/internal/resources"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"testing"
)

func TestDefaultNeo4jNameClusterReadReplica(t *testing.T) {
	t.Parallel()

	readReplica := model.NewReleaseName("foo")

	readReplicaManifest, err := model.HelmTemplateForRelease(t, readReplica, model.ClusterReadReplicaHelmChart, useDataModeAndAcceptLicense, readReplicaTesting...)
	if !assert.NoError(t, err) {
		return
	}

	statefulSet := readReplicaManifest.Only(t, &appsv1.StatefulSet{}).(*appsv1.StatefulSet)
	selector := statefulSet.Labels

	assert.Contains(t, selector, "app")
	assert.Equal(t, selector["app"], "neo4j-cluster")
}

// TestReadReplicaInstallationFailure checks whether read replica installation is failing or not when clusters are not setup
// Since it's a unit test case failure will occur as cluster is not in place
func TestReadReplicaInstallationFailure(t *testing.T) {
	t.Parallel()

	readReplica := model.NewReleaseName("foo")

	_, err := model.HelmTemplateForRelease(t, readReplica, model.ClusterReadReplicaHelmChart, useDataModeAndAcceptLicense)
	if !assert.Error(t, err) {
		return
	}
	if !assert.Contains(t, err.Error(), "Cannot install Read Replica until a cluster of 3 or more cores is formed") {
		return
	}
}

// TODO : assert dbms mode to be READ_REPLICA
// TestReadReplicaInternalPorts checks if the internals services for read replica contains the expected ports
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

	readReplicaManifest, err := model.HelmTemplateForRelease(t, readReplica, model.ClusterReadReplicaHelmChart, useDataModeAndAcceptLicense, readReplicaTesting...)
	if !assert.NoError(t, err) {
		return
	}

	internalService := readReplicaManifest.OfTypeWithName(&v1.Service{}, readReplica.InternalServiceName()).(*v1.Service)

	checkPortsMatchExpected(t, expectedPorts, internalService)
}

// TestReadReplicaServerGroups checks if the configMap data has an entry called causal_clustering.server_groups
// It also checks if the key causal_clustering.server_groups contains a value "read-replicas" or not
func TestReadReplicaServerGroups(t *testing.T) {
	t.Parallel()

	readReplica := model.NewReleaseName("foo")

	readReplicaManifest, err := model.HelmTemplateForRelease(t, readReplica, model.ClusterReadReplicaHelmChart, useDataModeAndAcceptLicense, readReplicaTesting...)
	if !assert.NoError(t, err) {
		return
	}

	defaultConfigMap := readReplicaManifest.OfTypeWithName(&v1.ConfigMap{}, readReplica.DefaultConfigMapName()).(*v1.ConfigMap)
	assert.Contains(t, defaultConfigMap.Data, "causal_clustering.server_groups")
	assert.Contains(t, defaultConfigMap.Data["causal_clustering.server_groups"], "read-replicas")
}

// TestReadReplicaAntiAffinityRule checks if the podSpec.podAntiAffinity rule exists under statefulset or not
func TestReadReplicaAntiAffinityRuleExists(t *testing.T) {
	t.Parallel()

	readReplica := model.NewReleaseName("foo")

	readReplicaManifest, err := model.HelmTemplateForRelease(t, readReplica, model.ClusterReadReplicaHelmChart, useDataModeAndAcceptLicense, readReplicaTesting...)
	if !assert.NoError(t, err) {
		return
	}

	statefulSet := readReplicaManifest.Only(t, &appsv1.StatefulSet{}).(*appsv1.StatefulSet)
	assert.NotEqual(t, statefulSet.Spec.Template.Spec.Affinity.PodAntiAffinity, nil)
}

// TestReadReplicaAntiAffinityRule checks that podAntiAffinity rule should not exist when --set podSpec.podAntiAffinity=false is passed
func TestReadReplicaAntiAffinityRuleDoesNotExists(t *testing.T) {
	t.Parallel()

	readReplica := model.NewReleaseName("foo")

	readReplicaManifest, err := model.HelmTemplateForRelease(
		t,
		readReplica,
		model.ClusterReadReplicaHelmChart,
		useDataModeAndAcceptLicense,
		append(readReplicaTesting, "--set", "podSpec.podAntiAffinity=false")...,
	)
	if !assert.NoError(t, err) {
		return
	}

	statefulSet := readReplicaManifest.Only(t, &appsv1.StatefulSet{}).(*appsv1.StatefulSet)
	assert.Empty(t, statefulSet.Spec.Template.Spec.Affinity)
}

// TestReadReplicaPanicOnShutDownConfig checks whether the dbms.panic.shutdown_on_panic attribute is set to the default value true or not
func TestReadReplicaPanicOnShutDownConfig(t *testing.T) {
	t.Parallel()

	readReplica := model.NewReleaseName("foo")

	readReplicaManifest, err := model.HelmTemplateForRelease(t, readReplica, model.ClusterReadReplicaHelmChart, useDataModeAndAcceptLicense, readReplicaTesting...)
	if !assert.NoError(t, err) {
		return
	}

	defaultConfigMap := readReplicaManifest.OfTypeWithName(&v1.ConfigMap{}, readReplica.DefaultConfigMapName()).(*v1.ConfigMap)
	assert.Contains(t, defaultConfigMap.Data, "dbms.panic.shutdown_on_panic")
	assert.Contains(t, defaultConfigMap.Data["dbms.panic.shutdown_on_panic"], "true")
}

// TestReadReplicaInstallationWithLookupDisabled performs helm template on read replica helm chart with disableLookups set to true and --dry-run flag enabled
func TestReadReplicaInstallationWithLookupDisabled(t *testing.T) {
	t.Parallel()

	helmValues := model.DefaultEnterpriseValues
	helmValues.DisableLookups = true

	manifest, err := model.HelmTemplateFromStruct(t, model.ClusterReadReplicaHelmChart, helmValues, "--dry-run")
	if !assert.NoError(t, err) {
		return
	}
	statefulSet := manifest.OfTypeWithName(&appsv1.StatefulSet{}, model.DefaultHelmTemplateReleaseName.String())
	if !assert.NotNil(t, statefulSet, fmt.Sprintf("no statefulset found with name %s", model.DefaultHelmTemplateReleaseName)) {
		return
	}
	labels := statefulSet.(*appsv1.StatefulSet).ObjectMeta.Labels
	assert.Equal(t, labels["helm.neo4j.com/dbms.mode"], "READ_REPLICA", "read replica not found")
}

// TestReadReplicaWithMaintenanceModeEnabled checks for no error to be thrown when installing read replica with maintenancemode enabled
func TestReadReplicaWithMaintenanceModeEnabled(t *testing.T) {
	t.Parallel()

	readReplica := model.NewReleaseName("foo")

	_, err := model.HelmTemplateForRelease(t, readReplica, model.ClusterReadReplicaHelmChart, useDataModeAndAcceptLicense, resources.OfflineMaintenanceModeEnabled.HelmArgs()...)
	if !assert.NoError(t, err) {
		return
	}

}

//TODO : This is to be enabled in 5.0
//TestReadReplicaDefaultLogFormat checks whether the dbms.logs.default_format value is set to JSON or not
//func TestReadReplicaDefaultLogFormat(t *testing.T) {
//	t.Parallel()
//
//	readReplica := model.NewReleaseName("foo")
//
//	readReplicaManifest, err := model.HelmTemplateForRelease(t, readReplica, model.ClusterReadReplicaHelmChart, useDataModeAndAcceptLicense, readReplicaTesting...)
//	if !assert.NoError(t, err) {
//		return
//	}
//
//	defaultConfigMap := readReplicaManifest.OfTypeWithName(&v1.ConfigMap{}, readReplica.DefaultConfigMapName()).(*v1.ConfigMap)
//	assert.Contains(t, defaultConfigMap.Data, "dbms.logs.default_format")
//	assert.Contains(t, defaultConfigMap.Data["dbms.logs.default_format"], "JSON")
//}
