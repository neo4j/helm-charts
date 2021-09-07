package unit_tests

import (
	"github.com/neo-technology/neo4j-helm-charts/internal/model"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
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
