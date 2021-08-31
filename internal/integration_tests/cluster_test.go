package integration_tests

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	. "neo4j.com/helm-charts-tests/internal/helpers"
	"neo4j.com/helm-charts-tests/internal/integration_tests/gcloud"
	"neo4j.com/helm-charts-tests/internal/model"
	"testing"
)

type parallelResult struct {
	Closeable
	error
}

type helmComponent interface {
	Name() model.ReleaseName
	Install(t *testing.T, clusterName model.ReleaseName) parallelResult
}

type clusterCore struct {
	name model.ReleaseName
}

func (c clusterCore) Name() model.ReleaseName {
	return c.name
}

func (c clusterCore) Install(t *testing.T, clusterName model.ReleaseName) parallelResult {
	var err error
	var cleanup Closeable
	cleanup, err = InstallNeo4jInGcloud(t, gcloud.CurrentZone(), gcloud.CurrentProject(), c.name, model.ClusterCoreHelmChart)
	return parallelResult{cleanup, err}
}

type clusterLoadBalancer struct {
	name model.ReleaseName
}

func (c clusterLoadBalancer) Name() model.ReleaseName {
	return c.name
}

func (c clusterLoadBalancer) Install(t *testing.T, clusterName model.ReleaseName) parallelResult {
	var err error
	var cleanup Closeable
	cleanup = func() error { return run(t, "helm", model.LoadBalancerHelmCommand("uninstall", c.name)...) }
	err = run(t, "helm", model.LoadBalancerHelmCommand("install", c.name)...)
	return parallelResult{cleanup, err}
}

func clusterTests(loadBalancerName model.ReleaseName) ([]SubTest, error) {
	expectedConfiguration, err := (&model.Neo4jConfiguration{}).PopulateFromFile(Neo4jConfFile)
	if err != nil {
		return nil, err
	}
	expectedConfiguration = addExpectedClusterConfiguration(expectedConfiguration)

	return []SubTest{
		{name: "Check Neo4j Configuration", test: func(t *testing.T) { assert.NoError(t, CheckNeo4jConfiguration(t, loadBalancerName, expectedConfiguration), "Neo4j Config check should succeed") }},
		{name: "Create Node", test: func(t *testing.T) { assert.NoError(t, CreateNode(t, loadBalancerName), "Create Node should succeed") }},
		{name: "Count Nodes", test: func(t *testing.T) { assert.NoError(t, CheckNodeCount(t, loadBalancerName), "Count Nodes should succeed") }},
	}, err
}

func addExpectedClusterConfiguration(configuration *model.Neo4jConfiguration) *model.Neo4jConfiguration {
	updatedConfig := configuration.UpdateFromMap(map[string]string{
		"dbms.mode":                                      "CORE",
		"causal_clustering.discovery_type":               "K8S",
		"causal_clustering.kubernetes.service_port_name": "tcp-discovery",
		"causal_clustering.kubernetes.label_selector":    "app=neo4j-cluster,helm.neo4j.com/service=internals,helm.neo4j.com/dbms.mode=CORE",
		"dbms.routing.default_router":                    "SERVER",
		"dbms.routing.enabled":                           "true",
	}, true)
	return &updatedConfig
}

func TestInstallNeo4jClusterInGcloud(t *testing.T) {
	if Neo4jEdition != "enterprise" {
		t.Skip()
		return
	}
	t.Parallel()

	clusterReleaseName := model.NewReleaseName("cluster-" + TestRunIdentifier)
	loadBalancer := clusterLoadBalancer{model.NewLoadBalancerReleaseName(clusterReleaseName)}
	core1 := clusterCore{model.NewCoreReleaseName(clusterReleaseName, 1)}
	core2 := clusterCore{model.NewCoreReleaseName(clusterReleaseName, 2)}
	core3 := clusterCore{model.NewCoreReleaseName(clusterReleaseName, 3)}
	cores := []clusterCore{core1, core2, core3}

	var closeables []Closeable
	addCloseable := func(closeable Closeable) {
		closeables = append([]Closeable{closeable}, closeables...)
	}

	t.Cleanup(func() { cleanupTest(t, AsCloseable(closeables)) })

	t.Logf("Starting setup of '%s'", t.Name())

	closeable, err := prepareK8s(t, clusterReleaseName)
	addCloseable(closeable)
	if !assert.NoError(t, err) {
		return
	}

	// Install one core synchronously, if all cores are installed simultaneously they run into conflicts all trying to create a -auth secret
	result := core1.Install(t, clusterReleaseName)
	addCloseable(result.Closeable)
	if !assert.NoError(t, result.error) {
		return
	}

	// parallel for loop using goroutines
	componentsToParallelInstall := []helmComponent{
		core2,
		core3,
		loadBalancer,
	}
	results := make(chan parallelResult)
	for _, component := range componentsToParallelInstall {
		backgroundInstall(t, results, component, clusterReleaseName)
	}

	var combinedError error
	for i := 0; i < len(componentsToParallelInstall); i++ {
		result := <-results
		addCloseable(result.Closeable)
		if result.error != nil {
			combinedError = CombineErrors(combinedError, result.error)
		}
	}
	if !assert.NoError(t, combinedError) {
		return
	}

	for _, core := range cores {
		err = run(t, "kubectl", "--namespace", string(core.Name().Namespace()), "rollout", "status", "--watch", "--timeout=180s", "statefulset/"+core.Name().String())
		if !assert.NoError(t, err) {
			return
		}
	}

	t.Logf("Succeeded with setup of '%s'", t.Name())

	subTests, err := clusterTests(loadBalancer.Name())
	if !assert.NoError(t, err) {
		return
	}
	runSubTests(t, subTests)

	t.Logf("Succeeded running all tests in '%s'", t.Name())
}

func backgroundInstall(t *testing.T, results chan parallelResult, component helmComponent, clusterName model.ReleaseName) {
	go func() {
		var parallelResult = parallelResult{
			Closeable: nil,
			error:     fmt.Errorf("illegal state: background install did not take place for %s in %s", component.Name(), clusterName),
		}
		defer func() { results <- parallelResult }()
		parallelResult = component.Install(t, clusterName)
	}()
}
