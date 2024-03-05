package integration_tests

import (
	"context"
	"encoding/base64"
	"fmt"
	"github.com/hashicorp/go-multierror"
	. "github.com/neo4j/helm-charts/internal/helpers"
	"github.com/neo4j/helm-charts/internal/integration_tests/gcloud"
	"github.com/neo4j/helm-charts/internal/model"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"
	"time"
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
func clusterTests(clusterRelease model.ReleaseName) ([]SubTest, error) {

	subTests := []SubTest{
		{name: "Install Backup Helm Chart For AWS", test: func(t *testing.T) {
			t.Parallel()
			assert.NoError(t, InstallNeo4jBackupAWSHelmChartWithNodeSelector(t, clusterRelease), "Backup to AWS should succeed")
		}},
		{name: "Install Backup Helm Chart For GCP With Workload Identity For Cluster", test: func(t *testing.T) {
			assert.NoError(t, InstallNeo4jBackupGCPHelmChartWithWorkloadIdentityForCluster(t, clusterRelease), "Backup to GCP with workload identity should succeed")
		}},
		{name: "Install Backup Helm Chart For AWS Using MinIO", test: func(t *testing.T) {
			t.Parallel()
			assert.NoError(t, InstallNeo4jBackupAWSHelmChartViaMinIO(t, clusterRelease), "Backup to AWS using MinIO should succeed")
		}},
		{name: "Check Cluster Core Logs Format", test: func(t *testing.T) {
			t.Parallel()
			assert.NoError(t, CheckLogsFormat(t, clusterRelease), "Cluster core logs format should be in JSON")
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
	}
	return subTests, nil
}

func InstallNeo4jBackupGCPHelmChartWithWorkloadIdentityForCluster(t *testing.T, clusterReleaseName model.ReleaseName) error {
	if model.Neo4jEdition == "community" {
		t.Skip()
		return nil
	}
	shortName := clusterReleaseName.ShortName()
	backupReleaseName := model.NewReleaseName(fmt.Sprintf("%s-gcp-workload-%s", shortName, TestRunIdentifier))
	gcpServiceAccountName := fmt.Sprintf("%s-%s", gcpServiceAccountNamePrefix, shortName)
	k8sServiceAccountName := fmt.Sprintf("%s-%s", k8sServiceAccountNamePrefix, shortName)
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
			assert.Contains(t, string(out), "Backup Completed for database system !!")
			assert.Contains(t, string(out), "Backup Completed for database neo4j !!")
			assert.Regexp(t, regexp.MustCompile("neo4j(.*)backup uploaded to GCS bucket"), string(out))
			assert.Regexp(t, regexp.MustCompile("system(.*)backup uploaded to GCS bucket"), string(out))
			assert.Regexp(t, regexp.MustCompile("No inconsistencies found !! No Inconsistency report generated."), string(out))
			//assert.Regexp(t, regexp.MustCompile("neo4j(.*)backup.report.tar.gz uploaded to GCS bucket"), string(out))
			//assert.Regexp(t, regexp.MustCompile("system(.*)backup.report.tar.gz uploaded to GCS bucket"), string(out))
			assert.NotContains(t, string(out), "Deleting file")
			break
		}
	}
	assert.Equal(t, true, found, "no gcp workload backup pod found")

	return nil
}

// InstallNeo4jBackupAWSHelmChartWithNodeSelector installs backup cronjob using the given nodeselector labels
func InstallNeo4jBackupAWSHelmChartWithNodeSelector(t *testing.T, releaseName model.ReleaseName) error {
	if model.Neo4jEdition == "community" {
		t.Skip()
		return nil
	}
	backupReleaseName := model.NewReleaseName("cluster-backup-aws-" + TestRunIdentifier)
	namespace := string(releaseName.Namespace())

	t.Cleanup(func() {
		_ = runAll(t, "helm", [][]string{
			{"uninstall", backupReleaseName.String(), "--wait", "--timeout", "3m", "--namespace", namespace},
		}, false)
	})

	bucketName := model.BucketName
	helmClient := model.NewHelmClient(model.DefaultNeo4jBackupChartName)
	helmValues := model.DefaultNeo4jBackupValues
	helmValues.Backup = model.Backup{
		BucketName:               bucketName,
		DatabaseAdminServiceName: fmt.Sprintf("%s-admin", releaseName.String()),
		DatabaseNamespace:        string(releaseName.Namespace()),
		Database:                 "neo4j,system",
		CloudProvider:            "aws",
		SecretName:               "awscred",
		SecretKeyName:            "credentials",
		Verbose:                  true,
		Type:                     "FULL",
	}
	helmValues.NodeSelector = map[string]string{
		"testLabel": "5",
	}
	helmValues.ConsistencyCheck.Database = "neo4j"
	_, err := helmClient.Install(t, backupReleaseName.String(), namespace, helmValues)
	assert.NoError(t, err)

	time.Sleep(2 * time.Minute)
	cronjob, err := Clientset.BatchV1().CronJobs(namespace).Get(context.Background(), backupReleaseName.String(), metav1.GetOptions{})
	assert.NoError(t, err, "cannot retrieve aws backup cronjob")
	assert.Equal(t, cronjob.Spec.Schedule, helmValues.Neo4J.JobSchedule, fmt.Sprintf("aws cronjob schedule %s not matching with the schedule defined in values.yaml %s", cronjob.Spec.Schedule, helmValues.Neo4J.JobSchedule))

	nodeSelectorNode, err := getNodeWithLabel("testLabel=5")
	assert.NoError(t, err)

	pods, err := Clientset.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{})
	assert.NoError(t, err, "error while retrieving pod list during aws backup operation")

	var found bool
	for _, pod := range pods.Items {
		if strings.Contains(pod.Name, "cluster-backup-aws") {
			found = true
			out, err := exec.Command("kubectl", "logs", pod.Name, "--namespace", namespace).CombinedOutput()
			assert.NoError(t, err, "error while getting aws backup pod logs")
			assert.NotNil(t, out, "aws backup logs cannot be retrieved")
			log.Printf("backup logs %v\n", string(out))
			assert.Contains(t, string(out), "Backup Completed for database system !!")
			assert.Contains(t, string(out), "Backup Completed for database neo4j !!")
			assert.Regexp(t, regexp.MustCompile("neo4j(.*)backup uploaded to s3 bucket"), string(out))
			assert.Regexp(t, regexp.MustCompile("system(.*)backup uploaded to s3 bucket"), string(out))
			assert.Regexp(t, regexp.MustCompile("No inconsistencies found !! No Inconsistency report generated."), string(out))
			//assert.Regexp(t, regexp.MustCompile("neo4j(.*)backup.report.tar.gz uploaded to s3 bucket"), string(out))
			//assert.NotRegexp(t, regexp.MustCompile("system(.*)backup.report.tar.gz uploaded to s3 bucket"), string(out))
			assert.Equal(t, nodeSelectorNode.Name, pod.Spec.NodeName, fmt.Sprintf("backup pod %s is not scheduled on the correct node %s", pod.Spec.NodeName, nodeSelectorNode.Name))
			break
		}
	}
	assert.Equal(t, true, found, "no aws backup pod found")
	return nil
}

// InstallNeo4jBackupAWSHelmChartViaMinIO installs backup cronjob and performs backup to minio bucket
func InstallNeo4jBackupAWSHelmChartViaMinIO(t *testing.T, releaseName model.ReleaseName) error {
	if model.Neo4jEdition == "community" {
		t.Skip()
		return nil
	}
	backupReleaseName := model.NewReleaseName("cluster-backup-aws-minio" + TestRunIdentifier)
	namespace := string(releaseName.Namespace())

	t.Cleanup(func() {
		_ = runAll(t, "helm", [][]string{
			{"uninstall", backupReleaseName.String(), "--wait", "--timeout", "3m", "--namespace", namespace},
		}, false)
		deleteMinio(namespace)
	})

	tenantName := "tenant1"
	secretName := "miniocred"
	err := installMinio(namespace, tenantName)
	assert.NoError(t, err, "error while installing minio")

	err = kCreateMinioSecret(namespace, tenantName, secretName)
	assert.NoError(t, err, "error while generating minio kubernetes secret")

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
		MinioEndpoint:            fmt.Sprintf("http://%s-hl.%s.svc.cluster.local:9000", tenantName, namespace),
		Verbose:                  true,
		Type:                     "FULL",
	}
	helmValues.ConsistencyCheck.Database = "neo4j"
	_, err = helmClient.Install(t, backupReleaseName.String(), namespace, helmValues)
	assert.NoError(t, err)

	time.Sleep(2 * time.Minute)
	cronjob, err := Clientset.BatchV1().CronJobs(namespace).Get(context.Background(), backupReleaseName.String(), metav1.GetOptions{})
	assert.NoError(t, err, "cannot retrieve aws backup cronjob")
	assert.Equal(t, cronjob.Spec.Schedule, helmValues.Neo4J.JobSchedule, fmt.Sprintf("aws cronjob schedule %s not matching with the schedule defined in values.yaml %s", cronjob.Spec.Schedule, helmValues.Neo4J.JobSchedule))

	pods, err := Clientset.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{})
	assert.NoError(t, err, "error while retrieving pod list during aws backup operation")

	var found bool
	for _, pod := range pods.Items {
		if strings.Contains(pod.Name, "cluster-backup-aws-minio") {
			found = true
			out, err := exec.Command("kubectl", "logs", pod.Name, "--namespace", namespace).CombinedOutput()
			assert.NoError(t, err, "error while getting aws backup pod logs")
			assert.NotNil(t, out, "aws backup logs cannot be retrieved")
			log.Printf("backup logs %v\n", string(out))
			assert.Contains(t, string(out), "Backup Completed for database system !!")
			assert.Contains(t, string(out), "Backup Completed for database neo4j !!")
			assert.Regexp(t, regexp.MustCompile("neo4j(.*)backup uploaded to s3 bucket"), string(out))
			assert.Regexp(t, regexp.MustCompile("system(.*)backup uploaded to s3 bucket"), string(out))
			assert.Regexp(t, regexp.MustCompile("No inconsistencies found !! No Inconsistency report generated."), string(out))
			//assert.Regexp(t, regexp.MustCompile("neo4j(.*)backup.report.tar.gz uploaded to s3 bucket"), string(out))
			//assert.NotRegexp(t, regexp.MustCompile("system(.*)backup.report.tar.gz uploaded to s3 bucket"), string(out))
			break
		}
	}
	assert.Equal(t, true, found, "no aws minio backup pod found")
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

func installMinio(namespace string, tenantName string) error {
	stdout, stderr, err := RunCommand(exec.Command("kubectl", "minio", "version"))
	if !strings.Contains(strings.ToLower(string(stdout)), "kubectl-plugin version") {
		if err != nil {
			log.Printf("%v", string(stderr))
			return err
		}
	}

	stdout, stderr, err = RunCommand(exec.Command("kubectl", "minio", "init", "-n", namespace))
	if !strings.Contains(strings.ToLower(string(stdout)), "To open Operator UI, start a port forward using this command") {
		if err != nil {
			log.Printf("%v", string(stderr))
			return err
		}
	}

	stdout, stderr, err = RunCommand(exec.Command("kubectl", "minio", "tenant", "create", tenantName, "--servers", "2", "--volumes", "4", "--capacity", "10Gi", "--disable-tls", "-n", namespace))
	if err != nil {
		log.Printf("%v", string(stderr))
		return err
	}

	return nil
}

func deleteMinio(namespace string) error {
	_, stderr, err := RunCommand(exec.Command("kubectl", "minio", "delete", "-f", "-d", "-n", namespace))
	if err != nil {
		log.Printf("%v", string(stderr))
		return err
	}
	return nil
}

func getMiniIOKeys(namespace string, tenantName string) (string, string, error) {
	secretName := fmt.Sprintf("%s-user-1", tenantName)
	stdout, stderr, err := RunCommand(exec.Command("kubectl", "get", "secret", secretName, "-n", namespace, "--template={{.data.CONSOLE_ACCESS_KEY}}"))
	if err != nil {
		log.Printf("%v", string(stderr))
		return "", "", err
	}
	accessKey, err := base64.StdEncoding.DecodeString(string(stdout))
	if err != nil {
		log.Printf("Unable to decode minio access key")
		return "", "", err
	}

	stdout, stderr, err = RunCommand(exec.Command("kubectl", "get", "secret", secretName, "-n", namespace, "--template={{.data.CONSOLE_SECRET_KEY}}"))
	if err != nil {
		log.Printf("%v", string(stderr))
		return "", "", err
	}
	secretKey, err := base64.StdEncoding.DecodeString(string(stdout))
	if err != nil {
		log.Printf("Unable to decode minio secret key")
		return "", "", err
	}
	log.Printf("Access Key = %s , Secret Key = %s", string(accessKey), string(secretKey))
	return string(accessKey), string(secretKey), nil
}

func kCreateMinioSecret(namespace string, tenantName string, secretName string) error {
	accessKey, secretKey, err := getMiniIOKeys(namespace, tenantName)
	if err != nil {
		return err
	}

	tempDir, err := os.MkdirTemp("", namespace)
	if err != nil {
		return err
	}
	path, err := createAwsCredFile(tempDir, accessKey, secretKey)
	if err != nil {
		return err
	}
	_, stderr, err := RunCommand(exec.Command("kubectl", "create", "secret", "-n", namespace, "generic", secretName, fmt.Sprintf("--from-file=credentials=%s", path)))
	if err != nil {
		log.Printf("%v", string(stderr))
		return err
	}

	port, cleanupProxy, err := proxyMinioTenant(namespace, tenantName)
	defer cleanupProxy()
	if err != nil {
		return err
	}

	endpoint := fmt.Sprintf("http://127.0.0.1:%d", port)
	err = createMinioBucket(accessKey, secretKey, endpoint)
	if err != nil {
		return err
	}
	return nil
}

func createMinioBucket(accessKey string, secretKey string, endpoint string) error {
	_, stderr, err := RunCommand(exec.Command("mc", "alias", "set", "myminio", endpoint, accessKey, secretKey))
	if err != nil {
		log.Printf("%v", string(stderr))
		return err
	}
	_, stderr, err = RunCommand(exec.Command("mc", "mb", "myminio/helm-backup-test"))
	if err != nil {
		log.Printf("%v", string(stderr))
		return err
	}
	return nil
}
