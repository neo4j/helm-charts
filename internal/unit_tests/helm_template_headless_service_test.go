package unit_tests

import (
	"fmt"
	"github.com/neo4j/helm-charts/internal/model"
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
	assert.Equal(t, v1.ServiceType("ClusterIP"), service.Spec.Type)
	assert.Equal(t, service.Spec.ClusterIP, "None")
}

// TestHeadlessServiceForPortRemapping checks whether headless service issues an error during port remapping or not
func TestHeadlessServiceForPortRemapping(t *testing.T) {

	t.Parallel()

	portRemappingMessage := fmt.Sprintf("port re-mapping is not allowed in headless service. Please remove custom port 9000 from values.yaml")
	portRemappingArgs := []string{
		"--set", "ports.http.enabled=true",
		"--set", "ports.http.port=9000", // this should fail since only 7474 port is allowed
	}

	_, err := model.HelmTemplate(t, model.HeadlessServiceHelmChart, portRemappingArgs)
	if !assert.Error(t, err) {
		return
	}
	if !assert.Contains(t, err.Error(), portRemappingMessage) {
		return
	}

	portRemappingArgs = []string{
		"--set", "ports.http.enabled=true",
		"--set", "ports.http.port=7474",
	}

	_, err = model.HelmTemplate(t, model.HeadlessServiceHelmChart, portRemappingArgs)
	if !assert.NoError(t, err) {
		return
	}
}
