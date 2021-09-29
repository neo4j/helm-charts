package integration_tests

import (
	"context"
	"fmt"
	. "github.com/neo-technology/neo4j-helm-charts/internal/helpers"
	"github.com/neo-technology/neo4j-helm-charts/internal/integration_tests/gcloud"
	"github.com/neo-technology/neo4j-helm-charts/internal/model"
	"github.com/neo-technology/neo4j-helm-charts/internal/resources"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"os/exec"
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

type clusterHeadLessService struct {
	name model.ReleaseName
}

func (c clusterHeadLessService) Name() model.ReleaseName {
	return c.name
}

func (c clusterHeadLessService) Install(t *testing.T) parallelResult {
	var err error
	var cleanup Closeable
	cleanup = func() error { return run(t, "helm", model.HeadlessServiceHelmCommand("uninstall", c.name)...) }
	err = run(t, "helm", model.HeadlessServiceHelmCommand("install", c.name)...)
	return parallelResult{cleanup, err}
}

//clusterTests contains all the tests related to cluster
func clusterTests(loadBalancerName model.ReleaseName, core model.ReleaseName) ([]SubTest, error) {
	expectedConfiguration, err := (&model.Neo4jConfiguration{}).PopulateFromFile(Neo4jConfFile)
	if err != nil {
		return nil, err
	}
	expectedConfiguration = addExpectedClusterConfiguration(expectedConfiguration)

	subTests := []SubTest{
		{name: "Check Cluster Core Logs Format", test: func(t *testing.T) {
			t.Parallel()
			assert.NoError(t, CheckLogsFormat(t, core), "Cluster core logs format should be in JSON")
		}},
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
		{name: "Check Cluster Password failure", test: func(t *testing.T) {
			t.Parallel()
			assert.NoError(t, CheckClusterCorePasswordFailure(t), "Cluster core installation should not succeed with incorrect password")
		}},
		{name: "Create Database customers", test: func(t *testing.T) {
			assert.NoError(t, CreateDatabase(t, loadBalancerName, "customers"), "Creates customer database")
		}},
		{name: "Check Database customers exists", test: func(t *testing.T) {
			assert.NoError(t, CheckDataBaseExists(t, loadBalancerName, "customers"), "Checks if customer database exists or not")
		}},
	}
	return subTests, nil
}

//CheckLogsFormat checks whether the neo4j logs are in json format or not
// we check for the json "level":"INFO","message":"Started."} in /logs/neo4j.log
func CheckLogsFormat(t *testing.T, releaseName model.ReleaseName) error {

	stdout, stderr, err := ExecInPod(releaseName, []string{"cat", "/logs/neo4j.log"})
	if !assert.NoError(t, err) {
		return fmt.Errorf("error seen while executing command `cat /logs/neo4j.log' ,\n err :- %v", err)
	}
	if !assert.Contains(t, stdout, ",\"level\":\"INFO\",\"message\":\"Started.\"}") {
		return fmt.Errorf("foes not contain the required json format\n stdout := %s", stdout)
	}
	if !assert.Len(t, stderr, 0) {
		return fmt.Errorf("stderr found while checking logs \n stderr := %s", stderr)
	}
	return nil
}

//headLessServiceTests contains all the tests related to headless service
func headLessServiceTests(headlessService model.ReleaseName) []SubTest {
	return []SubTest{
		{name: "Check Headless Service Configuration", test: func(t *testing.T) {
			t.Parallel()
			assert.NoError(t, CheckHeadlessServiceConfiguration(t, headlessService), "Checks Headless Service configuration")
		}},
		{name: "Check Headless Service Endpoints", test: func(t *testing.T) {
			t.Parallel()
			assert.NoError(t, CheckHeadlessServiceEndpoints(t, headlessService), "headless service endpoints should be equal to the cluster core created")
		}},
	}
}

//readReplicaTests contains all the tests related to read replicas
func readReplicaTests(readReplica1Name model.ReleaseName, readReplica2Name model.ReleaseName, loadBalancerName model.ReleaseName) []SubTest {
	return []SubTest{
		{name: "Check ReadReplica Logs Format", test: func(t *testing.T) {
			t.Parallel()
			assert.NoError(t, CheckLogsFormat(t, readReplica1Name), "Read Replica logs format should be in JSON")
		}},
		{name: "Check Read Replica Logs Format", test: func(t *testing.T) {
			t.Parallel()
			assert.NoError(t, CheckLogsFormat(t, readReplica1Name), "Checks Read Replica Logs Format")
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
	}
}

//CheckClusterCorePasswordFailure checks if a cluster core is failing on installation or not with an incorrect password
func CheckClusterCorePasswordFailure(t *testing.T) error {
	//creating a sample cluster core definition (which is not supposed to get installed)
	clusterReleaseName := model.NewReleaseName("cluster-" + TestRunIdentifier)
	core := clusterCore{model.NewCoreReleaseName(clusterReleaseName, 4)}
	releaseName := core.Name()
	diskName := releaseName.DiskName()
	// we are not using the customized run() func here since we need to assert the error received on stdout
	//(present in out variable and not in err)
	out, err := exec.Command(
		"helm",
		model.BaseHelmCommand(
			"install",
			releaseName,
			model.ClusterCoreHelmChart,
			model.Neo4jEdition,
			&diskName,
			"--set", "neo4j.password=my-password")...).CombinedOutput()
	if !assert.Error(t, err) {
		return fmt.Errorf("helm install should fail without the default password")
	}
	if !assert.Contains(t, string(out), "The desired password does not match the password stored in the Kubernetes Secret") {
		return fmt.Errorf("error thrown on password failure is different , err := %s", string(out))
	}
	return nil
}

//CheckLoadBalancerExclusion updates the label on the provided read replica to excluded it from loadbalancer
/*We check for two things : the loadbalancer count should be 4 (3 cores + 1 rr) and the loadbalancer endpoints list should not
contain the given read replica pod ip*/
func CheckLoadBalancerExclusion(t *testing.T, readReplicaName model.ReleaseName, loadBalancerName model.ReleaseName) error {

	//updating the read replica to exclude itself from loadbalancer
	if !assert.NoError(t, UpdateReadReplicaConfig(t, readReplicaName, resources.ExcludeLoadBalancer.HelmArgs()...)) {
		return fmt.Errorf("error seen while updating read replica config")
	}

	serviceName := fmt.Sprintf("%s-neo4j", loadBalancerName.String())
	lbService, err := Clientset.CoreV1().Services(string(loadBalancerName.Namespace())).Get(context.TODO(), serviceName, metav1.GetOptions{})
	if !assert.NoError(t, err) {
		return fmt.Errorf("loadbalancer service %s not found , Error seen := %v", loadBalancerName.String(), err)
	}

	manifest, err := getManifest(loadBalancerName.Namespace())
	if !assert.NoError(t, err) {
		return err
	}

	if !assert.NotNil(t, lbService) {
		return fmt.Errorf("loadbalancer service not found")
	}

	readReplicaPod := manifest.OfTypeWithName(&v1.Pod{}, readReplicaName.PodName()).(*v1.Pod)
	lbEndpoints := manifest.OfTypeWithName(&v1.Endpoints{}, lbService.Name).(*v1.Endpoints)

	if !assert.NotNil(t, readReplicaPod) {
		return fmt.Errorf("readReplicaPod with name %s should exist", readReplicaName.PodName())
	}
	if !assert.NotNil(t, lbEndpoints) {
		return fmt.Errorf("loadbalancer endpoints should not be empty")
	}
	if !assert.Len(t, lbEndpoints.Subsets, 1) {
		return fmt.Errorf("subsets length should be equal to 1")
	}
	if !assert.Len(t, lbEndpoints.Subsets[0].Addresses, 4) {
		return fmt.Errorf("number of endpoints should be 4")
	}
	if !assert.NotContains(t, lbEndpoints.Subsets[0].Addresses, readReplicaPod.Status.PodIP) {
		return fmt.Errorf("loadbalancer endpoints should not contains the readreplica podIP")
	}

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

	serviceName := fmt.Sprintf("%s-neo4j", name.String())
	lbService, err := Clientset.CoreV1().Services(string(name.Namespace())).Get(context.TODO(), serviceName, metav1.GetOptions{})
	if !assert.NoError(t, err) {
		return fmt.Errorf("loadbalancer service %s not found , Error seen := %v", name.String(), err)
	}

	lbEndpoints, err := Clientset.CoreV1().Endpoints(string(name.Namespace())).Get(context.TODO(), lbService.Name, metav1.GetOptions{})
	if !assert.NoError(t, err) {
		return fmt.Errorf("failed to get loadbalancer service endpoints %v", err)
	}
	if !assert.Len(t, lbEndpoints.Subsets, 1) {
		return fmt.Errorf("lbendpoints subsets length should be equal to 1")
	}
	if !assert.Len(t, lbEndpoints.Subsets[0].Addresses, expectedEndPoints) {
		return fmt.Errorf("loadbalancer endpoints count should be %d", expectedEndPoints)
	}
	return nil
}

//CheckPods checks for the number of pods which should be 5 (3 cluster core + 2 read replica)
func CheckPods(t *testing.T, name model.ReleaseName) error {
	pods, err := getAllPods(name.Namespace())
	if !assert.NoError(t, err) {
		return err
	}

	//5 = 3 cores + 2 read replica
	if !assert.Len(t, pods.Items, 5) {
		return fmt.Errorf("number of pods should be 5")
	}
	for _, pod := range pods.Items {
		if assert.Contains(t, pod.Labels, "app") {
			if !assert.Equal(t, "neo4j-cluster", pod.Labels["app"]) {
				return fmt.Errorf("pod should have label app=neo4jcluster , found app=%s", pod.Labels["app"])
			}
		}
	}

	return nil
}

//CheckHeadlessServiceConfiguration checks whether the provided service is headless service or not
func CheckHeadlessServiceConfiguration(t *testing.T, service model.ReleaseName) error {

	serviceName := fmt.Sprintf("%s-neo4j", service.String())
	headlessService, err := Clientset.CoreV1().Services(string(service.Namespace())).Get(context.TODO(), serviceName, metav1.GetOptions{})
	if !assert.NoError(t, err) {
		return fmt.Errorf("headless service %s not found , Error seen := %v", service.String(), err)
	}
	//throw error if headless service is nil which means no headless service object is released
	if !assert.NotNil(t, headlessService) {
		return fmt.Errorf("headless service is nil")
	}
	if !assert.Equal(t, headlessService.Spec.ClusterIP, "None") {
		return fmt.Errorf("provided clusterIP is not 'None'...it is %s", headlessService.Spec.ClusterIP)
	}

	return nil
}

//CheckHeadlessServiceEndpoints checks whether the headless endpoints have the cluster cores or not
// By default , headless service includes cluster core only and no read replicas
func CheckHeadlessServiceEndpoints(t *testing.T, service model.ReleaseName) error {

	serviceName := fmt.Sprintf("%s-neo4j", service.String())

	//get the endpoints associated with the headless service
	endpoints, err := Clientset.CoreV1().Endpoints(string(service.Namespace())).Get(context.TODO(), serviceName, metav1.GetOptions{})
	if !assert.NoError(t, err) {
		return fmt.Errorf("failed to get headless service endpoints %v", err)
	}

	if !assert.NotEmpty(t, endpoints.Subsets) {
		return fmt.Errorf("headlessService endpoints subset cannot be empty")
	}

	if !assert.Len(t, endpoints.Subsets, 1) {
		return fmt.Errorf("headlessService endpoints subset length should be 1 whereas it is %d", len(endpoints.Subsets))
	}

	if !assert.NotEmpty(t, endpoints.Subsets[0].Addresses) {
		return fmt.Errorf("headlessService endpoints addresses list cannot be empty")
	}

	//get the list of endpoint ip's
	var endPointIPs []string
	for _, endpointAddress := range endpoints.Subsets[0].Addresses {
		endPointIPs = append(endPointIPs, endpointAddress.IP)
	}

	headlessService, err := Clientset.CoreV1().Services(string(service.Namespace())).Get(context.TODO(), serviceName, metav1.GetOptions{})
	if !assert.NoError(t, err) {
		return fmt.Errorf("headless service %s not found , Error seen := %v", service.String(), err)
	}

	//get the list of pods which match the headless service selectors
	headlessServiceSelectors := labels.Set(headlessService.Spec.Selector)
	listOptions := metav1.ListOptions{LabelSelector: headlessServiceSelectors.AsSelector().String()}
	pods, err := Clientset.CoreV1().Pods(string(service.Namespace())).List(context.TODO(), listOptions)
	if !assert.NoError(t, err) {
		return fmt.Errorf("cannot get pods matching with headless service selector")
	}

	if !assert.NotEmpty(t, pods) {
		return fmt.Errorf("pods list matching headless service selector cannot be empty")
	}

	//get the list of podIPs matching the headless service selector
	var podIPs []string
	for _, pod := range pods.Items {
		podIPs = append(podIPs, pod.Status.PodIP)
	}

	//compare podIps and headlessService endPoint IPs ...both should match
	if !assert.ElementsMatch(t, podIPs, endPointIPs) {
		return fmt.Errorf("podIPs %v and endPointIps %v do not match", podIPs, endPointIPs)
	}

	return nil
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
	headlessService := clusterHeadLessService{model.NewHeadlessServiceReleaseName(clusterReleaseName)}
	readReplica1 := clusterReadReplica{model.NewReadReplicaReleaseName(clusterReleaseName, 1)}
	readReplica2 := clusterReadReplica{model.NewReadReplicaReleaseName(clusterReleaseName, 2)}
	core1 := clusterCore{model.NewCoreReleaseName(clusterReleaseName, 1)}
	core2 := clusterCore{model.NewCoreReleaseName(clusterReleaseName, 2)}
	core3 := clusterCore{model.NewCoreReleaseName(clusterReleaseName, 3)}
	cores := []clusterCore{core1, core2, core3}
	readReplicas := []clusterReadReplica{readReplica1, readReplica2}

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
		readReplica1,
		readReplica2,
		loadBalancer,
		headlessService,
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

	for _, readReplica := range readReplicas {
		err = run(t, "kubectl", "--namespace", string(readReplica.Name().Namespace()), "rollout", "status", "--watch", "--timeout=180s", "statefulset/"+readReplica.Name().String())
		if !assert.NoError(t, err) {
			return
		}
	}

	t.Logf("Succeeded with setup of '%s'", t.Name())

	subTests, err := clusterTests(loadBalancer.Name(), core1.Name())
	if !assert.NoError(t, err) {
		return
	}
	subTests = append(subTests, headLessServiceTests(headlessService.Name())...)
	subTests = append(subTests, readReplicaTests(readReplica1.Name(), readReplica2.Name(), loadBalancer.Name())...)
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
