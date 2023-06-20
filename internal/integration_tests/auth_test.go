package integration_tests

import (
	"context"
	"fmt"
	"github.com/neo4j/helm-charts/internal/model"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func TestAuthSecretsWrongKey(t *testing.T) {
	t.Parallel()
	releaseName := model.NewReleaseName("auth-wrong-key-" + TestRunIdentifier)
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
	helmClient := model.NewHelmClient(model.Neo4jStandaloneChartName)
	helmValues := model.DefaultEnterpriseValues
	helmValues.Neo4J.Edition = model.Neo4jEdition
	helmValues.Neo4J.PasswordFromSecret = secretWrongKeyName
	_, err = helmClient.Install(t, releaseName.String(), namespace, helmValues)
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
	releaseName := model.NewReleaseName("auth-invalid-password-" + TestRunIdentifier)
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
	helmClient := model.NewHelmClient(model.Neo4jStandaloneChartName)
	helmValues := model.DefaultEnterpriseValues
	helmValues.Neo4J.Edition = model.Neo4jEdition
	helmValues.Neo4J.PasswordFromSecret = secretInvalidPasswordName
	_, err = helmClient.Install(t, releaseName.String(), namespace, helmValues)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Password in secret invalid-password must start with the characters 'neo4j/'")
	t.Cleanup(func() {
		_ = runAll(t, "kubectl", [][]string{
			{"delete", "namespace", string(releaseName.Namespace())},
		}, false)
	})
}

func TestAuthSecretsWithLookupDisabled(t *testing.T) {
	t.Parallel()
	releaseName := model.NewReleaseName("auth-invalid-password-" + TestRunIdentifier)
	_, err := createNamespace(t, releaseName)
	if err != nil {
		return
	}
	namespace := string(releaseName.Namespace())

	helmClient := model.NewHelmClient(model.Neo4jStandaloneChartName, "--dry-run")
	helmValues := model.DefaultEnterpriseValues
	helmValues.Neo4J.Edition = model.Neo4jEdition
	helmValues.Neo4J.PasswordFromSecret = "missing-secret"
	helmValues.DisableLookups = true

	_, err = helmClient.Install(t, releaseName.String(), namespace, helmValues)
	assert.NoError(t, err)
	t.Cleanup(func() {
		_ = runAll(t, "kubectl", [][]string{
			{"delete", "namespace", string(releaseName.Namespace())},
		}, false)
	})
}

func TestAuthPasswordCannotBeDifferent(t *testing.T) {
	if model.Neo4jEdition != "enterprise" {
		t.Skip()
		return
	}
	t.Parallel()
	releaseName1 := model.NewReleaseName("auth-pass-1-" + TestRunIdentifier)
	releaseName2 := model.NewReleaseName("auth-pass-2-" + TestRunIdentifier)
	_, err := createNamespace(t, releaseName1)
	if err != nil {
		return
	}
	namespace := string(releaseName1.Namespace())
	helmClient := model.NewHelmClient(model.Neo4jClusterCoreChartName)
	helmValues := model.DefaultEnterpriseValues
	helmValues.Neo4J.Edition = model.Neo4jEdition
	helmValues.Neo4J.Password = "password1"
	helmOutput, err := helmClient.Install(t, releaseName1.String(), namespace, helmValues)
	assert.NoError(t, err)
	assert.Contains(t, helmOutput, "WARNING: Passwords set using 'neo4j.password' will be stored in plain text in the Helm release ConfigMap.\nPlease consider using 'neo4j.passwordFromSecret' for improved security.")
	helmValues.Neo4J.Password = "password2"
	_, err = helmClient.Install(t, releaseName2.String(), namespace, helmValues)
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

// TestAuthLdapSecretsWrongKey tests helm installation failure when a valid secret with invalid key is used for ldapPasswordFromSecret
func TestAuthLdapSecretsWrongKey(t *testing.T) {
	t.Parallel()
	releaseName := model.NewReleaseName("ldap-auth-wrong-key-" + TestRunIdentifier)
	_, err := createNamespace(t, releaseName)
	if err != nil {
		return
	}
	namespace := string(releaseName.Namespace())
	secretWrongKeyName := "secret-wrong-key"
	secretWrongKeyData := make(map[string][]byte, 1)
	secretWrongKeyData["NEO4J_PASSWORD"] = []byte("demo1234")
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
	helmClient := model.NewHelmClient(model.Neo4jStandaloneChartName)
	helmValues := model.DefaultEnterpriseValues
	helmValues.LdapPasswordFromSecret = secretWrongKeyName
	helmValues.LdapPasswordMountPath = "/config/ldaPassword"
	_, err = helmClient.Install(t, releaseName.String(), namespace, helmValues)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Secret secret-wrong-key must contain key LDAP_PASS")
	t.Cleanup(func() {
		_ = runAll(t, "kubectl", [][]string{
			{"delete", "namespace", string(releaseName.Namespace())},
		}, false)
	})
}

// TestAuthLdapInvalidSecret tests helm installation failure when an invalid secret name is used for ldapPasswordFromSecret
func TestAuthLdapInvalidSecret(t *testing.T) {
	t.Parallel()
	releaseName := model.NewReleaseName("ldap-auth-wrong-key-" + TestRunIdentifier)
	_, err := createNamespace(t, releaseName)
	if err != nil {
		return
	}
	namespace := string(releaseName.Namespace())

	helmClient := model.NewHelmClient(model.Neo4jStandaloneChartName)
	helmValues := model.DefaultEnterpriseValues
	helmValues.LdapPasswordFromSecret = "invalidSecret"
	helmValues.LdapPasswordMountPath = "/config/ldapPassword"
	_, err = helmClient.Install(t, releaseName.String(), namespace, helmValues)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Secret invalidSecret configured in 'ldapPasswordFromSecret' not found")
	t.Cleanup(func() {
		_ = runAll(t, "kubectl", [][]string{
			{"delete", "namespace", string(releaseName.Namespace())},
		}, false)
	})
}

// TestBackupInvalidSecretKeyName checks backup helm chart installation with a secret that exists but with a different key name
func TestBackupInvalidSecretKeyName(t *testing.T) {
	t.Parallel()

	releaseName := model.NewReleaseName("ldap-auth-wrong-key-" + TestRunIdentifier)
	_, err := createNamespace(t, releaseName)
	if err != nil {
		return
	}
	namespace := string(releaseName.Namespace())

	secretName := "secret-wrong-key"
	helmValues := model.DefaultNeo4jBackupValues
	helmValues.Backup.SecretName = secretName
	helmValues.Backup.SecretKeyName = "demo1"
	helmValues.Backup.CloudProvider = "aws"
	helmValues.Backup.BucketName = "demo2"
	helmValues.Backup.DatabaseAdminServiceName = "standalone-admin"
	helmValues.Backup.Database = "neo4j1"

	secretWrongKey := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: namespace,
		},
		Data: map[string][]byte{
			"demo": []byte("demo"),
		},
		Type: "Opaque",
	}

	_, err = Clientset.CoreV1().Secrets(namespace).Create(context.TODO(), secretWrongKey, metav1.CreateOptions{})
	assert.NoError(t, err, "error seen while creating secret")

	helmClient := model.NewHelmClient(model.DefaultNeo4jBackupChartName)
	_, err = helmClient.Install(t, releaseName.String(), namespace, helmValues)
	assert.Contains(t, err.Error(), fmt.Sprintf("Secret %s must contain key %s", helmValues.Backup.SecretName, helmValues.Backup.SecretKeyName))
}
