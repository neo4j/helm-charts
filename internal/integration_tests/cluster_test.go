package integration_tests

import (
	"fmt"
	. "github.com/neo-technology/neo4j-helm-charts/internal/helpers"
	"github.com/neo-technology/neo4j-helm-charts/internal/integration_tests/gcloud"
	"github.com/neo-technology/neo4j-helm-charts/internal/model"
	"github.com/neo-technology/neo4j-helm-charts/internal/resources"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	"strings"
	"testing"
)

type parallelResult struct {
	Closeable
	error
}

type helmComponent interface {
	Name() model.ReleaseName
	Install(t *testing.T) parallelResult
}

type clusterCore struct {
	name model.ReleaseName
}

func (c clusterCore) Name() model.ReleaseName {
	return c.name
}

func (c clusterCore) Install(t *testing.T) parallelResult {
	var err error
	var cleanup Closeable
	cleanup, err = InstallNeo4jInGcloud(t, gcloud.CurrentZone(), gcloud.CurrentProject(), c.name, model.ClusterCoreHelmChart)
	return parallelResult{cleanup, err}
}

type clusterReadReplica struct {
	name model.ReleaseName
}

func (c clusterReadReplica) Name() model.ReleaseName {
	return c.name
}

func (c clusterReadReplica) Install(t *testing.T) parallelResult {
	var err error
	var cleanup Closeable
	cleanup, err = InstallNeo4jInGcloud(t, gcloud.CurrentZone(), gcloud.CurrentProject(), c.name, model.ClusterReadReplicaHelmChart)
	return parallelResult{cleanup, err}
}

type clusterLoadBalancer struct {
	name model.ReleaseName
}

func (c clusterLoadBalancer) Name() model.ReleaseName {
	return c.name
}

func (c clusterLoadBalancer) Install(t *testing.T) parallelResult {
	var err error
	var cleanup Closeable
	cleanup = func() error { return run(t, "helm", model.LoadBalancerHelmCommand("uninstall", c.name)...) }
	err = run(t, "helm", model.LoadBalancerHelmCommand("install", c.name)...)
	return parallelResult{cleanup, err}
}

func clusterTests(loadBalancerName model.ReleaseName, readReplica1Name model.ReleaseName, readReplica2Name model.ReleaseName) ([]SubTest, error) {
	expectedConfiguration, err := (&model.Neo4jConfiguration{}).PopulateFromFile(Neo4jConfFile)
	if err != nil {
		return nil, err
	}
	expectedConfiguration = addExpectedClusterConfiguration(expectedConfiguration)

	return []SubTest{
		{name: "Check K8s", test: func(t *testing.T) {
			assert.NoError(t, CheckK8s(t, loadBalancerName), "Neo4j Config check should succeed")
		}},
		{name: "Check Neo4j Configuration", test: func(t *testing.T) {
			assert.NoError(t, CheckNeo4jConfiguration(t, loadBalancerName, expectedConfiguration), "Neo4j Config check should succeed")
		}},
		{name: "Create Node", test: func(t *testing.T) {
			assert.NoError(t, CreateNode(t, loadBalancerName), "Create Node should succeed")
		}},
		{name: "Count Nodes", test: func(t *testing.T) {
			assert.NoError(t, CheckNodeCount(t, loadBalancerName), "Count Nodes should succeed")
		}},
		{name: "Check Read Replica Configuration", test: func(t *testing.T) {
			assert.NoError(t, CheckReadReplicaConfiguration(t, readReplica1Name), "Checks Read Replica Configuration")
		}},
		{name: "Check Read Replica Server Groups", test: func(t *testing.T) {
			assert.NoError(t, CheckReadReplicaServerGroupsConfiguration(t, readReplica1Name), "Checks Read Replica Server Groups config contains read-replicas or not")
		}},
		{name: "Update Read Replica With Upstream Strategy on Read Replica 2", test: func(t *testing.T) {
			assert.NoError(t, UpdateReadReplicaConfig(t, readReplica2Name, resources.ReadReplicaUpstreamStrategy.HelmArgs()...), "Adds upstream strategy on read replica")
		}},
		{name: "Create Node on Read Replica 1", test: func(t *testing.T) {
			assert.NoError(t, CreateNodeOnReadReplica(t, readReplica1Name), "Create Node on read replica should be redirected to the cluster code")
		}},
		{name: "Count Nodes on Read Replica 1", test: func(t *testing.T) {
			assert.NoError(t, CheckNodeCountOnReadReplica(t, readReplica1Name, 2), "Count Nodes on read replica should succeed")
		}},
		{name: "Count Nodes on Read Replica 2 Via Upstream Strategy", test: func(t *testing.T) {
			assert.NoError(t, CheckNodeCountOnReadReplica(t, readReplica2Name, 2), "Count Nodes on read replica2 should succeed by fetching it from read replica 1")
		}},
		{name: "Update Read Replica 2 to exclude from load balancer", test: func(t *testing.T) {
			assert.NoError(t, UpdateReadReplicaConfig(t, readReplica2Name, resources.ExcludeLoadBalancer.HelmArgs()...), "Performs helm upgrade on read replica 2 to exclude it from loadbalancer")
		}},
		{name: "Check Load Balancer Exclusion Property", test: func(t *testing.T) {
			assert.NoError(t, CheckLoadBalancerExclusion(t, readReplica2Name, loadBalancerName), "LoadBalancer Exclusion Test should succeed")
		}},
	}, err
}

//CheckLoadBalancerExclusion updates the label on the provided read replica to excluded it from loadbalancer
/*We check for two things : the loadbalancer count should be 4 (3 cores + 1 rr) and the loadbalancer endpoints list should not
contain the given read replica pod ip*/
func CheckLoadBalancerExclusion(t *testing.T, readReplicaName model.ReleaseName, loadBalancerName model.ReleaseName) error {

	////updating the read replica to exclude itself from loadbalancer
	if !assert.NoError(t, UpdateReadReplicaConfig(t, readReplicaName, resources.ExcludeLoadBalancer.HelmArgs()...)) {
		return fmt.Errorf("error seen while updating read replica config")
	}

	manifest, err := getManifest(loadBalancerName.Namespace())
	if !assert.Nil(t, err) {
		return err
	}

	services := manifest.OfType(&v1.Service{})
	var lbService *v1.Service
	for _, service := range services {
		if strings.HasSuffix(service.(*v1.Service).Name, "-neo4j") {
			lbService = service.(*v1.Service)
			break
		}
	}

	if !assert.NotNil(t, lbService) {
		return fmt.Errorf("loadbalancer service not found")
	}
	readReplicaPod := manifest.OfTypeWithName(&v1.Pod{}, readReplicaName.PodName()).(*v1.Pod)
	lbEndpoints := manifest.OfTypeWithName(&v1.Endpoints{}, lbService.Name).(*v1.Endpoints)

	assert.Len(t, lbEndpoints.Subsets, 1)
	assert.Len(t, lbEndpoints.Subsets[0].Addresses, 4)
	assert.NotContains(t, lbEndpoints.Subsets[0].Addresses, readReplicaPod.Status.PodIP)

	return nil
}

func CheckK8s(t *testing.T, name model.ReleaseName) error {
	t.Run("check pods", func(t *testing.T) {
		t.Parallel()
		assert.NoError(t, CheckPods(t, name))
	})
	t.Run("check lb", func(t *testing.T) {
		t.Parallel()
		assert.NoError(t, CheckLoadBalancerService(t, name, 5))
	})
	return nil
}

//CheckLoadBalancerService checks whether the loadbalancer exists or not
//It also checks that the number of endpoints should match with the given number of expected endpoints
func CheckLoadBalancerService(t *testing.T, name model.ReleaseName, expectedEndPoints int) error {
	manifest, err := getManifest(name.Namespace())
	if !assert.NoError(t, err) {
		return err
	}

	services := manifest.OfType(&v1.Service{})
	var lbService *v1.Service
	for _, service := range services {
		if strings.HasSuffix(service.(*v1.Service).Name, "-neo4j") {
			if !assert.Nil(t, lbService, "There should only be one -neo4j service in this namespace") {
				return fmt.Errorf("There should only be one -neo4j service in this namespace")
			}
			lbService = service.(*v1.Service)
			break
		}
	}

	lbEndpoints := manifest.OfTypeWithName(&v1.Endpoints{}, lbService.Name).(*v1.Endpoints)
	assert.Len(t, lbEndpoints.Subsets, 1)
	assert.Len(t, lbEndpoints.Subsets[0].Addresses, expectedEndPoints)
	return nil
}

//CheckPods checks for the number of pods which should be 5 (3 cluster core + 2 read replica)
func CheckPods(t *testing.T, name model.ReleaseName) error {
	pods, err := getAllPods(name.Namespace())
	if !assert.NoError(t, err) {
		return err
	}

	//5 = 3 cores + 2 read replica
	assert.Len(t, pods.Items, 5)
	for _, pod := range pods.Items {
		if assert.Contains(t, pod.Labels, "app") {
			assert.Equal(t, "neo4j-cluster", pod.Labels["app"])
		}
	}

	return err
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
	if model.Neo4jEdition != "enterprise" {
		t.Skip()
		return
	}
	t.Parallel()

	clusterReleaseName := model.NewReleaseName("cluster-" + TestRunIdentifier)
	loadBalancer := clusterLoadBalancer{model.NewLoadBalancerReleaseName(clusterReleaseName)}
	readReplica1 := clusterReadReplica{model.NewReadReplicaReleaseName(clusterReleaseName, 1)}
	readReplica2 := clusterReadReplica{model.NewReadReplicaReleaseName(clusterReleaseName, 2)}
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
	result := core1.Install(t)
	addCloseable(result.Closeable)
	if !assert.NoError(t, result.error) {
		return
	}

	// parallel for loop using goroutines
	componentsToParallelInstall := []helmComponent{
		core2,
		core3,
		loadBalancer,
		readReplica1,
		readReplica2,
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

	subTests, err := clusterTests(loadBalancer.Name(), readReplica1.Name(), readReplica2.Name())
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
		parallelResult = component.Install(t)
	}()
}
