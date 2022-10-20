package integration_tests

import (
	"context"
	"fmt"
	"github.com/hashicorp/go-multierror"
	. "github.com/neo4j/helm-charts/internal/helpers"
	"github.com/neo4j/helm-charts/internal/model"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"os/exec"
	"strings"
	"testing"
)

// labelNodes labels all the node with testLabel=<number>
func labelNodes(t *testing.T) error {

	var errors *multierror.Error
	nodesList, err := getNodesList()
	if err != nil {
		return err
	}

	for index, node := range nodesList.Items {
		labelName := fmt.Sprintf("testLabel=%d", index+1)
		err = run(t, "kubectl", "label", "nodes", node.ObjectMeta.Name, labelName)
		if err != nil {
			errors = multierror.Append(errors, err)
			t.Logf("Node Label failed for %s: %v", node.ObjectMeta.Name, err)
		}
	}

	return errors.ErrorOrNil()
}

// removeLabelFromNodes removes label testLabel from all the nodes added via labelNodes func
func removeLabelFromNodes(t *testing.T) error {

	var errors *multierror.Error
	nodesList, err := getNodesList()
	if err != nil {
		return err
	}

	for _, node := range nodesList.Items {
		err = run(t, "kubectl", "label", "nodes", node.ObjectMeta.Name, "testLabel-")
		if err != nil {
			errors = multierror.Append(errors, err)
			t.Logf("Node Label removal failed for %s: %v", node.ObjectMeta.Name, err)
		}
	}

	return errors.ErrorOrNil()
}

// clusterTests contains all the tests related to cluster
func clusterTests(loadBalancerName model.ReleaseName) ([]SubTest, error) {

	subTests := []SubTest{

		{name: "Check Cluster Core Logs Format", test: func(t *testing.T) {
			t.Parallel()
			assert.NoError(t, CheckLogsFormat(t, loadBalancerName), "Cluster core logs format should be in JSON")
		}},
		{name: "ImagePullSecret tests", test: func(t *testing.T) {
			t.Parallel()
			assert.NoError(t, imagePullSecretTests(t, loadBalancerName), "Perform ImagePullSecret Tests")
		}},
		{name: "Check PriorityClassName", test: func(t *testing.T) {
			t.Parallel()
			assert.NoError(t, checkPriorityClassName(t, loadBalancerName), "priorityClassName should match")
		}},
		{name: "Check Cluster Password failure", test: func(t *testing.T) {
			t.Parallel()
			assert.NoError(t, checkClusterCorePasswordFailure(t), "Cluster core installation should not succeed with incorrect password")
		}},
		{name: "Check K8s", test: func(t *testing.T) {
			assert.NoError(t, checkK8s(t, loadBalancerName), "Neo4j Config check should succeed")
		}},
		{name: "Create Node", test: func(t *testing.T) {
			assert.NoError(t, createNode(t, loadBalancerName), "Create Node should succeed")
		}},
		{name: "Count Nodes", test: func(t *testing.T) {
			assert.NoError(t, checkNodeCount(t, loadBalancerName), "Count Nodes should succeed")
		}},
		{name: "Database Creation Tests", test: func(t *testing.T) {
			assert.NoError(t, databaseCreationTests(t, loadBalancerName, "customers"), "Creates \"customer\" database and checks for its existence")
		}},
	}
	return subTests, nil
}

// CheckLogsFormat checks whether the neo4j logs are in json format or not
func CheckLogsFormat(t *testing.T, releaseName model.ReleaseName) error {

	stdout, stderr, err := ExecInPod(releaseName, []string{"cat", "/logs/neo4j.log"})
	if !assert.NoError(t, err) {
		return fmt.Errorf("error seen while executing command `cat /logs/neo4j.log' ,\n err :- %v", err)
	}
	if !assert.Contains(t, stdout, ",\"level\":\"INFO\",\"category\":\"c.n.s.e.EnterpriseBootstrapper\",\"message\":\"Command expansion is explicitly enabled for configuration\"}") {
		return fmt.Errorf("foes not contain the required json format\n stdout := %s", stdout)
	}
	if !assert.Len(t, stderr, 0) {
		return fmt.Errorf("stderr found while checking logs \n stderr := %s", stderr)
	}
	return nil
}

// imagePullSecretTests runs tests related to imagePullSecret feature
func imagePullSecretTests(t *testing.T, name model.ReleaseName) error {
	t.Run("Check cluster core has imagePullSecret image", func(t *testing.T) {
		t.Parallel()
		assert.NoError(t, checkCoreImageName(t, name), "Core-1 image name should match with customImage")
	})
	t.Run("Check imagePullSecret \"demo\" is created", func(t *testing.T) {
		t.Parallel()
		assert.NoError(t, checkImagePullSecret(t, name), "ImagePullSecret named \"demo\" should be present")
	})
	return nil
}

// nodeSelectorTests runs tests related to nodeSelector feature
func nodeSelectorTests(name model.ReleaseName) []SubTest {
	return []SubTest{
		{name: fmt.Sprintf("Check cluster core 1 is assigned with label %s", model.NodeSelectorLabel), test: func(t *testing.T) {
			t.Parallel()
			assert.NoError(t, checkNodeSelectorLabel(t, name, model.NodeSelectorLabel), fmt.Sprintf("Core-1 Pod should be deployed on node with label %s", model.NodeSelectorLabel))
		}},
	}
}

// databaseCreationTests creates a database against a cluster and checks if its created or not
func databaseCreationTests(t *testing.T, loadBalancerName model.ReleaseName, dataBaseName string) error {
	t.Run("Create Database customers", func(t *testing.T) {
		assert.NoError(t, createDatabase(t, loadBalancerName, dataBaseName), "Creates database")
	})
	t.Run("Check Database customers exists", func(t *testing.T) {
		assert.NoError(t, checkDataBaseExists(t, loadBalancerName, dataBaseName), "Checks if database exists or not")
	})
	return nil
}

// checkPriorityClassName checks the priorityClassName is set to the pod or not
func checkPriorityClassName(t *testing.T, releaseName model.ReleaseName) error {

	pods, err := getAllPods(releaseName.Namespace())
	if !assert.NoError(t, err) {
		return err
	}
	for _, pod := range pods.Items {
		if strings.Contains(pod.Name, "core-2") {
			if !assert.Equal(t, model.PriorityClassName, pod.Spec.PriorityClassName) {
				return fmt.Errorf("priorityClassName %s not matching with %s", pod.Spec.PriorityClassName, model.PriorityClassName)
			}
			break
		}
	}
	return nil
}

// checkCoreImageName checks whether core-1 image is matching with imagePullSecret image or not
func checkCoreImageName(t *testing.T, releaseName model.ReleaseName) error {

	pods, err := getAllPods(releaseName.Namespace())
	if !assert.NoError(t, err) {
		return err
	}
	for _, pod := range pods.Items {
		if strings.Contains(pod.Name, "core-1") {
			container := pod.Spec.Containers[0]
			if !assert.Equal(t, container.Image, model.ImagePullSecretCustomImageName) {
				return fmt.Errorf("container image %s not matching with imagePullSecet customImage %s", container.Image, model.ImagePullSecretCustomImageName)
			}
			break
		}
	}
	return nil
}

// checkNodeSelectorLabel checks whether the given pod is associated with the correct node or not
func checkNodeSelectorLabel(t *testing.T, releaseName model.ReleaseName, labelName string) error {

	nodeSelectorNode, err := getNodeWithLabel(labelName)
	if !assert.NoError(t, err) {
		return err
	}
	pod, err := getSpecificPod(releaseName.Namespace(), releaseName.PodName())
	if !assert.NoError(t, err) {
		return fmt.Errorf("error while fetching pod list \n %v", err)
	}
	if !assert.Equal(t, nodeSelectorNode.Name, pod.Spec.NodeName) {
		return fmt.Errorf("pod %s is not scheduled on the correct node %s", pod.Spec.NodeName, nodeSelectorNode.Name)
	}

	return nil
}

// checkImagePullSecret checks whether a secret of type docker-registry is created or not
func checkImagePullSecret(t *testing.T, releaseName model.ReleaseName) error {

	secret, err := getSpecificSecret(releaseName.Namespace(), "demo")
	if !assert.NoError(t, err) {
		return fmt.Errorf("No secret found for the provided imagePullSecret \n %v", err)
	}
	if !assert.Equal(t, secret.Name, "demo") {
		return fmt.Errorf("imagePullSecret name %s does not match with demo", secret.Name)
	}
	return nil
}

// headLessServiceTests contains all the tests related to headless service
func headLessServiceTests(headlessService model.ReleaseName) []SubTest {
	return []SubTest{
		{name: "Check Headless Service Configuration", test: func(t *testing.T) {
			t.Parallel()
			assert.NoError(t, checkHeadlessServiceConfiguration(t, headlessService), "Checks Headless Service configuration")
		}},
		{name: "Check Headless Service Endpoints", test: func(t *testing.T) {
			t.Parallel()
			assert.NoError(t, checkHeadlessServiceEndpoints(t, headlessService), "headless service endpoints should be equal to the cluster core created")
		}},
	}
}

// apocConfigTests contains all the tests related to apoc configs
func apocConfigTests(releaseName model.ReleaseName) []SubTest {
	return []SubTest{
		{name: "Execute apoc query", test: func(t *testing.T) {
			t.Parallel()
			assert.NoError(t, checkApocConfig(t, releaseName), "Apoc Cypher Query failing to execute")
		}},
	}
}

// checkClusterCorePasswordFailure checks if a cluster core is failing on installation or not with an incorrect password
func checkClusterCorePasswordFailure(t *testing.T) error {
	//creating a sample cluster core definition (which is not supposed to get installed)
	clusterReleaseName := model.NewReleaseName("cluster-" + TestRunIdentifier)
	core := clusterCore{model.NewCoreReleaseName(clusterReleaseName, 4), nil}
	releaseName := core.Name()
	// we are not using the customized run() func here since we need to assert the error received on stdout
	//(present in out variable and not in err)
	out, err := exec.Command(
		"helm",
		model.BaseHelmCommand("install", releaseName, model.HelmChart, model.Neo4jEdition, "--set", "neo4j.password=my-password", "--set", "neo4j.name="+model.DefaultNeo4jName)...).CombinedOutput()
	if !assert.Error(t, err) {
		return fmt.Errorf("helm install should fail without the default password")
	}
	if !assert.Contains(t, string(out), "The desired password does not match the password stored in the Kubernetes Secret") {
		return fmt.Errorf("error thrown on password failure is different , err := %s", string(out))
	}
	return nil
}

func checkK8s(t *testing.T, name model.ReleaseName) error {
	t.Run("check pods", func(t *testing.T) {
		t.Parallel()
		assert.NoError(t, checkPods(t, name))
	})
	t.Run("check lb", func(t *testing.T) {
		t.Parallel()
		assert.NoError(t, checkLoadBalancerService(t, name, model.DefaultClusterSize))
	})
	return nil
}

// checkLoadBalancerService checks whether the loadbalancer exists or not
// It also checks that the number of endpoints should match with the given number of expected endpoints
func checkLoadBalancerService(t *testing.T, name model.ReleaseName, expectedEndPoints int) error {

	serviceName := fmt.Sprintf("%s-lb-neo4j", model.DefaultNeo4jName)
	lbService, err := Clientset.CoreV1().Services(string(name.Namespace())).Get(context.TODO(), serviceName, metav1.GetOptions{})
	if !assert.NoError(t, err) {
		return fmt.Errorf("loadbalancer service %s not found , Error seen := %v", serviceName, err)
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

// checkPods checks for the number of pods which should be 5 (3 cluster core + 2 read replica)
func checkPods(t *testing.T, name model.ReleaseName) error {
	pods, err := getAllPods(name.Namespace())
	if !assert.NoError(t, err) {
		return err
	}

	if !assert.Len(t, pods.Items, model.DefaultClusterSize) {
		return fmt.Errorf("number of pods should be %d", model.DefaultClusterSize)
	}
	for _, pod := range pods.Items {
		if assert.Contains(t, pod.Labels, "app") {
			if !assert.Equal(t, model.DefaultNeo4jName, pod.Labels["app"]) {
				return fmt.Errorf("pod should have label app=%s , found app=%s", model.DefaultNeo4jName, pod.Labels["app"])
			}
		}
	}

	return nil
}

// checkNeo4jLogsForAnyErrors checks whether neo4j.log and debug.log contain any errors or not
func checkNeo4jLogsForAnyErrors(t *testing.T, name model.ReleaseName) error {
	cmd := []string{
		"bash",
		"-c",
		"cat /logs/neo4j.log /logs/debug.log",
	}

	stdout, stderr, err := ExecInPod(name, cmd)
	if !assert.NoError(t, err) {
		return err
	}
	if !assert.Len(t, stderr, 0) {
		return fmt.Errorf("stderr found \n %s", stderr)
	}
	//commenting this one out, the issue is reported to kernel team (card created)
	//https://trello.com/c/z0g4J7om/7548-neo4j-447-startup-error-seen-in-community-edition
	// Should be uncommented or removed based on the findings in the above card
	if !assert.NotContains(t, stdout, " ERROR [") {
		return fmt.Errorf("Contains error logs \n%s", stdout)
	}
	return nil
}

// checkHeadlessServiceConfiguration checks whether the provided service is headless service or not
func checkHeadlessServiceConfiguration(t *testing.T, service model.ReleaseName) error {

	serviceName := fmt.Sprintf("%s-headless", model.DefaultNeo4jName)
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

// checkHeadlessServiceEndpoints checks whether the headless endpoints have the cluster cores or not
// By default , headless service includes cluster core only and no read replicas
func checkHeadlessServiceEndpoints(t *testing.T, service model.ReleaseName) error {

	serviceName := fmt.Sprintf("%s-headless", model.DefaultNeo4jName)

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

func performBackgroundInstall(t *testing.T, componentsToParallelInstall []helmComponent, clusterReleaseName model.ReleaseName) ([]Closeable, error) {

	results := make(chan parallelResult)
	for _, component := range componentsToParallelInstall {
		go func(comp helmComponent) {
			var parallelResult = parallelResult{
				Closeable: nil,
				error:     fmt.Errorf("illegal state: background install did not take place for %s in %s", comp.Name(), clusterReleaseName),
			}
			defer func() { results <- parallelResult }()
			parallelResult = comp.Install(t)
		}(component)
	}

	var closeables []Closeable
	var combinedError error
	for i := 0; i < len(componentsToParallelInstall); i++ {
		result := <-results
		closeables = append(closeables, result.Closeable)
		if result.error != nil {
			combinedError = CombineErrors(combinedError, result.error)
		}
	}
	if !assert.NoError(t, combinedError) {
		return closeables, combinedError
	}
	return closeables, nil
}
