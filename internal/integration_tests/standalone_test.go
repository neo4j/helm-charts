package integration_tests

import (
	"fmt"
	"github.com/neo4j/helm-charts/internal/integration_tests/gcloud"
	"github.com/neo4j/helm-charts/internal/model"
	"github.com/neo4j/helm-charts/internal/resources"
	"github.com/stretchr/testify/assert"
)
import "testing"

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
	runSubTests(t, []SubTest{
		{name: "Install Backup Helm Chart For GCP With Inconsistencies", test: func(t *testing.T) {
			assert.NoError(t, InstallNeo4jBackupGCPHelmChartWithInconsistencies(t, name), "Backup to GCP should succeed along with upload of inconsistencies report")
		}},
	})
}

func standaloneCleanup(t *testing.T, releaseName model.ReleaseName) func() {
	return func() {
		_ = runAll(t, "helm", [][]string{
			{"uninstall", releaseName.String(), "--wait", "--timeout", "3m", "--namespace", string(releaseName.Namespace())},
		}, false)
		_ = runAll(t, "kubectl", [][]string{
			{"delete", "pvc", fmt.Sprintf("%s-pvc", releaseName.String()), "--namespace", string(releaseName.Namespace()), "--ignore-not-found"},
			{"delete", "pv", fmt.Sprintf("%s-pv", releaseName.String()), "--ignore-not-found"},
		}, false)
		_ = runAll(t, "gcloud", [][]string{
			{"compute", "disks", "delete", fmt.Sprintf("neo4j-data-disk-%s", releaseName), "--zone=" + string(gcloud.CurrentZone()), "--project=" + string(gcloud.CurrentProject())},
		}, false)
		_ = runAll(t, "kubectl", [][]string{
			{"delete", "namespace", string(releaseName.Namespace()), "--ignore-not-found", "--force", "--grace-period=0"},
		}, false)
	}
}
