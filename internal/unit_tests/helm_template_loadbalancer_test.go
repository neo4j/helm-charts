package unit_tests

import (
	"github.com/neo4j/helm-charts/internal/model"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	"testing"
)

func TestLoadBalancerDefaults(t *testing.T) {
	t.Parallel()

	expectedPorts := map[int32]int32{
		7474: 7474,
		7687: 7687,
		7473: 7473,
	}
	k8s, err := model.HelmTemplate(t, model.LoadBalancerHelmChart, nil)
	assert.NoError(t, err, "no error should be seen while installing loadbalancer helm chart")

	assert.Len(t, k8s.All(), 1, "the loadbalancer chart should only create a single K8s object (a Service)")
	services := k8s.OfType(&v1.Service{})
	assert.Len(t, services, 1)

	service := services[0].(*v1.Service)
	assert.Equal(t, "neo4j-loadbalancer-lb-neo4j", service.Name)

	selector := service.Spec.Selector
	assert.Len(t, selector, 2)
	assert.Equal(t,
		map[string]string{
			"app":                               "neo4j-loadbalancer",
			"helm.neo4j.com/neo4j.loadbalancer": "include",
		},
		selector)

	checkPortsMatchExpected(t, expectedPorts, service)
}

func checkPortsMatchExpected(t *testing.T, expectedPorts map[int32]int32, service *v1.Service) {
	// Check the ports
	assert.Len(t, service.Spec.Ports, len(expectedPorts))
	for _, port := range service.Spec.Ports {
		assert.Equal(t, v1.ProtocolTCP, port.Protocol)
		assert.Contains(t, expectedPorts, port.TargetPort.IntVal)
		assert.Equal(t, expectedPorts[port.TargetPort.IntVal], port.Port)
	}
}

func TestLoadBalancerPorts(t *testing.T) {
	t.Parallel()

	extraHelmArgs := []string{
		"--set", "ports.http.port=80",
		"--set", "ports.https.port=443",
		"--set", "ports.bolt.port=500",
		"--set", "ports.backup.enabled=true",
		"--set", "ports.backup.port=600",
	}

	expectedPorts := map[int32]int32{
		7474: 80,
		7473: 443,
		7687: 500,
		6362: 600,
	}

	k8s, err := model.HelmTemplate(t, model.LoadBalancerHelmChart, extraHelmArgs)
	assert.NoError(t, err, "no error should be seen while installing loadbalancer helm chart")

	assert.Len(t, k8s.All(), 1, "the loadbalancer chart should only create a single K8s object (a Service)")
	services := k8s.OfType(&v1.Service{})
	assert.Len(t, services, 1)

	service := services[0].(*v1.Service)
	assert.Equal(t, "neo4j-loadbalancer-lb-neo4j", service.Name)

	selector := service.Spec.Selector
	assert.Len(t, selector, 2)
	assert.Equal(t,
		map[string]string{
			"app":                               "neo4j-loadbalancer",
			"helm.neo4j.com/neo4j.loadbalancer": "include",
		},
		selector)

	checkPortsMatchExpected(t, expectedPorts, service)
}

func TestOverrideLoadBalancerDefaultSettings(t *testing.T) {
	t.Parallel()

	// When no extra args are set...
	k8s, err := model.HelmTemplate(t, model.LoadBalancerHelmChart, nil)
	assert.NoError(t, err, "no error should be seen while installing loadbalancer helm chart")
	// Our "default" settings (externalTrafficPolicy: local) are applied
	service := k8s.OfTypeWithName(&v1.Service{}, "neo4j-loadbalancer-lb-neo4j").(*v1.Service)
	assert.Equal(t, v1.ServiceExternalTrafficPolicyTypeLocal, service.Spec.ExternalTrafficPolicy)

	// When user sets them explicitly
	extraHelmArgs := []string{
		"--set", "spec.externalTrafficPolicy=" + string(v1.ServiceExternalTrafficPolicyTypeCluster),
	}

	k8s, err = model.HelmTemplate(t, model.LoadBalancerHelmChart, extraHelmArgs)
	if !assert.NoError(t, err) {
		return
	}

	// Our "default" settings are overridden
	service = k8s.OfTypeWithName(&v1.Service{}, "neo4j-loadbalancer-lb-neo4j").(*v1.Service)
	assert.Equal(t, v1.ServiceExternalTrafficPolicyTypeCluster, service.Spec.ExternalTrafficPolicy)
}

func TestLoadBalancerNodePortSettings(t *testing.T) {
	t.Parallel()

	var helmValues model.Neo4jLoadBalancerValues
	helmValues.Neo4j.Name = "demo-lb"
	helmValues.Neo4j.Edition = "enterprise"
	helmValues.Spec.Type = "NodePort"
	helmValues.Ports.HTTP.Enabled = true
	helmValues.Ports.HTTP.NodePort = 1234

	manifest, err := model.HelmTemplateFromStruct(t, model.LoadBalancerHelmChart, helmValues, "--dry-run")
	if !assert.NoError(t, err) {
		return
	}

	services := manifest.OfType(&v1.Service{})
	assert.NotNil(t, services, "services missing")

	var nodePortService *v1.Service
	for _, service := range services {
		if string(service.(*v1.Service).Spec.Type) == "NodePort" {
			nodePortService = service.(*v1.Service)
			break
		}
	}
	assert.NotNil(t, nodePortService, "no nodeport service found")
	ports := nodePortService.Spec.Ports
	assert.NotNil(t, ports, "no ports found for nodeport service")

	var portFound bool
	for _, port := range ports {
		if port.Name == "http" {
			assert.Equal(t, port.NodePort, int32(1234), "nodePort found is not matching")
			portFound = true
			break
		}
	}
	assert.Equal(t, portFound, true, "nodePort not found")
}
