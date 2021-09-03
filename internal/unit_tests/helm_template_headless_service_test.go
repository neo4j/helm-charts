package unit_tests

import (
	"github.com/neo-technology/neo4j-helm-charts/internal/model"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	"testing"
)

func TestHeadlessServiceDefaults(t *testing.T) {
	t.Parallel()

	k8s, err := model.HelmTemplate(t, model.HeadlessServiceHelmChart, nil)
	if !assert.NoError(t, err) {
		return
	}

	services := k8s.OfType(&v1.Service{})
	assert.Len(t, k8s.All(), 1, "the loadbalancer chart should only create a single K8s object (a Service)")
	assert.Len(t, services, 1)

	service := services[0].(*v1.Service)
	assert.Equal(t, v1.ServiceType("ClusterIP"), service.Spec.Type )
	assert.Equal(t, service.Spec.ClusterIP, "None")
}
