package integration_tests

import (
	"context"
	"fmt"
	"github.com/neo4j/helm-charts/internal/integration_tests/gcloud"
	"github.com/neo4j/helm-charts/internal/model"
	"github.com/neo4j/helm-charts/internal/resources"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func TestAuthSecretsWrongKey(t *testing.T) {
	t.Parallel()
	releaseName := model.NewReleaseName("install-" + TestRunIdentifier)
	_, err := createNamespace(t, releaseName)
	if err != nil {
		return
	}
	namespace := string(releaseName.Namespace())
	secretWrongKeyName := "secret-wrong-key"
	secretWrongKeyData := make(map[string][]byte)
	secretWrongKeyData["NEO4J_PASSWORD"] = []byte("neo4j/foo123")
	secretWrongKey := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretWrongKeyName,
			Namespace: namespace,
		},
		Data: secretWrongKeyData,
		Type: "Opaque",
	}
	_, err = Clientset.CoreV1().Secrets(namespace).Create(context.TODO(), secretWrongKey, metav1.CreateOptions{})
	if err != nil {
		return
	}
	helmValues := model.DefaultEnterpriseValues
	helmValues.Neo4J.Edition = model.Neo4jEdition
	helmValues.Neo4J.PasswordFromSecret = secretWrongKeyName
	_, err = model.HelmInstallFromStruct(t, model.HelmChart, releaseName.String(), string(releaseName.Namespace()), helmValues)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Secret secret-wrong-key must contain key NEO4J_DATA")
	t.Cleanup(func() {
		_ = runAll(t, "kubectl", [][]string{
			{"delete", "namespace", string(releaseName.Namespace())},
		}, false)
	})
}

func TestAuthSecretsInvalidPassword(t *testing.T) {
	t.Parallel()
	releaseName := model.NewReleaseName("install-" + TestRunIdentifier)
	_, err := createNamespace(t, releaseName)
	if err != nil {
		return
	}
	namespace := string(releaseName.Namespace())
	secretInvalidPasswordName := "invalid-password"
	secretInvalidPasswordData := make(map[string][]byte)
	secretInvalidPasswordData["NEO4J_AUTH"] = []byte("user/foo123")
	secretWrongKey := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretInvalidPasswordName,
			Namespace: namespace,
		},
		Data: secretInvalidPasswordData,
		Type: "Opaque",
	}
	_, err = Clientset.CoreV1().Secrets(namespace).Create(context.TODO(), secretWrongKey, metav1.CreateOptions{})
	if err != nil {
		return
	}
	helmValues := model.DefaultEnterpriseValues
	helmValues.Neo4J.Edition = model.Neo4jEdition
	helmValues.Neo4J.PasswordFromSecret = secretInvalidPasswordName
	_, err = model.HelmInstallFromStruct(t, model.HelmChart, releaseName.String(), string(releaseName.Namespace()), helmValues)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Password in secret invalid-password must start with the characters 'neo4j/'")
	t.Cleanup(func() {
		_ = runAll(t, "kubectl", [][]string{
			{"delete", "namespace", string(releaseName.Namespace())},
		}, false)
	})
}

func TestAuthPasswordCannotBeDifferent(t *testing.T) {
	t.Parallel()
	releaseName1 := model.NewReleaseName("install1-" + TestRunIdentifier)
	releaseName2 := model.NewReleaseName("install2-" + TestRunIdentifier)
	releaseName2.Namespace()
	_, err := createNamespace(t, releaseName1)
	if err != nil {
		return
	}

	helmValues := model.DefaultEnterpriseValues
	helmValues.Neo4J.Edition = model.Neo4jEdition
	helmValues.Neo4J.Password = "password1"
	helmValues.Neo4J.MinimumClusterSize = 3
	_, err = model.HelmInstallFromStruct(t, model.HelmChart, releaseName1.String(), string(releaseName1.Namespace()), helmValues)
	assert.NoError(t, err)
	helmValues.Neo4J.Password = "password2"
	_, err = model.HelmInstallFromStruct(t, model.HelmChart, releaseName2.String(), string(releaseName1.Namespace()), helmValues)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "The desired password does not match the password stored in the Kubernetes Secret")
	t.Cleanup(func() {
		_ = runAll(t, "helm", [][]string{
			{"uninstall", releaseName1.String(), "--namespace", string(releaseName1.Namespace())},
		}, false)
		_ = runAll(t, "kubectl", [][]string{
			{"delete", "namespace", string(releaseName1.Namespace())},
		}, false)
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
