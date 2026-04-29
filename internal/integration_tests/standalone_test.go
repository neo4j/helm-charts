package integration_tests

import (
	"fmt"
	"testing"
	"time"

	"github.com/neo4j/helm-charts/internal/integration_tests/gcloud"
	"github.com/neo4j/helm-charts/internal/model"
	"github.com/neo4j/helm-charts/internal/resources"
	"github.com/stretchr/testify/assert"
)

// Install Neo4j on the provided GKE K8s cluster and then run the tests from the table above using it
func TestInstallStandaloneOnGCloudK8s(t *testing.T) {
	releaseName := model.NewReleaseName("install-" + TestRunIdentifier)
	chart := model.Neo4jHelmChartCommunityAndEnterprise

	t.Parallel()
	t.Logf("Starting setup of '%s'", t.Name())
	defaultHelmArgs := []string{}
	defaultHelmArgs = append(defaultHelmArgs, model.DefaultNeo4jNameArg...)
	defaultHelmArgs = append(defaultHelmArgs, resources.TestAntiAffinityRule.HelmArgs()...)
	defaultHelmArgs = append(defaultHelmArgs, resources.GdsStandaloneTest.HelmArgs()...)
	_, err := installNeo4j(t, releaseName, chart, defaultHelmArgs...)
	t.Cleanup(standaloneCleanup(t, releaseName))

	if !assert.NoError(t, err) {
		t.Logf("%#v", err)
		return
	}

	t.Logf("Succeeded with setup of '%s'", t.Name())

	subTests, err := k8sTests(releaseName, chart)
	if !assert.NoError(t, err) {
		return
	}
	runSubTests(t, subTests)
}

func standaloneCleanup(t *testing.T, releaseName model.ReleaseName) func() {
	return func() {
		namespace := string(releaseName.Namespace())

		err := run(t, "kubectl", "get", "namespace", namespace)
		if err == nil {
			_ = run(t, "kubectl", "get", "statefulset", releaseName.String(), "--namespace", namespace)
			if err == nil {
				_ = runAll(t, "kubectl", [][]string{
					{"scale", "statefulset", releaseName.String(), "--namespace", namespace, "--replicas=0"},
				}, false)

				time.Sleep(30 * time.Second)
			}

			_ = runAll(t, "helm", [][]string{
				{"uninstall", releaseName.String(), "--wait", "--timeout", "3m", "--namespace", namespace},
			}, false)

			time.Sleep(10 * time.Second)

			_ = runAll(t, "kubectl", [][]string{
				{"delete", "statefulset", releaseName.String(), "--namespace", namespace, "--force", "--grace-period=0", "--ignore-not-found"},
				{"delete", "pod", "--all", "--namespace", namespace, "--force", "--grace-period=0", "--ignore-not-found"},
				{"delete", "pvc", "--all", "--namespace", namespace, "--force", "--grace-period=0", "--ignore-not-found"},
			}, false)

			_ = runAll(t, "kubectl", [][]string{
				{"delete", "pv", "--all", "--force", "--grace-period=0", "--ignore-not-found"},
			}, false)

			_ = runAll(t, "kubectl", [][]string{
				{"delete", "namespace", namespace, "--force", "--grace-period=0", "--ignore-not-found"},
			}, false)
		}

		_ = runAll(t, "gcloud", [][]string{
			{"compute", "disks", "delete", fmt.Sprintf("neo4j-data-disk-%s", releaseName), "--zone=" + string(gcloud.CurrentZone()), "--project=" + string(gcloud.CurrentProject()), "--quiet"},
		}, false)
	}
}
