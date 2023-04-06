package integration_tests

import (
	"fmt"
	. "github.com/neo4j/helm-charts/internal/helpers"
	"github.com/neo4j/helm-charts/internal/integration_tests/gcloud"
	"github.com/neo4j/helm-charts/internal/model"
	"github.com/neo4j/helm-charts/internal/resources"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestInstallNeo4jClusterInGcloud(t *testing.T) {
	if model.Neo4jEdition != "enterprise" {
		t.Skip()
		return
	}
	t.Parallel()

	var closeables []Closeable
	addCloseable := func(closeableList ...Closeable) {
		for _, closeable := range closeableList {
			closeables = append([]Closeable{closeable}, closeables...)
		}
	}

	err := labelNodes(t)
	addCloseable(func() error {
		return removeLabelFromNodes(t)
	})
	if !assert.NoError(t, err) {
		return
	}

	clusterReleaseName := model.NewReleaseName("cluster-" + TestRunIdentifier)
	loadBalancer := clusterLoadBalancer{model.NewLoadBalancerReleaseName(clusterReleaseName), nil}
	headlessService := clusterHeadLessService{model.NewHeadlessServiceReleaseName(clusterReleaseName), nil}
	readReplica1 := clusterReadReplica{model.NewReadReplicaReleaseName(clusterReleaseName, 1), model.ImagePullSecretArgs}
	readReplica2 := clusterReadReplica{model.NewReadReplicaReleaseName(clusterReleaseName, 2), nil}
	defaultHelmArgs := model.LdapArgs
	core1HelmArgs := append(defaultHelmArgs, model.ImagePullSecretArgs...)
	core1HelmArgs = append(core1HelmArgs, model.NodeSelectorArgs...)
	core2HelmArgs := append(defaultHelmArgs, model.PriorityClassNameArgs...)
	core1 := clusterCore{model.NewCoreReleaseName(clusterReleaseName, 1), core1HelmArgs}
	core2 := clusterCore{model.NewCoreReleaseName(clusterReleaseName, 2), core2HelmArgs}
	core3 := clusterCore{model.NewCoreReleaseName(clusterReleaseName, 3), defaultHelmArgs}
	cores := []clusterCore{core1, core2, core3}
	readReplicas := []clusterReadReplica{readReplica1, readReplica2}

	//t.Cleanup(func() { cleanupTest(t, AsCloseable(closeables)) })
	t.Cleanup(clusterTestCleanup(t, clusterReleaseName, core1, core2, core3, readReplica1, readReplica2))

	t.Logf("Starting setup of '%s'", t.Name())

	closeable, err := prepareK8s(t, clusterReleaseName)
	addCloseable(closeable)
	if !assert.NoError(t, err) {
		return
	}
	cleanPriorityClass, err := createPriorityClass(t, clusterReleaseName)
	addCloseable(cleanPriorityClass)
	if !assert.NoError(t, err) {
		return
	}

	// Install one core synchronously, if all cores are installed simultaneously they run into conflicts all trying to create a -auth secret
	result := core1.Install(t)
	addCloseable(result.Closeable)
	if !assert.NoError(t, result.error) {
		return
	}

	componentsToParallelInstall := []helmComponent{core2, core3, loadBalancer, headlessService}
	closeablesNew, err := performBackgroundInstall(t, componentsToParallelInstall, clusterReleaseName)
	if !assert.NoError(t, err) {
		return
	}
	addCloseable(closeablesNew...)

	for _, core := range cores {
		err = run(t, "kubectl", "--namespace", string(core.Name().Namespace()), "rollout", "status", "--watch", "--timeout=180s", "statefulset/"+core.Name().String())
		if !assert.NoError(t, err) {
			return
		}
	}
	//install read replicas after cores are completed and cluster is formed or else the read replica installation will fail
	componentsToParallelInstall = []helmComponent{readReplica1, readReplica2}
	closeablesNew, err = performBackgroundInstall(t, componentsToParallelInstall, clusterReleaseName)
	if !assert.NoError(t, err) {
		return
	}
	addCloseable(closeablesNew...)

	for _, readReplica := range readReplicas {
		err = run(t, "kubectl", "--namespace", string(readReplica.Name().Namespace()), "rollout", "status", "--watch", "--timeout=180s", "statefulset/"+readReplica.Name().String())
		if !assert.NoError(t, err) {
			return
		}
	}

	t.Logf("Succeeded with setup of '%s'", t.Name())

	subTests, err := clusterTests(loadBalancer.Name())
	if !assert.NoError(t, err) {
		return
	}
	subTests = append(subTests, nodeSelectorTests(core1.Name())...)
	subTests = append(subTests, headLessServiceTests(headlessService.Name())...)
	subTests = append(subTests, readReplicaTests(readReplica1.Name(), readReplica2.Name(), loadBalancer.Name())...)
	runSubTests(t, subTests)

	t.Logf("Succeeded running all tests in '%s'", t.Name())
}

func TestInstallNeo4jClusterWithApocConfigInGcloud(t *testing.T) {
	if model.Neo4jEdition != "enterprise" {
		t.Skip()
		return
	}

	//This test case needs to be parallel in 4.4 since we are using a cluster of 6 nodes
	//Since all our neo4j installtion uses podAntiAffinity which makes only one neo4j installation per node , 6 tends to block the execution of this test case
	// Hence either we increase the number of nodes or make this run sequential
	//t.Parallel()

	var closeables []Closeable
	addCloseable := func(closeableList ...Closeable) {
		for _, closeable := range closeableList {
			closeables = append([]Closeable{closeable}, closeables...)
		}
	}

	clusterReleaseName := model.NewReleaseName("apoc-cluster-" + TestRunIdentifier)
	loadBalancer := clusterLoadBalancer{model.NewLoadBalancerReleaseName(clusterReleaseName), nil}

	apocCustomArgs := append(model.CustomApocImageArgs, resources.ApocClusterTestConfig.HelmArgs()...)
	core1 := clusterCore{model.NewCoreReleaseName(clusterReleaseName, 1), apocCustomArgs}
	core2 := clusterCore{model.NewCoreReleaseName(clusterReleaseName, 2), apocCustomArgs}
	core3 := clusterCore{model.NewCoreReleaseName(clusterReleaseName, 3), apocCustomArgs}
	cores := []clusterCore{core1, core2, core3}

	t.Cleanup(clusterTestCleanupWithoutReadReplicas(t, clusterReleaseName, core1, core2, core3))

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

	componentsToParallelInstall := []helmComponent{core2, core3, loadBalancer}
	closeablesNew, err := performBackgroundInstall(t, componentsToParallelInstall, clusterReleaseName)
	if !assert.NoError(t, err) {
		return
	}
	addCloseable(closeablesNew...)

	for _, core := range cores {
		err = run(t, "kubectl", "--namespace", string(core.Name().Namespace()), "rollout", "status", "--watch", "--timeout=180s", "statefulset/"+core.Name().String())
		if !assert.NoError(t, err) {
			return
		}
	}

	t.Logf("Succeeded with setup of '%s'", t.Name())

	subTests := apocConfigTests(loadBalancer.Name())
	if !assert.NoError(t, err) {
		return
	}
	runSubTests(t, subTests)

	t.Logf("Succeeded running all apoc config tests in '%s'", t.Name())
}

func clusterTestCleanup(t *testing.T, clusterReleaseName model.ReleaseName, core1 clusterCore, core2 clusterCore, core3 clusterCore, readReplica1 clusterReadReplica, readReplica2 clusterReadReplica) func() {
	return func() {
		_ = runAll(t, "helm", [][]string{
			{"uninstall", core1.name.String(), core2.name.String(), core3.name.String(), readReplica1.name.String(), readReplica2.name.String(), "--wait", "--timeout", "3m", "--namespace", string(clusterReleaseName.Namespace())},
			{"uninstall", clusterReleaseName.String() + "-headless", "--wait", "--timeout", "1m", "--namespace", string(clusterReleaseName.Namespace())},
		}, false)
		_ = runAll(t, "kubectl", [][]string{
			{"delete", "pvc", fmt.Sprintf("%s-pvc", core1.name.String()), "--namespace", string(clusterReleaseName.Namespace()), "--ignore-not-found"},
			{"delete", "pvc", fmt.Sprintf("%s-pvc", core2.name.String()), "--namespace", string(clusterReleaseName.Namespace()), "--ignore-not-found"},
			{"delete", "pvc", fmt.Sprintf("%s-pvc", core3.name.String()), "--namespace", string(clusterReleaseName.Namespace()), "--ignore-not-found"},
			{"delete", "pvc", fmt.Sprintf("%s-pvc", readReplica1.name.String()), "--namespace", string(clusterReleaseName.Namespace()), "--ignore-not-found"},
			{"delete", "pvc", fmt.Sprintf("%s-pvc", readReplica2.name.String()), "--namespace", string(clusterReleaseName.Namespace()), "--ignore-not-found"},
			{"delete", "pv", fmt.Sprintf("%s-pv", core1.name.String()), "--ignore-not-found"},
			{"delete", "pv", fmt.Sprintf("%s-pv", core2.name.String()), "--ignore-not-found"},
			{"delete", "pv", fmt.Sprintf("%s-pv", core3.name.String()), "--ignore-not-found"},
			{"delete", "pv", fmt.Sprintf("%s-pv", readReplica1.name.String()), "--ignore-not-found"},
			{"delete", "pv", fmt.Sprintf("%s-pv", readReplica2.name.String()), "--ignore-not-found"},
		}, false)
		_ = runAll(t, "gcloud", [][]string{
			{"compute", "disks", "delete", fmt.Sprintf("neo4j-data-disk-%s", core1.name), "--zone=" + string(gcloud.CurrentZone()), "--project=" + string(gcloud.CurrentProject())},
			{"compute", "disks", "delete", fmt.Sprintf("neo4j-data-disk-%s", core2.name), "--zone=" + string(gcloud.CurrentZone()), "--project=" + string(gcloud.CurrentProject())},
			{"compute", "disks", "delete", fmt.Sprintf("neo4j-data-disk-%s", core3.name), "--zone=" + string(gcloud.CurrentZone()), "--project=" + string(gcloud.CurrentProject())},
			{"compute", "disks", "delete", fmt.Sprintf("neo4j-data-disk-%s", readReplica1.name), "--zone=" + string(gcloud.CurrentZone()), "--project=" + string(gcloud.CurrentProject())},
			{"compute", "disks", "delete", fmt.Sprintf("neo4j-data-disk-%s", readReplica2.name), "--zone=" + string(gcloud.CurrentZone()), "--project=" + string(gcloud.CurrentProject())},
		}, false)
		_ = runAll(t, "kubectl", [][]string{
			{"delete", "namespace", string(clusterReleaseName.Namespace()), "--ignore-not-found", "--force", "--grace-period=0"},
			{"delete", "priorityClass", "high-priority", "--force", "--grace-period=0"},
		}, false)
		_ = removeLabelFromNodes(t)
	}
}

func clusterTestCleanupWithoutReadReplicas(t *testing.T, clusterReleaseName model.ReleaseName, core1 clusterCore, core2 clusterCore, core3 clusterCore) func() {
	return func() {
		_ = runAll(t, "helm", [][]string{
			{"uninstall", core1.name.String(), core2.name.String(), core3.name.String(), "--wait", "--timeout", "3m", "--namespace", string(clusterReleaseName.Namespace())},
			{"uninstall", clusterReleaseName.String() + "-headless", "--wait", "--timeout", "1m", "--namespace", string(clusterReleaseName.Namespace())},
		}, false)
		_ = runAll(t, "kubectl", [][]string{
			{"delete", "pvc", fmt.Sprintf("%s-pvc", core1.name.String()), "--namespace", string(clusterReleaseName.Namespace()), "--ignore-not-found"},
			{"delete", "pvc", fmt.Sprintf("%s-pvc", core2.name.String()), "--namespace", string(clusterReleaseName.Namespace()), "--ignore-not-found"},
			{"delete", "pvc", fmt.Sprintf("%s-pvc", core3.name.String()), "--namespace", string(clusterReleaseName.Namespace()), "--ignore-not-found"},
			{"delete", "pv", fmt.Sprintf("%s-pv", core1.name.String()), "--ignore-not-found"},
			{"delete", "pv", fmt.Sprintf("%s-pv", core2.name.String()), "--ignore-not-found"},
			{"delete", "pv", fmt.Sprintf("%s-pv", core3.name.String()), "--ignore-not-found"},
		}, false)
		_ = runAll(t, "gcloud", [][]string{
			{"compute", "disks", "delete", fmt.Sprintf("neo4j-data-disk-%s", core1.name), "--zone=" + string(gcloud.CurrentZone()), "--project=" + string(gcloud.CurrentProject())},
			{"compute", "disks", "delete", fmt.Sprintf("neo4j-data-disk-%s", core2.name), "--zone=" + string(gcloud.CurrentZone()), "--project=" + string(gcloud.CurrentProject())},
			{"compute", "disks", "delete", fmt.Sprintf("neo4j-data-disk-%s", core3.name), "--zone=" + string(gcloud.CurrentZone()), "--project=" + string(gcloud.CurrentProject())},
		}, false)
		_ = runAll(t, "kubectl", [][]string{
			{"delete", "namespace", string(clusterReleaseName.Namespace()), "--ignore-not-found", "--force", "--grace-period=0"},
			{"delete", "priorityClass", "high-priority", "--force", "--grace-period=0"},
		}, false)
	}
}
