package integration_tests

import (
	"fmt"
	"testing"
	"time"

	. "github.com/neo4j/helm-charts/internal/helpers"
	"github.com/neo4j/helm-charts/internal/integration_tests/gcloud"
	"github.com/neo4j/helm-charts/internal/model"
	"github.com/neo4j/helm-charts/internal/resources"
	"github.com/neo4j/helm-charts/internal/testutil/timeouts"
	"github.com/stretchr/testify/assert"
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
	clusterReleaseName := model.NewReleaseName("cluster-" + TestNamespace(t))
	namespace := string(clusterReleaseName.Namespace())
	err := labelNodes(t, namespace)
	addCloseable(func() error {
		return removeLabelFromNodes(t)
	})
	if !assert.NoError(t, err) {
		return
	}

	defaultHelmArgs := []string{}
	defaultHelmArgs = append(defaultHelmArgs, model.DefaultNeo4jNameArg...)
	headlessService := clusterHeadLessService{model.NewHeadlessServiceReleaseName(clusterReleaseName), defaultHelmArgs}
	defaultHelmArgs = append(defaultHelmArgs, model.DefaultClusterSizeArg...)
	defaultHelmArgs = append(defaultHelmArgs, model.LdapArgs...)
	core1HelmArgs := append(defaultHelmArgs, model.ImagePullSecretArgs...)
	core1HelmArgs = append(core1HelmArgs, model.NodeSelectorArgs(namespace)...)
	core2HelmArgs := append(defaultHelmArgs, model.PriorityClassNameArgs(namespace)...)
	core3HelmArgs := append(defaultHelmArgs, model.EnableServerArgs()...)
	core1 := clusterCore{model.NewCoreReleaseName(clusterReleaseName, 1), core1HelmArgs}
	core2 := clusterCore{model.NewCoreReleaseName(clusterReleaseName, 2), core2HelmArgs}
	core3 := clusterCore{model.NewCoreReleaseName(clusterReleaseName, 3), core3HelmArgs}
	cores := []clusterCore{core1, core2, core3}

	t.Cleanup(clusterTestCleanup(t, clusterReleaseName, core1, core2, core3, true))

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
	// Install one core synchronously, if all cores are installed simultaneously they run into conflicts all trying to create a -auth secret.
	// Re-apply the nodeSelector label right before install so a GKE node replacement between labelNodes() and this install doesn't leave core-1 unschedulable.
	if err := ensureNodeLabel(t, fmt.Sprintf("%s-1", namespace)); !assert.NoError(t, err) {
		return
	}
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
		err = run(t, "kubectl", "--namespace", string(core.Name().Namespace()), "rollout", "status", "--watch", timeouts.KubectlRollout(), "statefulset/"+core.Name().String())
		if !assert.NoError(t, err) {
			return
		}
		// rollout status returns as soon as the partition-update count is
		// satisfied for RollingUpdate StatefulSets — it does NOT wait for
		// Ready. Follow up with an explicit Ready wait so subsequent sub-tests
		// don't fire against a pod that's still initializing (or worse,
		// permanently Pending after a node replacement).
		err = run(t, "kubectl", "--namespace", string(core.Name().Namespace()), "wait", "--for=condition=Ready", "pod/"+core.Name().PodName(), timeouts.KubectlPodReady())
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
	t.Parallel()

	var closeables []Closeable
	addCloseable := func(closeableList ...Closeable) {
		for _, closeable := range closeableList {
			closeables = append([]Closeable{closeable}, closeables...)
		}
	}

	clusterReleaseName := model.NewReleaseName("apoc-cluster-" + TestNamespace(t))
	defaultHelmArgs := []string{}
	defaultHelmArgs = append(defaultHelmArgs, model.DefaultNeo4jNameArg...)
	defaultHelmArgs = append(defaultHelmArgs, model.DefaultClusterSizeArg...)
	defaultHelmArgs = append(defaultHelmArgs, resources.ApocClusterTestConfig.HelmArgs()...)
	//defaultHelmArgs = append(defaultHelmArgs, model.CustomApocImageArgs...)
	core1 := clusterCore{model.NewCoreReleaseName(clusterReleaseName, 1), defaultHelmArgs}
	core2 := clusterCore{model.NewCoreReleaseName(clusterReleaseName, 2), defaultHelmArgs}
	core3 := clusterCore{model.NewCoreReleaseName(clusterReleaseName, 3), defaultHelmArgs}
	cores := []clusterCore{core1, core2, core3}

	t.Cleanup(clusterTestCleanup(t, clusterReleaseName, core1, core2, core3, false))

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
		err = run(t, "kubectl", "--namespace", string(core.Name().Namespace()), "rollout", "status", "--watch", timeouts.KubectlRollout(), "statefulset/"+core.Name().String())
		if !assert.NoError(t, err) {
			return
		}
		// rollout status returns as soon as the partition-update count is
		// satisfied for RollingUpdate StatefulSets — it does NOT wait for
		// Ready. Follow up with an explicit Ready wait so subsequent sub-tests
		// don't fire against a pod that's still initializing (or worse,
		// permanently Pending after a node replacement).
		err = run(t, "kubectl", "--namespace", string(core.Name().Namespace()), "wait", "--for=condition=Ready", "pod/"+core.Name().PodName(), timeouts.KubectlPodReady())
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

func clusterTestCleanup(t *testing.T, clusterReleaseName model.ReleaseName, core1 clusterCore, core2 clusterCore, core3 clusterCore, removeLabels bool) func() {
	return func() {
		namespace := string(clusterReleaseName.Namespace())

		err := run(t, "kubectl", "get", "namespace", namespace)
		if err == nil {
			for _, core := range []clusterCore{core1, core2, core3} {
				stsErr := run(t, "kubectl", "get", "statefulset", core.name.String(), "--namespace", namespace)
				if stsErr == nil {
					_ = runAll(t, "kubectl", [][]string{
						{"scale", "statefulset", core.name.String(), "--namespace", namespace, "--replicas=0"},
					}, false)
				}
			}

			waitForPodsTerminated(t, namespace, 60*time.Second)

			// Force-delete any stragglers (waitForPodsTerminated already does this on timeout,
			// but calling it again is cheap and guards against pods created between scale-down
			// and uninstall). This is the critical fix: if a pod is stuck Terminating,
			// `helm uninstall --wait` blocks for its full --timeout, which is how we ran
			// out of the Go test deadline previously.
			_ = run(t, "kubectl", "delete", "pod", "--all", "--namespace", namespace, "--force", "--grace-period=0", "--ignore-not-found")

			_ = runAll(t, "helm", [][]string{
				{"uninstall", core1.name.String(), core2.name.String(), core3.name.String(), "--timeout", "30s", "--namespace", namespace},
				{"uninstall", clusterReleaseName.String() + "-headless", "--timeout", "30s", "--namespace", namespace},
			}, false)

			_ = runAll(t, "kubectl", [][]string{
				{"delete", "pvc", "--all", "--namespace", namespace, "--force", "--grace-period=0", "--ignore-not-found"},
			}, false)

			pvDeleteCmds := [][]string{}
			for _, core := range []clusterCore{core1, core2, core3} {
				pvDeleteCmds = append(pvDeleteCmds, []string{
					"delete", "pv", fmt.Sprintf("%s-pv", core.name.String()), "--force", "--grace-period=0", "--ignore-not-found",
				})
			}
			_ = runAll(t, "kubectl", pvDeleteCmds, false)

			_ = runAll(t, "kubectl", [][]string{
				{"delete", "namespace", namespace, "--force", "--grace-period=0", "--ignore-not-found"},
			}, false)
		}

		_ = runAll(t, "gcloud", [][]string{
			{"compute", "disks", "delete", fmt.Sprintf("neo4j-data-disk-%s", core1.name), "--zone=" + string(gcloud.CurrentZone()), "--project=" + string(gcloud.CurrentProject()), "--quiet"},
			{"compute", "disks", "delete", fmt.Sprintf("neo4j-data-disk-%s", core2.name), "--zone=" + string(gcloud.CurrentZone()), "--project=" + string(gcloud.CurrentProject()), "--quiet"},
			{"compute", "disks", "delete", fmt.Sprintf("neo4j-data-disk-%s", core3.name), "--zone=" + string(gcloud.CurrentZone()), "--project=" + string(gcloud.CurrentProject()), "--quiet"},
		}, false)

		if removeLabels {
			_ = removeLabelFromNodes(t)
		}
	}
}
