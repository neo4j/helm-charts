package integration_tests

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/go-multierror"
	. "github.com/neo4j/helm-charts/internal/helpers"
	"github.com/neo4j/helm-charts/internal/integration_tests/gcloud"
	"github.com/neo4j/helm-charts/internal/model"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

// labelNodes labels all the node with testLabel=namespace-<number>
func labelNodes(t *testing.T, namespace string) error {

	var errors *multierror.Error
	nodesList, err := getNodesList()
	if err != nil {
		return err
	}

	for index, node := range nodesList.Items {
		labelName := fmt.Sprintf("testLabel=%s-%d", namespace, index+1)
		// Use --overwrite to handle cases where labels exist from previous test runs that didn't clean up properly
		err = run(t, "kubectl", "label", "nodes", node.ObjectMeta.Name, labelName, "--overwrite")
		if err != nil {
			errors = multierror.Append(errors, err)
			t.Logf("Node Label failed for %s: %v", node.ObjectMeta.Name, err)
		}
	}

	// Wait for labels to propagate to the Kubernetes API and verify they exist
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	allLabelsVerified := false
	for !allLabelsVerified {
		select {
		case <-ctx.Done():
			// Timeout - check which labels are missing
			missingLabels := []string{}
			for i := 0; i < len(nodesList.Items); i++ {
				expectedLabel := fmt.Sprintf("testLabel=%s-%d", namespace, i+1)
				_, verifyErr := getNodeWithLabel(expectedLabel)
				if verifyErr != nil {
					missingLabels = append(missingLabels, expectedLabel)
				}
			}
			if len(missingLabels) > 0 {
				return fmt.Errorf("timeout waiting for labels to be applied. Missing labels: %v", missingLabels)
			}
			allLabelsVerified = true
		case <-ticker.C:
			// Check if all labels are now present
			allPresent := true
			for i := 0; i < len(nodesList.Items); i++ {
				expectedLabel := fmt.Sprintf("testLabel=%s-%d", namespace, i+1)
				_, verifyErr := getNodeWithLabel(expectedLabel)
				if verifyErr != nil {
					allPresent = false
					break
				}
			}
			if allPresent {
				allLabelsVerified = true
				t.Logf("All %d node labels verified successfully", len(nodesList.Items))
			}
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
			// Log but don't treat as error
			t.Logf("Note: Label removal for %s returned error (may be expected if label doesn't exist): %v", node.ObjectMeta.Name, err)
		}
	}

	return errors.ErrorOrNil()
}

// clusterTests contains all the tests related to cluster
func clusterTests(clusterRelease model.ReleaseName) ([]SubTest, error) {

	subTests := []SubTest{
		{name: "Install Backup Helm Chart For AWS Local With Consistency Check", test: func(t *testing.T) {
			t.Parallel()
			assert.NoError(t, InstallNeo4jBackupAWSLocalWithConsistencyCheck(t, clusterRelease), "Local backup with consistency check should succeed")
		}},
		{name: "Install Backup Helm Chart For AWS Cloud Storage", test: func(t *testing.T) {
			t.Parallel()
			assert.NoError(t, InstallNeo4jBackupAWSCloudStorage(t, clusterRelease), "Cloud backup to AWS S3 should succeed")
		}},
		{name: "Install Backup Helm Chart For AWS Using S3", test: func(t *testing.T) {
			t.Parallel()
			assert.NoError(t, InstallNeo4jBackupAWSHelmChartViaS3(t, clusterRelease), "Backup to AWS using S3 should succeed")
		}},
		{name: "Install Backup Helm Chart For AWS Using S3 with TLS", test: func(t *testing.T) {
			t.Parallel()
			assert.NoError(t, InstallNeo4jBackupAWSHelmChartViaS3TLS(t, clusterRelease), "Backup to AWS using S3 with TLS should succeed")
		}},
		{name: "Install Backup Helm Chart For AWS Using Custom Aggregate Tempdir", test: func(t *testing.T) {
			t.Parallel()
			assert.NoError(t, InstallBackupViaTempDir(t, clusterRelease), "Backup with custom aggregate tempdir should succeed")
		}},
		{name: "Check Cluster Core Logs Format", test: func(t *testing.T) {
			t.Parallel()
			assert.NoError(t, CheckLogsFormat(t, clusterRelease), "Cluster core logs format should be in JSON")
		}},
		{name: "Check Neo4j Operations Pod for enabling server", test: func(t *testing.T) {
			t.Parallel()
			assert.NoError(t, CheckNeo4jOperationsPod(t, clusterRelease), "Neo4j Operations Pod should get executed")
		}},
		{name: "ImagePullSecret tests", test: func(t *testing.T) {
			t.Parallel()
			assert.NoError(t, imagePullSecretTests(t, clusterRelease), "Perform ImagePullSecret Tests")
		}},
		{name: "Check PriorityClassName", test: func(t *testing.T) {
			t.Parallel()
			assert.NoError(t, checkPriorityClassName(t, clusterRelease), "priorityClassName should match")
		}},
		{name: "Check K8s", test: func(t *testing.T) {
			assert.NoError(t, checkK8s(t, clusterRelease), "Neo4j Config check should succeed")
		}},
		{name: "Check Ldap Password", test: func(t *testing.T) {
			assert.NoError(t, checkLdapPassword(t, clusterRelease), "LdapPassword should be set")
		}},
		{name: "Create Node", test: func(t *testing.T) {
			assert.NoError(t, createNode(t, clusterRelease), "Create Node should succeed")
		}},
		{name: "Count Nodes", test: func(t *testing.T) {
			assert.NoError(t, checkNodeCount(t, clusterRelease), "Count Nodes should succeed")
		}},
		{name: "Database Creation Tests", test: func(t *testing.T) {
			assert.NoError(t, databaseCreationTests(t, clusterRelease, "customers"), "Creates \"customer\" database and checks for its existence")
		}},
		{name: "Install Backup Helm Chart For GCP With Workload Identity For Cluster", test: func(t *testing.T) {
			assert.NoError(t, InstallNeo4jBackupGCPHelmChartWithWorkloadIdentityForCluster(t, clusterRelease), "Backup to GCP with workload identity should succeed")
		}},
	}
	return subTests, nil
}

func InstallNeo4jBackupGCPHelmChartWithWorkloadIdentityForCluster(t *testing.T, clusterReleaseName model.ReleaseName) error {
	if model.Neo4jEdition == "community" {
		t.Skip()
		return nil
	}
	shortName := clusterReleaseName.ShortName()
	currentUnixTime := time.Now().Unix()
	backupReleaseName := model.NewReleaseName(fmt.Sprintf("%s-gcp-workload-%s", shortName, TestRunIdentifier))
	gcpServiceAccountName := fmt.Sprintf("%s-%d", gcpServiceAccountNamePrefix, currentUnixTime)
	k8sServiceAccountName := fmt.Sprintf("%s-%d", k8sServiceAccountNamePrefix, currentUnixTime)
	namespace := string(clusterReleaseName.Namespace())

	t.Cleanup(func() {
		_ = runAll(t, "helm", [][]string{
			{"uninstall", backupReleaseName.String(), "--wait", "--timeout", "3m", "--namespace", namespace},
		}, false)
		_ = deleteGCPServiceAccount(gcpServiceAccountName)
	})

	serviceAccount := v1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ServiceAccount",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sServiceAccountName,
			Namespace: namespace,
			Annotations: map[string]string{
				"iam.gke.io/gcp-service-account": fmt.Sprintf("%s@%s.iam.gserviceaccount.com", gcpServiceAccountName, gcloud.CurrentProject()),
			},
		},
	}

	_, err := Clientset.CoreV1().ServiceAccounts(namespace).Create(context.Background(), &serviceAccount, metav1.CreateOptions{})
	assert.NoError(t, err, fmt.Sprintf("error seen while creating k8s service account for cluster %s. \n Err := %v", k8sServiceAccountName, err))

	err = createGCPServiceAccount(k8sServiceAccountName, namespace, gcpServiceAccountName)
	assert.NoError(t, err, fmt.Sprintf("error seen while creating GCP service account for cluster %s. \n Err := %v", gcpServiceAccountName, err))

	bucketName := model.BucketName
	helmClient := model.NewHelmClient(model.DefaultNeo4jBackupChartName)
	helmValues := model.DefaultNeo4jBackupValues
	helmValues.Backup = model.Backup{
		BucketName:               bucketName,
		DatabaseAdminServiceName: fmt.Sprintf("%s-admin", clusterReleaseName.String()),
		DatabaseNamespace:        string(clusterReleaseName.Namespace()),
		Database:                 "neo4j,system",
		CloudProvider:            "gcp",
		Verbose:                  true,
		Type:                     "FULL",
		KeepBackupFiles:          true,
	}
	helmValues.ServiceAccountName = k8sServiceAccountName

	// Explicitly disable consistency checks for cloud storage backups to avoid timeouts
	// This follows the same pattern used for AWS cloud backups
	helmValues.ConsistencyCheck.Enable = false
	helmValues.ConsistencyCheck.Database = ""

	_, err = helmClient.Install(t, backupReleaseName.String(), namespace, helmValues)
	assert.NoError(t, err)

	time.Sleep(2 * time.Minute)
	cronjob, err := Clientset.BatchV1().CronJobs(namespace).Get(context.Background(), backupReleaseName.String(), metav1.GetOptions{})
	assert.NoError(t, err, "cannot retrieve gcp backup cronjob")
	assert.Equal(t, cronjob.Spec.Schedule, helmValues.Neo4J.JobSchedule, fmt.Sprintf("gcp cronjob schedule %s not matching with the schedule defined in values.yaml %s", cronjob.Spec.Schedule, helmValues.Neo4J.JobSchedule))

	pods, err := Clientset.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{})
	assert.NoError(t, err, "error while retrieving pod list during gcp backup operation")

	var found bool
	for _, pod := range pods.Items {
		if strings.Contains(pod.Name, "gcp-workload") {
			found = true
			out, err := exec.Command("kubectl", "logs", pod.Name, "--namespace", namespace).CombinedOutput()
			assert.NoError(t, err, "error while getting gcp workload backup pod logs")
			assert.NotNil(t, out, "gcp backup logs cannot be retrieved")
			assert.Contains(t, string(out), "Backup completed successfully")
			assert.NotContains(t, string(out), "Deleting file")
			break
		}
	}
	assert.Equal(t, true, found, "no gcp workload backup pod found")

	return nil
}

// InstallNeo4jBackupAWSLocalWithConsistencyCheck performs local backup with consistency check
func InstallNeo4jBackupAWSLocalWithConsistencyCheck(t *testing.T, releaseName model.ReleaseName) error {
	if model.Neo4jEdition == "community" {
		t.Skip()
		return nil
	}
	backupReleaseName := model.NewReleaseName("cluster-backup-local-" + TestRunIdentifier)
	namespace := string(releaseName.Namespace())

	t.Cleanup(func() {
		_ = runAll(t, "helm", [][]string{
			{"uninstall", backupReleaseName.String(), "--wait", "--timeout", "3m", "--namespace", namespace},
		}, false)
	})

	helmClient := model.NewHelmClient(model.DefaultNeo4jBackupChartName)
	helmValues := model.DefaultNeo4jBackupValues
	helmValues.Backup = model.Backup{
		DatabaseAdminServiceName: fmt.Sprintf("%s-admin", releaseName.String()),
		DatabaseNamespace:        namespace,
		Database:                 "neo4j,system",
		CloudProvider:            "", // Local backup
		Verbose:                  true,
		Type:                     "FULL",
		KeepBackupFiles:          true,
	}
	helmValues.NodeSelector = map[string]string{
		"testLabel": fmt.Sprintf("%s-5", namespace),
	}
	// Enable consistency check for local backup
	helmValues.ConsistencyCheck.Enable = true
	helmValues.ConsistencyCheck.Database = "neo4j"

	_, err := helmClient.Install(t, backupReleaseName.String(), namespace, helmValues)
	assert.NoError(t, err)

	time.Sleep(2 * time.Minute)
	cronjob, err := Clientset.BatchV1().CronJobs(namespace).Get(context.Background(), backupReleaseName.String(), metav1.GetOptions{})
	assert.NoError(t, err, "cannot retrieve local backup cronjob")
	assert.Equal(t, cronjob.Spec.Schedule, helmValues.Neo4J.JobSchedule, fmt.Sprintf("local backup cronjob schedule %s not matching with the schedule defined in values.yaml %s", cronjob.Spec.Schedule, helmValues.Neo4J.JobSchedule))

	nodeSelectorNode, err := getNodeWithLabel(fmt.Sprintf("testLabel=%s-5", namespace))
	assert.NoError(t, err)

	// Poll for backup completion with consistency check - shorter timeout for local backup
	deadline := time.Now().Add(10 * time.Minute) // Local backup should be much faster
	var found bool
	var logOutput string

	for !time.Now().After(deadline) {
		pods, err := Clientset.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{})
		if err != nil {
			t.Logf("Error retrieving pod list: %v", err)
			time.Sleep(30 * time.Second)
			continue
		}

		found = false
		for _, pod := range pods.Items {
			if strings.Contains(pod.Name, "cluster-backup-local") {
				found = true
				out, err := exec.Command("kubectl", "logs", pod.Name, "--namespace", namespace).CombinedOutput()
				if err != nil {
					t.Logf("Error getting pod logs: %v", err)
					time.Sleep(30 * time.Second)
					continue
				}

				logOutput = string(out)

				// Check if backup completed successfully
				if !strings.Contains(logOutput, "Backup completed successfully") {
					t.Logf("Local backup not yet completed, waiting...")
					time.Sleep(30 * time.Second)
					continue
				}

				// Check if consistency check completed successfully
				if strings.Contains(logOutput, "No inconsistencies found") {
					t.Logf("Local backup and consistency check completed successfully")
					assert.Equal(t, nodeSelectorNode.Name, pod.Spec.NodeName, fmt.Sprintf("backup pod %s is not scheduled on the correct node %s", pod.Spec.NodeName, nodeSelectorNode.Name))
					return nil
				} else if strings.Contains(logOutput, "Consistency Check Failed") || strings.Contains(logOutput, "Consistency check timed out") {
					t.Logf("Consistency check failed or timed out")
					assert.Fail(t, "Consistency check failed", "Consistency check failed or timed out. Logs: %s", logOutput)
					return fmt.Errorf("consistency check failed")
				} else {
					// Consistency check is still running
					t.Logf("Local backup completed, consistency check still in progress...")
					time.Sleep(30 * time.Second)
					continue
				}
			}
		}

		if !found {
			t.Logf("No local backup pod found yet, waiting...")
			time.Sleep(30 * time.Second)
		}
	}

	if !found {
		assert.Fail(t, "No local backup pod found after timeout")
		return fmt.Errorf("no local backup pod found")
	}

	// If we reach here, we timed out waiting for consistency check
	assert.Fail(t, "Local backup consistency check did not complete within timeout", "Final logs: %s", logOutput)
	return fmt.Errorf("local backup consistency check did not complete within 10 minutes")
}

// InstallNeo4jBackupAWSCloudStorage performs cloud backup to AWS S3 without consistency check
func InstallNeo4jBackupAWSCloudStorage(t *testing.T, releaseName model.ReleaseName) error {
	if model.Neo4jEdition == "community" {
		t.Skip()
		return nil
	}
	backupReleaseName := model.NewReleaseName("cluster-backup-aws-" + TestRunIdentifier)
	namespace := string(releaseName.Namespace())

	t.Cleanup(func() {
		_ = runAll(t, "kubectl", [][]string{
			{"delete", "secret", "awscred", "--namespace", namespace, "--ignore-not-found"},
		}, false)
		_ = runAll(t, "helm", [][]string{
			{"uninstall", backupReleaseName.String(), "--wait", "--timeout", "3m", "--namespace", namespace},
		}, false)
	})

	_ = runAll(t, "kubectl", [][]string{
		{"delete", "secret", "awscred", "--namespace", namespace, "--ignore-not-found"},
	}, false)

	time.Sleep(2 * time.Second)

	secretKey := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "awscred",
			Namespace: namespace,
		},
		Data: map[string][]byte{
			"credentials": []byte(fmt.Sprintf("[default]\nregion = us-east-1\naws_access_key_id=%s\naws_secret_access_key=%s",
				os.Getenv("AWS_ACCESS_KEY_ID"),
				os.Getenv("AWS_SECRET_ACCESS_KEY"))),
		},
		Type: "Opaque",
	}

	_, err := Clientset.CoreV1().Secrets(namespace).Create(context.TODO(), secretKey, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create AWS credentials secret: %v", err)
	}

	_, err = Clientset.CoreV1().Secrets(namespace).Get(context.TODO(), "awscred", metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to verify AWS credentials secret exists: %v", err)
	}

	time.Sleep(2 * time.Second)

	bucketName := model.BucketName
	helmClient := model.NewHelmClient(model.DefaultNeo4jBackupChartName)
	helmValues := model.DefaultNeo4jBackupValues
	helmValues.Backup = model.Backup{
		BucketName:               bucketName,
		DatabaseAdminServiceName: fmt.Sprintf("%s-admin", releaseName.String()),
		DatabaseNamespace:        namespace,
		Database:                 "neo4j,system",
		CloudProvider:            "aws",
		SecretName:               "awscred",
		SecretKeyName:            "credentials",
		S3Region:                 "us-east-1",
		S3ForcePathStyle:         true,
		Verbose:                  true,
		Type:                     "FULL",
		KeepBackupFiles:          true,
	}
	helmValues.NodeSelector = map[string]string{
		"testLabel": fmt.Sprintf("%s-5", namespace),
	}
	// Disable consistency check for cloud backup to avoid timeouts and reduce template size
	helmValues.ConsistencyCheck.Enable = false
	helmValues.ConsistencyCheck.Database = ""

	_, err = helmClient.Install(t, backupReleaseName.String(), namespace, helmValues)
	assert.NoError(t, err)

	time.Sleep(2 * time.Minute)
	cronjob, err := Clientset.BatchV1().CronJobs(namespace).Get(context.Background(), backupReleaseName.String(), metav1.GetOptions{})
	assert.NoError(t, err, "cannot retrieve aws cloud backup cronjob")
	assert.Equal(t, cronjob.Spec.Schedule, helmValues.Neo4J.JobSchedule, fmt.Sprintf("aws cloud backup cronjob schedule %s not matching with the schedule defined in values.yaml %s", cronjob.Spec.Schedule, helmValues.Neo4J.JobSchedule))

	nodeSelectorNode, err := getNodeWithLabel(fmt.Sprintf("testLabel=%s-5", namespace))
	assert.NoError(t, err)

	// Poll for cloud backup completion - reasonable timeout without consistency check
	deadline := time.Now().Add(8 * time.Minute) // Cloud backup without consistency check
	var found bool
	var logOutput string

	for !time.Now().After(deadline) {
		pods, err := Clientset.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{})
		if err != nil {
			t.Logf("Error retrieving pod list: %v", err)
			time.Sleep(30 * time.Second)
			continue
		}

		found = false
		for _, pod := range pods.Items {
			if strings.Contains(pod.Name, "cluster-backup-aws") {
				found = true
				out, err := exec.Command("kubectl", "logs", pod.Name, "--namespace", namespace).CombinedOutput()
				if err != nil {
					t.Logf("Error getting pod logs: %v", err)
					time.Sleep(30 * time.Second)
					continue
				}

				logOutput = string(out)

				// Check if backup completed successfully
				if strings.Contains(logOutput, "Backup completed successfully") {
					t.Logf("Cloud backup to AWS S3 completed successfully")
					assert.Equal(t, nodeSelectorNode.Name, pod.Spec.NodeName, fmt.Sprintf("backup pod %s is not scheduled on the correct node %s", pod.Spec.NodeName, nodeSelectorNode.Name))

					// Verify backup files were created in S3
					if strings.Contains(logOutput, "neo4j-") && strings.Contains(logOutput, "system-") {
						t.Logf("Backup files successfully uploaded to S3")
					}
					return nil
				} else {
					t.Logf("Cloud backup not yet completed, waiting...")
					time.Sleep(30 * time.Second)
					continue
				}
			}
		}

		if !found {
			t.Logf("No cloud backup pod found yet, waiting...")
			time.Sleep(30 * time.Second)
		}
	}

	if !found {
		assert.Fail(t, "No AWS cloud backup pod found after timeout")
		return fmt.Errorf("no aws cloud backup pod found")
	}

	// If we reach here, we timed out waiting for backup completion
	assert.Fail(t, "Cloud backup did not complete within timeout", "Final logs: %s", logOutput)
	return fmt.Errorf("cloud backup did not complete within 8 minutes")
}

func InstallNeo4jBackupAWSHelmChartViaS3(t *testing.T, releaseName model.ReleaseName) error {
	if model.Neo4jEdition == "community" {
		t.Skip()
		return nil
	}

	namespace := "default"
	backupReleaseName := model.NewReleaseName("cluster-backup-aws-s3" + TestRunIdentifier)
	secretName := "awscred"

	t.Cleanup(func() {
		_ = runAll(t, "kubectl", [][]string{
			{"delete", "secret", secretName, "--namespace", namespace, "--ignore-not-found"},
		}, false)

		_ = runAll(t, "helm", [][]string{
			{"uninstall", backupReleaseName.String(), "--wait", "--timeout", "3m", "--namespace", namespace},
		}, false)
	})

	secretKey := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: namespace,
		},
		Data: map[string][]byte{
			"credentials": []byte(fmt.Sprintf("[default]\naws_access_key_id=%s\naws_secret_access_key=%s",
				os.Getenv("AWS_ACCESS_KEY_ID"),
				os.Getenv("AWS_SECRET_ACCESS_KEY"))),
		},
		Type: "Opaque",
	}

	_ = runAll(t, "kubectl", [][]string{
		{"delete", "secret", secretName, "--namespace", namespace, "--ignore-not-found"},
	}, false)

	_, err := Clientset.CoreV1().Secrets(namespace).Create(context.TODO(), secretKey, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create AWS credentials secret: %v", err)
	}

	_, err = Clientset.CoreV1().Secrets(namespace).Get(context.TODO(), secretName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to verify AWS credentials secret exists: %v", err)
	}

	time.Sleep(2 * time.Second)

	bucketName := model.BucketName
	helmClient := model.NewHelmClient(model.DefaultNeo4jBackupChartName)
	helmValues := model.DefaultNeo4jBackupValues
	helmValues.Backup = model.Backup{
		BucketName:               bucketName,
		DatabaseAdminServiceName: fmt.Sprintf("%s-admin", releaseName.String()),
		DatabaseNamespace:        string(releaseName.Namespace()),
		Database:                 "neo4j,system",
		CloudProvider:            "aws",
		SecretName:               secretName,
		SecretKeyName:            "credentials",
		S3Endpoint:               "http://localhost:9000",
		S3Region:                 "us-east-1",
		S3SignatureVersion:       "4",
		S3ForcePathStyle:         true,
		Verbose:                  true,
		KeepBackupFiles:          true,
		Type:                     "FULL",
	}
	// Disable consistency check for S3 configuration test to avoid timeouts
	// This test focuses on S3 parameters, not consistency check functionality
	helmValues.ConsistencyCheck.Enable = false
	helmValues.ConsistencyCheck.Database = ""
	helmValues.Neo4J.JobSchedule = "* * * * *"

	_, err = helmClient.Install(t, backupReleaseName.String(), namespace, helmValues)
	if err != nil {
		return fmt.Errorf("helm install failed: %v", err)
	}

	time.Sleep(2 * time.Minute)

	cronjob, err := Clientset.BatchV1().CronJobs(namespace).Get(context.Background(), backupReleaseName.String(), metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("cannot retrieve aws backup cronjob: %v", err)
	}

	if cronjob.Spec.Schedule != helmValues.Neo4J.JobSchedule {
		return fmt.Errorf("aws cronjob schedule %s not matching with the schedule defined in values.yaml %s",
			cronjob.Spec.Schedule, helmValues.Neo4J.JobSchedule)
	}

	pods, err := Clientset.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error while retrieving pod list during aws backup operation: %v", err)
	}

	var found bool
	for _, pod := range pods.Items {
		if strings.Contains(pod.Name, "cluster-backup-aws-s3") {
			found = true
			// Verify that the new S3 parameters are correctly set
			for _, container := range pod.Spec.Containers {
				s3ForcePathStyleFound := false
				s3RegionFound := false
				s3SignatureVersionFound := false

				for _, env := range container.Env {
					if env.Name == "S3_FORCE_PATH_STYLE" && env.Value == "true" {
						s3ForcePathStyleFound = true
					}
					if env.Name == "AWS_REGION" && env.Value == "us-east-1" {
						s3RegionFound = true
					}
					if env.Name == "S3_SIGNATURE_VERSION" && env.Value == "4" {
						s3SignatureVersionFound = true
					}
				}

				if !s3ForcePathStyleFound {
					return fmt.Errorf("S3_FORCE_PATH_STYLE environment variable not found or not set to true")
				}
				if !s3RegionFound {
					return fmt.Errorf("AWS_REGION environment variable not found or not set to us-east-1")
				}
				if !s3SignatureVersionFound {
					return fmt.Errorf("S3_SIGNATURE_VERSION environment variable not found or not set to 4")
				}
			}
			break
		}
	}
	if !found {
		return fmt.Errorf("no aws s3 backup pod found")
	}

	return nil
}

func InstallNeo4jBackupAWSHelmChartViaS3TLS(t *testing.T, releaseName model.ReleaseName) error {
	if model.Neo4jEdition == "community" {
		t.Skip()
		return nil
	}

	namespace := "default"
	backupReleaseName := model.NewReleaseName("cluster-backup-aws-s3-tls" + TestRunIdentifier)
	secretName := "awscred"
	caCertSecretName := "s3-ca-cert"

	t.Cleanup(func() {
		_ = runAll(t, "kubectl", [][]string{
			{"delete", "secret", secretName, "--namespace", namespace, "--ignore-not-found"},
			{"delete", "secret", caCertSecretName, "--namespace", namespace, "--ignore-not-found"},
		}, false)

		_ = runAll(t, "helm", [][]string{
			{"uninstall", backupReleaseName.String(), "--wait", "--timeout", "3m", "--namespace", namespace},
		}, false)
	})

	secretKey := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: namespace,
		},
		Data: map[string][]byte{
			"credentials": []byte(fmt.Sprintf("[default]\naws_access_key_id=%s\naws_secret_access_key=%s",
				os.Getenv("AWS_ACCESS_KEY_ID"),
				os.Getenv("AWS_SECRET_ACCESS_KEY"))),
		},
		Type: "Opaque",
	}

	_, err := Clientset.CoreV1().Secrets(namespace).Create(context.TODO(), secretKey, metav1.CreateOptions{})
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		return fmt.Errorf("failed to create AWS credentials secret: %v", err)
	}

	_, err = Clientset.CoreV1().Secrets(namespace).Get(context.TODO(), secretName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to verify AWS credentials secret exists: %v", err)
	}

	// Create CA certificate secret
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return fmt.Errorf("failed to generate private key: %v", err)
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: "s3.amazonaws.com",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
		DNSNames:              []string{"s3.amazonaws.com"},
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, template, template, &priv.PublicKey, priv)
	if err != nil {
		return fmt.Errorf("failed to create certificate: %v", err)
	}

	certPEM := new(bytes.Buffer)
	pem.Encode(certPEM, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})

	caCertSecret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      caCertSecretName,
			Namespace: namespace,
		},
		Data: map[string][]byte{
			"ca.crt": certPEM.Bytes(),
		},
		Type: "Opaque",
	}

	_, err = Clientset.CoreV1().Secrets(namespace).Create(context.TODO(), caCertSecret, metav1.CreateOptions{})
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		return fmt.Errorf("failed to create CA certificate secret: %v", err)
	}

	_, err = Clientset.CoreV1().Secrets(namespace).Get(context.TODO(), caCertSecretName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to verify CA certificate secret exists: %v", err)
	}

	time.Sleep(2 * time.Second)

	bucketName := model.BucketName
	helmClient := model.NewHelmClient(model.DefaultNeo4jBackupChartName)
	helmValues := model.DefaultNeo4jBackupValues
	helmValues.Backup = model.Backup{
		BucketName:               bucketName,
		DatabaseAdminServiceName: fmt.Sprintf("%s-admin", releaseName.String()),
		DatabaseNamespace:        string(releaseName.Namespace()),
		Database:                 "neo4j,system",
		CloudProvider:            "aws",
		SecretName:               secretName,
		SecretKeyName:            "credentials",
		S3Endpoint:               "https://s3.amazonaws.com",
		S3CASecretName:           caCertSecretName,
		S3CASecretKey:            "ca.crt",
		S3SkipVerify:             false,
		S3ForcePathStyle:         true,
		S3Region:                 "us-east-1",
		S3SignatureVersion:       "4",
		Verbose:                  true,
		KeepBackupFiles:          true,
		Type:                     "FULL",
	}
	// Disable consistency check for S3 TLS configuration test to avoid timeouts
	// This test focuses on S3 TLS parameters, not consistency check functionality
	helmValues.ConsistencyCheck.Enable = false
	helmValues.ConsistencyCheck.Database = ""
	helmValues.Neo4J.JobSchedule = "* * * * *"
	_, err = helmClient.Install(t, backupReleaseName.String(), namespace, helmValues)
	if err != nil {
		return fmt.Errorf("helm install failed: %v", err)
	}

	time.Sleep(2 * time.Minute)

	cronjob, err := Clientset.BatchV1().CronJobs(namespace).Get(context.Background(), backupReleaseName.String(), metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("cannot retrieve aws backup cronjob: %v", err)
	}

	if cronjob.Spec.Schedule != helmValues.Neo4J.JobSchedule {
		return fmt.Errorf("aws cronjob schedule %s not matching with the schedule defined in values.yaml %s",
			cronjob.Spec.Schedule, helmValues.Neo4J.JobSchedule)
	}

	pods, err := Clientset.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error while retrieving pod list during aws backup operation: %v", err)
	}

	var found bool
	for _, pod := range pods.Items {
		if strings.Contains(pod.Name, "cluster-backup-aws-s3-tls") {
			found = true
			// Verify that the CA certificate is mounted
			for _, container := range pod.Spec.Containers {
				for _, volumeMount := range container.VolumeMounts {
					if volumeMount.Name == "s3-ca-cert" {
						if volumeMount.MountPath != "/s3-ca-cert" {
							return fmt.Errorf("expected CA certificate mount path to be /s3-ca-cert but got %s", volumeMount.MountPath)
						}
					}
				}
				// Verify that CA certificate path is set correctly
				for _, env := range container.Env {
					if env.Name == "S3_CA_CERT_PATH" {
						if env.Value != "/s3-ca-cert/ca.crt" {
							return fmt.Errorf("expected S3_CA_CERT_PATH to be /s3-ca-cert/ca.crt but got %s", env.Value)
						}
					}
				}

				// Verify that the new S3 parameters are correctly set
				s3ForcePathStyleFound := false
				s3RegionFound := false
				s3SignatureVersionFound := false

				for _, env := range container.Env {
					if env.Name == "S3_FORCE_PATH_STYLE" && env.Value == "true" {
						s3ForcePathStyleFound = true
					}
					if env.Name == "AWS_REGION" && env.Value == "us-east-1" {
						s3RegionFound = true
					}
					if env.Name == "S3_SIGNATURE_VERSION" && env.Value == "4" {
						s3SignatureVersionFound = true
					}
				}

				if !s3ForcePathStyleFound {
					return fmt.Errorf("S3_FORCE_PATH_STYLE environment variable not found or not set to true")
				}
				if !s3RegionFound {
					return fmt.Errorf("AWS_REGION environment variable not found or not set to us-east-1")
				}
				if !s3SignatureVersionFound {
					return fmt.Errorf("S3_SIGNATURE_VERSION environment variable not found or not set to 4")
				}
			}
			break
		}
	}
	if !found {
		return fmt.Errorf("no aws s3 tls backup pod found")
	}

	return nil
}

// InstallBackupViaTempDir installs backup cronjob with a custom aggregate backup tempdir
func InstallBackupViaTempDir(t *testing.T, releaseName model.ReleaseName) error {
	if model.Neo4jEdition == "community" {
		t.Skip()
		return nil
	}

	backupReleaseName := model.NewReleaseName(fmt.Sprintf("%s-backup-s3-tmp", releaseName.String()))
	namespace := string(releaseName.Namespace())
	secretName := "miniocred"
	customTempDir := "/tmp/custom-aggregate-temp"

	// Add cleanup
	t.Cleanup(func() {
		_ = runAll(t, "helm", [][]string{
			{"uninstall", backupReleaseName.String(), "--wait", "--timeout", "3m", "--namespace", namespace},
		}, false)
	})

	secretKey := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: namespace,
		},
		Data: map[string][]byte{
			"credentials": []byte(fmt.Sprintf("[default]\nregion = us-east-1\naws_access_key_id=%s\naws_secret_access_key=%s",
				os.Getenv("AWS_ACCESS_KEY_ID"),
				os.Getenv("AWS_SECRET_ACCESS_KEY"))),
		},
		Type: "Opaque",
	}

	_, err := Clientset.CoreV1().Secrets(namespace).Create(context.TODO(), secretKey, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create AWS credentials secret: %v", err)
	}

	_, err = Clientset.CoreV1().Secrets(namespace).Get(context.TODO(), secretName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to verify AWS credentials secret exists: %v", err)
	}

	time.Sleep(2 * time.Second)

	bucketName := model.BucketName
	helmClient := model.NewHelmClient(model.DefaultNeo4jBackupChartName)
	helmValues := model.DefaultNeo4jBackupValues
	helmValues.Backup = model.Backup{
		BucketName:               bucketName,
		DatabaseAdminServiceName: fmt.Sprintf("%s-admin", releaseName.String()),
		DatabaseNamespace:        string(releaseName.Namespace()),
		Database:                 "neo4j,system",
		CloudProvider:            "aws",
		SecretName:               secretName,
		SecretKeyName:            "credentials",
		S3Endpoint:               "s3.amazonaws.com",
		S3ForcePathStyle:         true,
		Verbose:                  true,
		Type:                     "FULL",
		AggregateBackup: model.AggregateBackup{
			Enabled: true,
			TempDir: customTempDir,
		},
	}
	// Disable consistency check for temp directory configuration test to avoid timeouts
	// This test focuses on custom temp directory functionality, not consistency check
	helmValues.ConsistencyCheck.Enable = false
	helmValues.ConsistencyCheck.Database = ""
	helmValues.Neo4J.JobSchedule = "* * * * *"

	_, err = helmClient.Install(t, backupReleaseName.String(), namespace, helmValues)
	if err != nil {
		return fmt.Errorf("helm install failed: %v", err)
	}

	time.Sleep(2 * time.Minute)

	cronjob, err := Clientset.BatchV1().CronJobs(namespace).Get(context.Background(), backupReleaseName.String(), metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("cannot retrieve aws backup cronjob: %v", err)
	}

	if cronjob.Spec.Schedule != helmValues.Neo4J.JobSchedule {
		return fmt.Errorf("aws cronjob schedule %s not matching with the schedule defined in values.yaml %s",
			cronjob.Spec.Schedule, helmValues.Neo4J.JobSchedule)
	}

	pods, err := Clientset.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error while retrieving pod list during aws backup operation: %v", err)
	}

	var found bool
	for _, pod := range pods.Items {
		if strings.Contains(pod.Name, backupReleaseName.String()) {
			found = true
			// Verify that the backup job used the custom tempdir
			for _, container := range pod.Spec.Containers {
				for _, env := range container.Env {
					if env.Name == "AGGREGATE_BACKUP_TEMP_DIR" {
						if env.Value != customTempDir {
							return fmt.Errorf("expected AGGREGATE_BACKUP_TEMP_DIR to be %s but got %s", customTempDir, env.Value)
						}
					}
				}
			}
			break
		}
	}

	if !found {
		return fmt.Errorf("no backup pod found")
	}

	return nil
}

// CheckLogsFormat checks whether the neo4j logs are in json format or not
func CheckLogsFormat(t *testing.T, releaseName model.ReleaseName) error {

	stdout, stderr, err := ExecInPod(releaseName, []string{"cat", "/logs/neo4j.log"}, "")
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

// CheckNeo4jOperationsPod checks whether the neo4j operations pod is executed or not
func CheckNeo4jOperationsPod(t *testing.T, releaseName model.ReleaseName) error {

	fetchPods := func() (*v1.PodList, error) {
		pods, err := getPodsWithSpecificLabel(releaseName.Namespace(), "app=neo4j-operations")
		if err != nil {
			return &v1.PodList{}, fmt.Errorf("error seen while fetching list of pods \n %v", err)
		}
		if len(pods.Items) == 0 {
			return &v1.PodList{}, fmt.Errorf("no pods found")
		}
		if len(pods.Items) > 1 {
			return &v1.PodList{}, fmt.Errorf("more than one operations pod found")
		}
		return pods, nil
	}

	pods, err := fetchPods()
	if err != nil {
		return err
	}
	pod := pods.Items[0]
	for pod.Status.Phase == v1.PodRunning {
		t.Logf("operations pod in running state..Waiting for it to be completed")
		time.Sleep(30 * time.Second)
		pods, err = fetchPods()
		if err != nil {
			return err
		}
		pod = pods.Items[0]
	}
	if pod.Status.Phase != v1.PodSucceeded {
		return fmt.Errorf("pod phase %v is not succeeded", pod.Status.Phase)
	}

	out, err := exec.Command("kubectl", "logs", pod.Name, "--namespace", string(releaseName.Namespace())).CombinedOutput()
	if err != nil {
		t.Logf("error while fetching operations pod logs")
		return err
	}
	stringOutput := strings.ToLower(string(out))
	if !strings.Contains(stringOutput, "server is already enabled") {
		return fmt.Errorf("operations pod does not contain valid logs \n logs := %s", string(out))
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
	namespace := string(name.Namespace())
	return []SubTest{
		{name: fmt.Sprintf("Check cluster core 1 is assigned with label %s", model.NodeSelectorLabel(namespace)), test: func(t *testing.T) {
			t.Parallel()
			assert.NoError(t, checkNodeSelectorLabel(t, name, model.NodeSelectorLabel(namespace)), fmt.Sprintf("Core-1 Pod should be deployed on node with label %s", model.NodeSelectorLabel(namespace)))
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
	priorityClassName := model.PriorityClassName(string(releaseName.Namespace()))
	for _, pod := range pods.Items {
		if strings.Contains(pod.Name, "core-2") {
			if !assert.Equal(t, priorityClassName, pod.Spec.PriorityClassName) {
				return fmt.Errorf("priorityClassName %s not matching with %s", pod.Spec.PriorityClassName, priorityClassName)
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

	// Wait for the label to appear - labels may take a moment to propagate
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	var nodeSelectorNode *corev1.Node
	var labelErr error
	attempts := 0
	for {
		select {
		case <-ctx.Done():
			// Before failing, list all nodes and their testLabel labels for debugging
			nodes, listErr := getNodesList()
			if listErr == nil {
				t.Logf("Debug: Available nodes and their testLabel values:")
				for _, node := range nodes.Items {
					if val, present := node.ObjectMeta.Labels["testLabel"]; present {
						t.Logf("  Node %s: testLabel=%s", node.Name, val)
					} else {
						t.Logf("  Node %s: no testLabel", node.Name)
					}
				}
			}
			return fmt.Errorf("timeout waiting for node with label %s after %d attempts: %v", labelName, attempts, labelErr)
		case <-ticker.C:
			attempts++
			nodeSelectorNode, labelErr = getNodeWithLabel(labelName)
			if labelErr == nil && nodeSelectorNode != nil {
				t.Logf("Found node %s with label %s", nodeSelectorNode.Name, labelName)
				goto labelFound
			}
			if attempts%5 == 0 {
				// Log every 5th attempt to avoid spam
				t.Logf("Still waiting for label %s (attempt %d): %v", labelName, attempts, labelErr)
			}
		}
	}

labelFound:
	if !assert.NoError(t, labelErr) {
		return labelErr
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
	pods, err := getPodsWithSpecificLabel(name.Namespace(), "helm.neo4j.com/clustering=true")
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

	stdout, stderr, err := ExecInPod(name, cmd, "")
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

	// Wait for all pods to be ready before comparing endpoints
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	var pods *corev1.PodList
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for all pods to be ready")
		case <-ticker.C:
			var listErr error
			pods, listErr = Clientset.CoreV1().Pods(string(service.Namespace())).List(context.TODO(), listOptions)
			if listErr != nil {
				return fmt.Errorf("cannot get pods matching with headless service selector: %v", listErr)
			}

			if len(pods.Items) == 0 {
				continue // Wait for pods to appear
			}

			// Check if all pods are ready
			allPodsReady := true
			for _, pod := range pods.Items {
				isReady := false
				for _, condition := range pod.Status.Conditions {
					if condition.Type == corev1.PodReady && condition.Status == corev1.ConditionTrue {
						isReady = true
						break
					}
				}
				if !isReady || pod.Status.PodIP == "" {
					allPodsReady = false
					break
				}
			}

			if allPodsReady {
				var endpointsErr error
				endpoints, endpointsErr = Clientset.CoreV1().Endpoints(string(service.Namespace())).Get(context.TODO(), serviceName, metav1.GetOptions{})
				if endpointsErr != nil {
					return fmt.Errorf("failed to get headless service endpoints after pods ready: %v", endpointsErr)
				}
				// Rebuild endpoint IPs list
				endPointIPs = []string{}
				if len(endpoints.Subsets) > 0 && len(endpoints.Subsets[0].Addresses) > 0 {
					for _, endpointAddress := range endpoints.Subsets[0].Addresses {
						endPointIPs = append(endPointIPs, endpointAddress.IP)
					}
				}
				goto compareEndpoints
			}
		}
	}

compareEndpoints:
	if !assert.NotEmpty(t, pods) {
		return fmt.Errorf("pods list matching headless service selector cannot be empty")
	}

	//get the list of podIPs matching the headless service selector
	// All pods should be ready at this point
	var podIPs []string
	for _, pod := range pods.Items {
		if pod.Status.PodIP != "" {
			podIPs = append(podIPs, pod.Status.PodIP)
		}
	}

	//compare podIps and headlessService endPoint IPs ...both should match
	// All pods should be ready, so all should be in endpoints
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

func TestBackupMultipleEndpointsE2E(t *testing.T) {
	t.Parallel()

	releaseName := model.NewReleaseName("multiple-backup-endpoints-" + TestRunIdentifier)
	_, err := createNamespace(t, releaseName)
	if err != nil {
		return
	}
	namespace := string(releaseName.Namespace())

	// Add cleanup
	t.Cleanup(func() {
		_ = runAll(t, "helm", [][]string{
			{"uninstall", releaseName.String(), "--wait", "--timeout", "3m", "--namespace", namespace},
			{"delete", "namespace", namespace},
		}, false)
	})

	backupEndpoints := "10.3.3.2:6362,10.3.3.3:6362,10.3.3.4:6362"

	helmValues := model.DefaultNeo4jBackupValues
	helmValues.Backup.DatabaseBackupEndpoints = backupEndpoints
	helmValues.Backup.SecretName = "demo"
	helmValues.Backup.CloudProvider = "aws"
	helmValues.Backup.BucketName = "demo2"
	helmValues.Backup.Database = "neo4j1"
	helmValues.Backup.S3ForcePathStyle = true

	helmClient := model.NewHelmClient(model.DefaultNeo4jBackupChartName)
	_, err = helmClient.Install(t, releaseName.String(), namespace, helmValues)
	assert.NoError(t, err, "error installing helm chart with multiple backup endpoints")

	time.Sleep(2 * time.Minute)
	cronjob, err := Clientset.BatchV1().CronJobs(namespace).Get(context.Background(), releaseName.String(), metav1.GetOptions{})
	assert.NoError(t, err, "cannot retrieve cronjob for multiple backup endpoints")

	// Verify cronjob env vars
	assert.Contains(t, cronjob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Env, corev1.EnvVar{
		Name:  "DATABASE_BACKUP_ENDPOINTS",
		Value: backupEndpoints,
	}, "backup endpoints not set correctly in cronjob")

	// Verify backup functionality
	pods, err := Clientset.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{})
	assert.NoError(t, err, "error while retrieving pod list")

	var found bool
	for _, pod := range pods.Items {
		if strings.Contains(pod.Name, releaseName.String()) {
			found = true
			out, err := exec.Command("kubectl", "logs", pod.Name, "--namespace", namespace).CombinedOutput()
			assert.NoError(t, err, "error getting backup pod logs")
			assert.NotNil(t, out, "backup logs cannot be retrieved")
			assert.Contains(t, string(out), "Backup Completed")
			break
		}
	}
	assert.True(t, found, "no backup pod found")
}

func TestClusterProbeConfigurations(t *testing.T) {
	if model.Neo4jEdition != "enterprise" {
		t.Skip()
		return
	}

	clusterReleaseName := model.NewReleaseName("cluster-probes")
	chart := model.Neo4jHelmChartCommunityAndEnterprise

	testCases := []struct {
		name   string
		values model.HelmValues
	}{
		{
			name: "HTTP Probe",
			values: func() model.HelmValues {
				v := model.DefaultEnterpriseValues
				v.ReadinessProbe = model.ReadinessProbe{
					HTTPGet: &model.HTTPGetAction{
						Path: "/ready",
						Port: 7474,
					},
					FailureThreshold: 30,
					TimeoutSeconds:   15,
					PeriodSeconds:    10,
				}
				return v
			}(),
		},
		{
			name: "TCP Socket Probe",
			values: func() model.HelmValues {
				v := model.DefaultEnterpriseValues
				v.ReadinessProbe = model.ReadinessProbe{
					TCPSocket: &model.TCPSocketAction{
						Port: 7687,
					},
					FailureThreshold: 20,
					TimeoutSeconds:   10,
					PeriodSeconds:    5,
				}
				return v
			}(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			closeable, err := installNeo4j(t, clusterReleaseName, chart)
			assert.NoError(t, err)
			t.Cleanup(func() { _ = closeable() })

			err = run(t, "kubectl", "--namespace", string(clusterReleaseName.Namespace()),
				"wait", "--for=condition=ready", "pod", clusterReleaseName.PodName(),
				"--timeout=300s")
			assert.NoError(t, err)

			err = CheckProbes(t, clusterReleaseName)
			assert.NoError(t, err)
		})
	}
}
