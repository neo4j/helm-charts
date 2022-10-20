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
	defaultHelmArgs := []string{}
	defaultHelmArgs = append(defaultHelmArgs, model.DefaultNeo4jNameArg...)
	headlessService := clusterHeadLessService{model.NewHeadlessServiceReleaseName(clusterReleaseName), defaultHelmArgs}
	defaultHelmArgs = append(defaultHelmArgs, model.DefaultClusterSizeArg...)
	core1HelmArgs := append(defaultHelmArgs, model.ImagePullSecretArgs...)
	core1HelmArgs = append(core1HelmArgs, model.NodeSelectorArgs...)
	core2HelmArgs := append(defaultHelmArgs, model.PriorityClassNameArgs...)
	core1 := clusterCore{model.NewCoreReleaseName(clusterReleaseName, 1), core1HelmArgs}
	core2 := clusterCore{model.NewCoreReleaseName(clusterReleaseName, 2), core2HelmArgs}
	core3 := clusterCore{model.NewCoreReleaseName(clusterReleaseName, 3), defaultHelmArgs}
	cores := []clusterCore{core1, core2, core3}

	//t.Cleanup(func() { cleanupTest(t, AsCloseable(closeables)) })
	t.Cleanup(clusterTestCleanup(t, clusterReleaseName, core1, core2, core3))

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

	componentsToParallelInstall := []helmComponent{core2, core3, headlessService}
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
	addCloseable(closeablesNew...)

	t.Logf("Succeeded with setup of '%s'", t.Name())

	subTests, err := clusterTests(core1.Name())
	if !assert.NoError(t, err) {
		return
	}
	subTests = append(subTests, nodeSelectorTests(core1.Name())...)
	subTests = append(subTests, headLessServiceTests(headlessService.Name())...)
	runSubTests(t, subTests)

	t.Logf("Succeeded running all tests in '%s'", t.Name())
}

func TestInstallNeo4jClusterWithApocConfigInGcloud(t *testing.T) {
	if model.Neo4jEdition != "enterprise" {
		t.Skip()
		return
	}

	//if we make this in parallel with the other cluster tests , it will fail
	// we need to wait for this cluster test to complete so that the other cluster test can complete
	//t.Parallel()

	var closeables []Closeable
	addCloseable := func(closeableList ...Closeable) {
		for _, closeable := range closeableList {
			closeables = append([]Closeable{closeable}, closeables...)
		}
	}

	clusterReleaseName := model.NewReleaseName("apoc-cluster-" + TestRunIdentifier)
	defaultHelmArgs := []string{}
	defaultHelmArgs = append(defaultHelmArgs, model.DefaultNeo4jNameArg...)
	defaultHelmArgs = append(defaultHelmArgs, model.DefaultClusterSizeArg...)
	defaultHelmArgs = append(defaultHelmArgs, resources.ApocClusterTestConfig.HelmArgs()...)
	//defaultHelmArgs = append(defaultHelmArgs, model.CustomApocImageArgs...)
	core1 := clusterCore{model.NewCoreReleaseName(clusterReleaseName, 1), defaultHelmArgs}
	core2 := clusterCore{model.NewCoreReleaseName(clusterReleaseName, 2), defaultHelmArgs}
	core3 := clusterCore{model.NewCoreReleaseName(clusterReleaseName, 3), defaultHelmArgs}
	cores := []clusterCore{core1, core2, core3}

	t.Cleanup(clusterTestCleanup(t, clusterReleaseName, core1, core2, core3))

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

	componentsToParallelInstall := []helmComponent{core2, core3}
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

	subTests := apocConfigTests(clusterReleaseName)
	if !assert.NoError(t, err) {
		return
	}
	runSubTests(t, subTests)

	t.Logf("Succeeded running all apoc config tests in '%s'", t.Name())
}

func clusterTestCleanup(t *testing.T, clusterReleaseName model.ReleaseName, core1 clusterCore, core2 clusterCore, core3 clusterCore) func() {
	return func() {
		runAll(t, "helm", [][]string{
			{"uninstall", core1.name.String(), "--wait", "--timeout", "1m", "--namespace", string(clusterReleaseName.Namespace())},
			{"uninstall", core2.name.String(), "--wait", "--timeout", "1m", "--namespace", string(clusterReleaseName.Namespace())},
			{"uninstall", core3.name.String(), "--wait", "--timeout", "1m", "--namespace", string(clusterReleaseName.Namespace())},
			{"uninstall", clusterReleaseName.String() + "-headless", "--wait", "--timeout", "1m", "--namespace", string(clusterReleaseName.Namespace())},
		}, false)
		runAll(t, "kubectl", [][]string{
			{"delete", "pvc", fmt.Sprintf("%s-pvc", core1.name.String()), "--namespace", string(clusterReleaseName.Namespace()), "--ignore-not-found"},
			{"delete", "pvc", fmt.Sprintf("%s-pvc", core2.name.String()), "--namespace", string(clusterReleaseName.Namespace()), "--ignore-not-found"},
			{"delete", "pvc", fmt.Sprintf("%s-pvc", core3.name.String()), "--namespace", string(clusterReleaseName.Namespace()), "--ignore-not-found"},
			{"delete", "pv", fmt.Sprintf("%s-pv", core1.name.String()), "--ignore-not-found"},
			{"delete", "pv", fmt.Sprintf("%s-pv", core2.name.String()), "--ignore-not-found"},
			{"delete", "pv", fmt.Sprintf("%s-pv", core3.name.String()), "--ignore-not-found"},
		}, false)
		runAll(t, "gcloud", [][]string{
			{"compute", "disks", "delete", fmt.Sprintf("neo4j-data-disk-%s", core1.name), "--zone=" + string(gcloud.CurrentZone()), "--project=" + string(gcloud.CurrentProject())},
			{"compute", "disks", "delete", fmt.Sprintf("neo4j-data-disk-%s", core2.name), "--zone=" + string(gcloud.CurrentZone()), "--project=" + string(gcloud.CurrentProject())},
			{"compute", "disks", "delete", fmt.Sprintf("neo4j-data-disk-%s", core3.name), "--zone=" + string(gcloud.CurrentZone()), "--project=" + string(gcloud.CurrentProject())},
		}, false)
		runAll(t, "kubectl", [][]string{
			{"delete", "namespace", string(clusterReleaseName.Namespace()), "--ignore-not-found", "--force", "--grace-period=0"},
			{"delete", "priorityClass", "high-priority", "--force", "--grace-period=0"},
		}, false)
	}
}
