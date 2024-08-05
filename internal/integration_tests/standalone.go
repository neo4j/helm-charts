package integration_tests

import (
	"bufio"
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	. "github.com/neo4j/helm-charts/internal/helpers"
	"github.com/neo4j/helm-charts/internal/integration_tests/gcloud"
	"github.com/neo4j/helm-charts/internal/model"
	"github.com/stretchr/testify/assert"
	"io"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"log"
	"math/big"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type SubTest struct {
	name string
	test func(*testing.T)
}

func CheckError(err error) {
	if err != nil {
		log.Panic(err)
	}
}

var (
	Clientset                   *kubernetes.Clientset
	Config                      *restclient.Config
	gcpServiceAccountNamePrefix = "gcp-sa"
	k8sServiceAccountNamePrefix = "k8s-sa"
	mutex                       sync.Mutex
)

func init() {
	//os.Setenv("KUBECONFIG", ".kube/config")
	// gets kubeconfig from env variable
	Config, err := clientcmd.BuildConfigFromFlags("", os.Getenv("KUBECONFIG"))
	CheckError(err)
	Clientset, err = kubernetes.NewForConfig(Config)
	CheckError(err)
}

func publicKey(priv interface{}) interface{} {
	switch k := priv.(type) {
	case *rsa.PrivateKey:
		return &k.PublicKey
	case *ecdsa.PrivateKey:
		return &k.PublicKey
	default:
		return nil
	}
}

func pemBlockForKey(priv interface{}) *pem.Block {
	switch k := priv.(type) {
	case *rsa.PrivateKey:
		return &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(k)}
	case *ecdsa.PrivateKey:
		b, err := x509.MarshalECPrivateKey(k)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to marshal ECDSA private key: %v", err)
			os.Exit(2)
		}
		return &pem.Block{Type: "EC PRIVATE KEY", Bytes: b}
	default:
		return nil
	}
}

func generateCerts(tempDir string) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		log.Fatal(err)
	}
	template, err := buildCert(rand.Reader, priv, time.Now(), big.NewInt(1))
	if err != nil {
		log.Fatal(err)
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, template, template, publicKey(priv), priv)
	if err != nil {
		log.Fatalf("Failed to create certificate: %s", err)
	}
	out := &bytes.Buffer{}
	pem.Encode(out, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})

	f, err := os.Create(tempDir + "/public.crt")

	if err != nil {
		log.Fatal(err)
	}

	_, err = f.WriteString(out.String())
	f.Close()

	if err != nil {
		log.Fatal(err)
	}
	out.Reset()
	pem.Encode(out, pemBlockForKey(priv))
	f, err = os.Create(tempDir + "/private.key")
	if err != nil {
		log.Fatal(err)
	}

	_, err = f.WriteString(out.String())
	f.Close()

	if err != nil {
		log.Fatal(err)
	}
}

func buildCert(random io.Reader, private *ecdsa.PrivateKey, validFrom time.Time, serialNumber *big.Int) (*x509.Certificate, error) {

	template := x509.Certificate{}

	template.Subject = pkix.Name{
		CommonName: string("localhost"),
	}
	template.DNSNames = []string{"localhost", "localhost:7473", "localhost:7687"}
	template.NotBefore = validFrom
	template.NotAfter = validFrom.Add(100 * time.Hour)
	template.KeyUsage = x509.KeyUsageCertSign
	template.IsCA = true
	template.BasicConstraintsValid = true

	template.SerialNumber = serialNumber

	derBytes, err := x509.CreateCertificate(
		random, &template, &template, &private.PublicKey, private)
	if err != nil {
		return nil, fmt.Errorf("Failed to create certificate: %v", err)
	}
	return x509.ParseCertificate(derBytes)
}

func createAwsCredFile(dirName string, accessKey string, secretKey string) (string, error) {
	fileContent := `
[default]
region = us-east-1
`
	fileContent = fileContent + fmt.Sprintf("aws_access_key_id = %s\naws_secret_access_key = %s", accessKey, secretKey)
	filePath := fmt.Sprintf("%s/awscredentials", dirName)
	err := os.WriteFile(filePath, []byte(fileContent), 0666)
	if err != nil {
		return "", err
	}
	return filePath, nil
}

func createAzureCredFile(dirName string) (string, error) {
	fileContent := fmt.Sprintf("AZURE_STORAGE_ACCOUNT_NAME=%s\nAZURE_STORAGE_ACCOUNT_KEY=%s", os.Getenv("AZURE_STORAGE_ACCOUNT_NAME"), os.Getenv("AZURE_STORAGE_ACCOUNT_KEY"))
	filePath := fmt.Sprintf("%s/azurecredentials", dirName)
	err := os.WriteFile(filePath, []byte(fileContent), 0666)
	if err != nil {
		return "", err
	}
	return filePath, nil
}

func createGCPCredFile(dirName string) (string, error) {
	filePath := fmt.Sprintf("%s/gcpcredentials", dirName)
	err := os.WriteFile(filePath, []byte(os.Getenv("GCP_SERVICE_ACCOUNT_CRED")), 0666)
	if err != nil {
		return "", err
	}
	return filePath, nil
}

func kCreateSecret(namespace model.Namespace) ([][]string, Closeable, error) {
	tempDir, err := os.MkdirTemp("", string(namespace))
	closeable := func() error { return os.RemoveAll(tempDir) }
	if err != nil {
		return nil, closeable, err
	}
	generateCerts(tempDir)
	awsCredFileName, err := createAwsCredFile(tempDir, os.Getenv("AWS_ACCESS_KEY_ID"), os.Getenv("AWS_SECRET_ACCESS_KEY"))
	if err != nil {
		return nil, closeable, err
	}
	azureCredFileName, err := createAzureCredFile(tempDir)
	if err != nil {
		return nil, closeable, err
	}
	gcpCredFileName, err := createGCPCredFile(tempDir)
	if err != nil {
		return nil, closeable, err
	}
	return [][]string{
		{"create", "secret", "-n", string(namespace), "generic", model.DefaultAuthSecretName, fmt.Sprintf("--from-literal=NEO4J_AUTH=neo4j/%s", model.DefaultPassword)},
		{"create", "secret", "-n", string(namespace), "generic", "bolt-cert", fmt.Sprintf("--from-file=%s/public.crt", tempDir)},
		{"create", "secret", "-n", string(namespace), "generic", "https-cert", fmt.Sprintf("--from-file=%s/public.crt", tempDir)},
		{"create", "secret", "-n", string(namespace), "generic", "bolt-key", fmt.Sprintf("--from-file=%s/private.key", tempDir)},
		{"create", "secret", "-n", string(namespace), "generic", "https-key", fmt.Sprintf("--from-file=%s/private.key", tempDir)},
		{"create", "secret", "-n", string(namespace), "generic", "bloom-license", fmt.Sprintf("--from-literal=bloom.license=%s", os.Getenv("BLOOM_LICENSE"))},
		{"create", "secret", "-n", string(namespace), "generic", "ldapsecret", "--from-literal=LDAP_PASS=demo123"},
		{"create", "secret", "-n", string(namespace), "generic", "awscred", fmt.Sprintf("--from-file=credentials=%s", awsCredFileName)},
		{"create", "secret", "-n", string(namespace), "generic", "azurecred", fmt.Sprintf("--from-file=credentials=%s", azureCredFileName)},
		{"create", "secret", "-n", string(namespace), "generic", "gcpcred", fmt.Sprintf("--from-file=credentials=%s", gcpCredFileName)},
		{"create", "secret", "-n", string(namespace), "tls", "ingress-secret", fmt.Sprintf("--key=%s/%s", tempDir, "private.key"), fmt.Sprintf("--cert=%s/%s", tempDir, "public.crt")},
	}, closeable, err
}

func helmCleanupCommands(releaseName model.ReleaseName) [][]string {
	return [][]string{
		{"uninstall", releaseName.String(), "--wait", "--timeout", "2m", "--namespace", string(releaseName.Namespace())},
	}
}

func kCleanupCommands(namespace model.Namespace) [][]string {
	return [][]string{{"delete", "namespace", string(namespace), "--ignore-not-found", "--force", "--grace-period=0"}}
}

var portOffset int32 = 0

func proxyBolt(t *testing.T, releaseName model.ReleaseName, connectToPod bool) (int32, Closeable, error) {
	localHttpPort := 9000 + atomic.AddInt32(&portOffset, 1)
	localBoltPort := 9100 + atomic.AddInt32(&portOffset, 1)

	program := "kubectl"

	args := []string{"--namespace", string(releaseName.Namespace()), "port-forward", fmt.Sprintf("pod/%s", releaseName.PodName()), fmt.Sprintf("%d:7474", localHttpPort), fmt.Sprintf("%d:7687", localBoltPort)}
	if !connectToPod {
		args = []string{"--namespace", string(releaseName.Namespace()), "port-forward", fmt.Sprintf("service/%s-lb-neo4j", model.DefaultNeo4jName), fmt.Sprintf("%d:7474", localHttpPort), fmt.Sprintf("%d:7687", localBoltPort)}
	}

	t.Logf("running: %s %s\n", program, args)
	cmd := exec.Command(program, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return localBoltPort, nil, err
	}
	// Use the same pipe for standard error
	cmd.Stderr = cmd.Stdout

	// Make a new channel which will be used to signal that we are ready
	started := make(chan struct{})

	// Create a scanner which scans in a line-by-line fashion
	scanner := bufio.NewScanner(stdout)

	// Use the scanner to scan the output line by line and log it
	// It's running in a goroutine so that it doesn't block
	go func() {
		var once sync.Once
		notifyStarted := func() { started <- struct{}{} }

		// We're all done, unblock the channel
		defer func() {
			once.Do(notifyStarted)
		}()

		// Read line by line and process it until we see that Forwarding has begun
		for scanner.Scan() {
			line := scanner.Text()
			t.Log("PortForward:", line)
			if strings.HasPrefix(line, "Forwarding from") {
				once.Do(notifyStarted)
			}
		}
		scannerErr := scanner.Err()
		if scannerErr != nil {
			t.Logf("Scanner logged error %s - this is usually expected when the proxy is terminated", scannerErr)
		}
	}()

	// Start the command and check for errors
	err = cmd.Start()
	if err == nil {
		// Wait for output to indicate we actually started forwarding
		<-started
		if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
			err = fmt.Errorf("port forward process exited unexpectedly")
		}
	}

	return localBoltPort, func() error {
		var cmdErr = cmd.Process.Kill()
		if cmdErr != nil {
			t.Log("failed to kill process: ", cmdErr)
		}
		stdout.Close()
		return cmdErr
	}, err
}

func proxyMinioTenant(namespace string, tenantName string) (int, Closeable, error) {

	time.Sleep(1 * time.Minute)
	localPort := 9000
	program := "kubectl"
	args := []string{"--namespace", namespace, "port-forward", fmt.Sprintf("svc/%s-hl", tenantName), fmt.Sprintf("%d:9000", localPort)}

	log.Printf("running: %s %s\n", program, args)
	cmd := exec.Command(program, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return localPort, nil, err
	}
	// Use the same pipe for standard error
	cmd.Stderr = cmd.Stdout

	// Make a new channel which will be used to signal that we are ready
	started := make(chan struct{})

	// Create a scanner which scans in a line-by-line fashion
	scanner := bufio.NewScanner(stdout)

	// Use the scanner to scan the output line by line and log it
	// It's running in a goroutine so that it doesn't block
	go func() {
		var once sync.Once
		notifyStarted := func() { started <- struct{}{} }

		// We're all done, unblock the channel
		defer func() {
			once.Do(notifyStarted)
		}()

		// Read line by line and process it until we see that Forwarding has begun
		for scanner.Scan() {
			line := scanner.Text()
			log.Printf("PortForward:%s", line)
			if strings.HasPrefix(line, "Forwarding from") {
				once.Do(notifyStarted)
			}
		}
		scannerErr := scanner.Err()
		if scannerErr != nil {
			log.Printf("Scanner logged error %s - this is usually expected when the proxy is terminated", scannerErr)
		}
	}()

	// Start the command and check for errors
	err = cmd.Start()
	if err == nil {
		// Wait for output to indicate we actually started forwarding
		<-started
		if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
			err = fmt.Errorf("port forward process exited unexpectedly")
		}
	}

	return localPort, func() error {
		var cmdErr = cmd.Process.Kill()
		if cmdErr != nil {
			log.Printf("failed to kill process: %v", cmdErr)
		}
		stdout.Close()
		return cmdErr
	}, err
}

func runAll(t *testing.T, bin string, commands [][]string, failFast bool) error {
	var combinedErrors error
	for _, command := range commands {
		err := run(t, bin, command...)
		if err != nil {
			if failFast {
				return err
			} else {
				combinedErrors = CombineErrors(combinedErrors, fmt.Errorf("error: '%s' running %s %s", err, bin, command))
			}
		}
	}
	return combinedErrors
}

func createNamespace(t *testing.T, releaseName model.ReleaseName) (Closeable, error) {
	err := run(t, "kubectl", "create", "ns", string(releaseName.Namespace()))
	return func() error {
		return runAll(t, "kubectl", kCleanupCommands(releaseName.Namespace()), false)
	}, err
}

// createPriorityClass create priority class to test the priorityClassName feature
func createPriorityClass(t *testing.T, releaseName model.ReleaseName) (Closeable, error) {
	//kubectl create priorityclass high-priority-<namespace> --value=1000 --description="high priority -n <namespace>"
	priorityClassName := model.PriorityClassName(string(releaseName.Namespace()))
	err := run(t, "kubectl", "create", "priorityclass", priorityClassName, "--value=1000", "--description=\"high priority\"", "-n", string(releaseName.Namespace()))
	return func() error {
		return runAll(t, "kubectl",
			[][]string{{"delete", "priorityClass", priorityClassName, "--force", "--grace-period=0"}},
			false)
	}, err
}

func run(t *testing.T, command string, args ...string) error {
	t.Logf("running: %s %s\n", command, args)
	out, err := exec.Command(command, args...).CombinedOutput()
	if out != nil {
		t.Logf("output: %s\n", out)
	}
	return err
}

func AsCloseable(closeables []Closeable) Closeable {
	return func() error {
		var combinedErrors error
		if closeables != nil {
			for _, closeable := range closeables {
				innerErr := closeable()
				if innerErr != nil {
					combinedErrors = CombineErrors(combinedErrors, innerErr)
				}
			}
		}
		return combinedErrors
	}
}

func InstallNeo4jInGcloud(t *testing.T, zone gcloud.Zone, project gcloud.Project, releaseName model.ReleaseName, chart model.Neo4jHelmChartBuilder, extraHelmInstallArgs ...string) (Closeable, error) {

	var closeables []Closeable
	addCloseable := func(closeable Closeable) {
		closeables = append([]Closeable{closeable}, closeables...)
	}

	completed := false
	// This is here to ensure that closeables are closed if there is a panic
	defer func() (err error) {
		if !completed {
			err = AsCloseable(closeables)()
			t.Log(err)
		}
		return err
	}()

	cleanupGcloud, diskName, err := gcloud.InstallGcloud(t, zone, project, releaseName)
	createPersistentVolume(diskName, zone, project, releaseName)
	if err != nil {
		return AsCloseable(closeables), err
	}
	addCloseable(cleanupGcloud)
	// delete the statefulset like this otherwise the pods will hang around for their termination grace period
	addCloseable(func() error {
		return runAll(t, "kubectl", [][]string{
			{"delete", "statefulset", releaseName.String(), "--namespace", string(releaseName.Namespace()), "--grace-period=0", "--force", "--ignore-not-found"},
			{"delete", "pod", releaseName.PodName(), "--namespace", string(releaseName.Namespace()), "--grace-period=0", "--wait", "--timeout=120s", "--ignore-not-found"},
			{"delete", "pvc", fmt.Sprintf("%s-pvc", releaseName.String()), "--grace-period=0", "--wait", "--timeout=120s", "--ignore-not-found"},
			{"delete", "pv", fmt.Sprintf("%s-pv", releaseName.String()), "--grace-period=0", "--wait", "--timeout=10s", "--ignore-not-found"},
		}, false)
	})
	addCloseable(func() error { return runAll(t, "helm", helmCleanupCommands(releaseName), false) })

	err = run(t, "helm", model.BaseHelmCommand("install", releaseName, chart, model.Neo4jEdition, extraHelmInstallArgs...)...)

	if err != nil {
		return AsCloseable(closeables), err
	}

	completed = true
	return AsCloseable(closeables), err
}

func createPersistentVolume(name *model.PersistentDiskName, zone gcloud.Zone, project gcloud.Project, release model.ReleaseName) (*v1.PersistentVolumeClaim, error) {
	pv := &v1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-pv", release.String()),
			Namespace: string(release.Namespace()),
		},
		Spec: v1.PersistentVolumeSpec{
			Capacity: v1.ResourceList{
				v1.ResourceStorage: *resource.NewQuantity(10*1024*1024*1024, resource.BinarySI),
			},
			PersistentVolumeSource: v1.PersistentVolumeSource{
				CSI: &v1.CSIPersistentVolumeSource{
					Driver:       "pd.csi.storage.gke.io",
					VolumeHandle: fmt.Sprintf("projects/%s/zones/%s/disks/%s", project, zone, string(*name)),
					FSType:       "ext4",
				},
			},
			AccessModes:                   []v1.PersistentVolumeAccessMode{v1.ReadWriteOnce},
			PersistentVolumeReclaimPolicy: v1.PersistentVolumeReclaimDelete,
			ClaimRef: &v1.ObjectReference{
				Kind:       "PersistentVolumeClaim",
				Namespace:  string(release.Namespace()),
				Name:       fmt.Sprintf("%s-pvc", release.String()),
				APIVersion: "v1",
			},

			StorageClassName: "",
		},
	}
	pvc := &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-pvc", release.String()),
			Namespace: string(release.Namespace()),
		},
		Spec: v1.PersistentVolumeClaimSpec{
			AccessModes: pv.Spec.AccessModes,
			Resources: v1.VolumeResourceRequirements{
				Requests: pv.Spec.Capacity,
			},
			VolumeName:       pv.Name,
			StorageClassName: &pv.Spec.StorageClassName,
		},
	}
	_, err := Clientset.CoreV1().PersistentVolumes().Create(context.TODO(), pv, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	return Clientset.CoreV1().PersistentVolumeClaims(string(release.Namespace())).Create(context.TODO(), pvc, metav1.CreateOptions{})
}

func prepareK8s(t *testing.T, releaseName model.ReleaseName) (Closeable, error) {
	var closeables []Closeable

	addCloseable := func(closeable Closeable) {
		closeables = append([]Closeable{closeable}, closeables...)
	}

	cleanupNamespace, err := createNamespace(t, releaseName)
	addCloseable(cleanupNamespace)
	if err != nil {
		return AsCloseable(closeables), err
	}

	createSecretCommands, cleanupCertificates, err := kCreateSecret(releaseName.Namespace())
	addCloseable(cleanupCertificates)
	if err != nil {
		return AsCloseable(closeables), err
	}

	err = runAll(t, "kubectl", createSecretCommands, true)
	if err != nil {
		return AsCloseable(closeables), err
	}

	return AsCloseable(closeables), nil
}

func runSubTests(t *testing.T, subTests []SubTest) {
	t.Cleanup(func() { t.Logf("Finished running all tests in '%s'", t.Name()) })

	for _, test := range subTests {

		t.Run(test.name, func(t *testing.T) {
			t.Logf("Started running subtest '%s'", t.Name())
			t.Cleanup(func() { t.Logf("Finished running subtest '%s'", t.Name()) })
			test.test(t)
		})
	}
}

func installNeo4j(t *testing.T, releaseName model.ReleaseName, chart model.Neo4jHelmChartBuilder, extraHelmInstallArgs ...string) (Closeable, error) {
	closeables := []Closeable{}
	addCloseable := func(closeable Closeable) {
		closeables = append([]Closeable{closeable}, closeables...)
	}

	closeable, err := prepareK8s(t, releaseName)
	addCloseable(closeable)
	if err != nil {
		return AsCloseable(closeables), err
	}

	closeable, err = InstallNeo4jInGcloud(t, gcloud.CurrentZone(), gcloud.CurrentProject(), releaseName, chart, extraHelmInstallArgs...)
	addCloseable(closeable)
	if err != nil {
		return AsCloseable(closeables), err
	}

	err = run(t, "kubectl", "--namespace", string(releaseName.Namespace()), "rollout", "status", "--watch", "--timeout=120s", "statefulset/"+releaseName.String())
	return AsCloseable(closeables), err
}

func k8sTests(name model.ReleaseName, chart model.Neo4jHelmChartBuilder) ([]SubTest, error) {
	expectedConfiguration, err := (&model.Neo4jConfiguration{}).PopulateFromFile(Neo4jConfFile)
	if err != nil {
		return nil, err
	}
	log.Printf("%v", expectedConfiguration)
	return []SubTest{
		{name: "Check Neo4j Logs For Any Errors", test: func(t *testing.T) {
			t.Parallel()
			assert.NoError(t, checkNeo4jLogsForAnyErrors(t, name), "Neo4j Logs check should succeed")
		}},
		{name: "Check Neo4j Configuration", test: func(t *testing.T) {
			assert.NoError(t, checkNeo4jConfiguration(t, name, expectedConfiguration), "Neo4j Config check should succeed")
		}},
		{name: "Check Bloom Version", test: func(t *testing.T) { assert.NoError(t, checkBloomVersion(t, name), "Retrieve a valid BLOOM version") }},
		{name: "Create Node", test: func(t *testing.T) { assert.NoError(t, createNode(t, name), "Create Node should succeed") }},
		{name: "Delete Resources", test: func(t *testing.T) { assert.NoError(t, ResourcesCleanup(t, name), "Cleanup Resources should succeed") }},
		{name: "Reinstall Resources", test: func(t *testing.T) {
			assert.NoError(t, ResourcesReinstall(t, name, chart), "Reinstall Resources should succeed")
		}},
		{name: "Count Nodes", test: func(t *testing.T) { assert.NoError(t, checkNodeCount(t, name), "Count Nodes should succeed") }},
		{name: "Check Probes", test: func(t *testing.T) { assert.NoError(t, CheckProbes(t, name), "Probes Matching should succeed") }},
		{name: "Check Service Annotations", test: func(t *testing.T) {
			assert.NoError(t, CheckServiceAnnotations(t, name, chart), "Services should have annotations")
		}},
		{name: "Check RunAsNonRoot", test: func(t *testing.T) { assert.NoError(t, RunAsNonRoot(t, name), "RunAsNonRoot check should succeed") }},
		{name: "Exec in Pod", test: func(t *testing.T) { assert.NoError(t, CheckExecInPod(t, name), "Exec in Pod should succeed") }},
		{name: "Install Backup Helm Chart For GCP With Inconsistencies", test: func(t *testing.T) {
			assert.NoError(t, InstallNeo4jBackupGCPHelmChartWithInconsistencies(t, name), "Backup to GCP should succeed along with upload of inconsistencies report")
		}},
		{name: "Install Backup Helm Chart For GCP With Workload Identity", test: func(t *testing.T) {
			t.Parallel()
			assert.NoError(t, InstallNeo4jBackupGCPHelmChartWithWorkloadIdentity(t, name), "Backup to GCP with workload identity should succeed")
		}},
		{name: "Install Backup Helm Chart For AWS", test: func(t *testing.T) {
			t.Parallel()
			assert.NoError(t, InstallNeo4jBackupAWSHelmChart(t, name), "Backup to AWS should succeed")
		}},
		{name: "Install Backup Helm Chart For Azure", test: func(t *testing.T) {
			t.Parallel()
			assert.NoError(t, InstallNeo4jBackupAzureHelmChart(t, name), "Backup to Azure should succeed")
		}},
		{name: "Install Backup Helm Chart For GCP", test: func(t *testing.T) {
			t.Parallel()
			assert.NoError(t, InstallNeo4jBackupGCPHelmChart(t, name), "Backup to GCP should succeed")
		}},
		{name: "Install Reverse Proxy Helm Chart", test: func(t *testing.T) {
			t.Parallel()
			assert.NoError(t, InstallReverseProxyHelmChart(t, name), "Reverse Proxy installation with ingress should succeed")
		}},
	}, err
}

func InstallNeo4jBackupAWSHelmChart(t *testing.T, standaloneReleaseName model.ReleaseName) error {
	if model.Neo4jEdition == "community" {
		t.Skip()
		return nil
	}
	backupReleaseName := model.NewReleaseName("standalone-backup-aws-" + TestRunIdentifier)
	backupBucketName := fmt.Sprintf("helm-charts-%s", TestRunIdentifier)
	namespace := string(standaloneReleaseName.Namespace())

	t.Cleanup(func() {
		_ = runAll(t, "helm", [][]string{
			{"uninstall", backupReleaseName.String(), "--wait", "--timeout", "3m", "--namespace", namespace},
		}, false)
		_ = deleteAWSBucket(os.Getenv("AWS_ACCESS_KEY_ID"), os.Getenv("AWS_SECRET_ACCESS_KEY"), "us-east-1", backupBucketName)
	})

	err := createAWSBucket(os.Getenv("AWS_ACCESS_KEY_ID"), os.Getenv("AWS_SECRET_ACCESS_KEY"), "us-east-1", backupBucketName)
	if err != nil {
		return err
	}

	helmClient := model.NewHelmClient(model.DefaultNeo4jBackupChartName)
	helmValues := model.DefaultNeo4jBackupValues
	helmValues.Backup = model.Backup{
		BucketName:               backupBucketName,
		DatabaseAdminServiceName: fmt.Sprintf("%s-admin", standaloneReleaseName.String()),
		DatabaseNamespace:        string(standaloneReleaseName.Namespace()),
		Database:                 "neo4j,system",
		CloudProvider:            "aws",
		SecretName:               "awscred",
		SecretKeyName:            "credentials",
		Verbose:                  true,
		KeepBackupFiles:          true,
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
		if strings.Contains(pod.Name, "standalone-backup-aws") {
			found = true
			out, err := exec.Command("kubectl", "logs", pod.Name, "--namespace", namespace).CombinedOutput()
			assert.NoError(t, err, "error while getting aws backup pod logs")
			assert.NotNil(t, out, "aws backup logs cannot be retrieved")
			assert.Contains(t, string(out), "Backup Completed for database neo4j system !!")
			assert.Regexp(t, regexp.MustCompile("neo4j(.*)backup uploaded to s3 bucket"), string(out))
			assert.Regexp(t, regexp.MustCompile("system(.*)backup uploaded to s3 bucket"), string(out))
			assert.Regexp(t, regexp.MustCompile("No inconsistencies found"), string(out))
			break
		}
	}
	assert.Equal(t, true, found, "no aws backup pod found")

	aggregateBackupReleaseName := model.NewReleaseName("standalone-aggregate-aws-" + TestRunIdentifier)
	helmValues.Backup = model.Backup{
		CloudProvider: "aws",
		SecretName:    "awscred",
		SecretKeyName: "credentials",
		AggregateBackup: model.AggregateBackup{
			Enabled:  true,
			Verbose:  true,
			FromPath: fmt.Sprintf("s3://%s", backupBucketName),
			Database: "neo4j",
		},
	}
	_, err = helmClient.Install(t, aggregateBackupReleaseName.String(), namespace, helmValues)
	assert.NoError(t, err)

	time.Sleep(2 * time.Minute)
	cronjobs, err := Clientset.BatchV1().CronJobs(namespace).List(context.Background(),
		metav1.ListOptions{
			TypeMeta:      metav1.TypeMeta{},
			LabelSelector: "app.kubernetes.io/component=aggregate-backup",
		})
	assert.NoError(t, err, "cannot retrieve aws aggregate backup cronjob")
	assert.NotEqual(t, len(cronjobs.Items), 0)

	pods, err = Clientset.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{})
	assert.NoError(t, err, "error while retrieving pod list during aws backup operation")

	found = false
	for _, pod := range pods.Items {
		if strings.Contains(pod.Name, "standalone-aggregate-aws") {
			found = true
			out, err := exec.Command("kubectl", "logs", pod.Name, "--namespace", namespace).CombinedOutput()
			assert.NoError(t, err, "error while getting aws backup pod logs")
			assert.NotNil(t, out, "aws backup logs cannot be retrieved")
			assert.Contains(t, string(out), "Found backup chain with no diffs, no need to aggregate")
			break
		}
	}
	assert.Equal(t, true, found, "no aggregate backup pod found")

	return nil
}

func InstallNeo4jBackupAzureHelmChart(t *testing.T, standaloneReleaseName model.ReleaseName) error {
	if model.Neo4jEdition == "community" {
		t.Skip()
		return nil
	}
	backupReleaseName := model.NewReleaseName("standalone-backup-azure-" + TestRunIdentifier)
	namespace := string(standaloneReleaseName.Namespace())

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
		DatabaseAdminServiceName: fmt.Sprintf("%s-admin", standaloneReleaseName.String()),
		DatabaseNamespace:        string(standaloneReleaseName.Namespace()),
		Database:                 "neo4j,system",
		CloudProvider:            "azure",
		SecretName:               "azurecred",
		SecretKeyName:            "credentials",
		Verbose:                  true,
		Type:                     "FULL",
	}
	helmValues.ConsistencyCheck.Database = "system"
	_, err := helmClient.Install(t, backupReleaseName.String(), namespace, helmValues)
	assert.NoError(t, err)

	time.Sleep(2 * time.Minute)
	cronjob, err := Clientset.BatchV1().CronJobs(namespace).Get(context.Background(), backupReleaseName.String(), metav1.GetOptions{})
	assert.NoError(t, err, "cannot retrieve azure backup cronjob")
	assert.Equal(t, cronjob.Spec.Schedule, helmValues.Neo4J.JobSchedule, fmt.Sprintf("azure cronjob schedule %s not matching with the schedule defined in values.yaml %s", cronjob.Spec.Schedule, helmValues.Neo4J.JobSchedule))

	pods, err := Clientset.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{})
	assert.NoError(t, err, "error while retrieving pod list during azure backup operation")

	var found bool
	for _, pod := range pods.Items {
		if strings.Contains(pod.Name, "standalone-backup-azure") {
			found = true
			out, err := exec.Command("kubectl", "logs", pod.Name, "--namespace", namespace).CombinedOutput()
			assert.NoError(t, err, "error while getting azure backup pod logs")
			assert.NotNil(t, out, "azure backup logs cannot be retrieved")
			assert.Contains(t, string(out), "Backup Completed for database neo4j system !!")
			assert.Regexp(t, regexp.MustCompile("neo4j(.*)backup uploaded to azure container"), string(out))
			assert.Regexp(t, regexp.MustCompile("system(.*)backup uploaded to azure container"), string(out))
			assert.Regexp(t, regexp.MustCompile("No inconsistencies found"), string(out))
			break
		}
	}
	assert.Equal(t, true, found, "no azure backup pod found")
	return nil
}

func InstallNeo4jBackupGCPHelmChart(t *testing.T, standaloneReleaseName model.ReleaseName) error {
	if model.Neo4jEdition == "community" {
		t.Skip()
		return nil
	}
	backupReleaseName := model.NewReleaseName("standalone-backup-gcp-" + TestRunIdentifier)
	namespace := string(standaloneReleaseName.Namespace())

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
		DatabaseAdminServiceName: fmt.Sprintf("%s-admin", standaloneReleaseName.String()),
		DatabaseNamespace:        string(standaloneReleaseName.Namespace()),
		Database:                 "neo4j",
		CloudProvider:            "gcp",
		SecretName:               "gcpcred",
		SecretKeyName:            "credentials",
		Verbose:                  true,
		Type:                     "FULL",
		KeepBackupFiles:          true,
	}

	_, err := helmClient.Install(t, backupReleaseName.String(), namespace, helmValues)
	assert.NoError(t, err)

	time.Sleep(2 * time.Minute)
	cronjob, err := Clientset.BatchV1().CronJobs(namespace).Get(context.Background(), backupReleaseName.String(), metav1.GetOptions{})
	assert.NoError(t, err, "cannot retrieve gcp backup cronjob")
	assert.Equal(t, cronjob.Spec.Schedule, helmValues.Neo4J.JobSchedule, fmt.Sprintf("gcp cronjob schedule %s not matching with the schedule defined in values.yaml %s", cronjob.Spec.Schedule, helmValues.Neo4J.JobSchedule))

	pods, err := Clientset.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{})
	assert.NoError(t, err, "error while retrieving pod list during gcp backup operation")

	var found bool
	for _, pod := range pods.Items {
		if strings.Contains(pod.Name, "standalone-backup-gcp") {
			found = true
			out, err := exec.Command("kubectl", "logs", pod.Name, "--namespace", namespace).CombinedOutput()
			assert.NoError(t, err, "error while getting gcp backup pod logs")
			assert.NotNil(t, out, "gcp backup logs cannot be retrieved")
			assert.Contains(t, string(out), "Backup Completed for database neo4j !!")
			assert.Regexp(t, regexp.MustCompile("neo4j(.*)backup uploaded to GCS bucket"), string(out))
			assert.Regexp(t, regexp.MustCompile("No inconsistencies found"), string(out))
			assert.NotContains(t, string(out), "Deleting file")
			break
		}
	}
	assert.Equal(t, true, found, "no gcp backup pod found")
	return nil
}

func InstallNeo4jBackupGCPHelmChartWithInconsistencies(t *testing.T, standaloneReleaseName model.ReleaseName) error {

	if model.Neo4jEdition == "community" {
		t.Skip()
		return nil
	}

	err := introduceInconsistency(t, standaloneReleaseName)
	if !assert.NoError(t, err) {
		return err
	}

	backupReleaseName := model.NewReleaseName("standalone-backup-gcp-incon-" + TestRunIdentifier)
	namespace := string(standaloneReleaseName.Namespace())

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
		DatabaseAdminServiceName: fmt.Sprintf("%s-admin", standaloneReleaseName.String()),
		DatabaseNamespace:        string(standaloneReleaseName.Namespace()),
		Database:                 "neo4j,system",
		CloudProvider:            "gcp",
		SecretName:               "gcpcred",
		SecretKeyName:            "credentials",
		Verbose:                  true,
		Type:                     "FULL",
		KeepBackupFiles:          true,
	}
	helmValues.ConsistencyCheck.Database = "neo4j,system"

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
		if strings.Contains(pod.Name, "standalone-backup-gcp-incon-") {
			found = true
			out, err := exec.Command("kubectl", "logs", pod.Name, "--namespace", namespace).CombinedOutput()
			assert.NoError(t, err, "error while getting gcp backup pod logs")
			assert.NotNil(t, out, "gcp backup logs cannot be retrieved")
			assert.Contains(t, string(out), "Backup Completed for database neo4j system !!")
			assert.Regexp(t, regexp.MustCompile("neo4j(.*)backup uploaded to GCS bucket"), string(out))
			assert.Regexp(t, regexp.MustCompile("system(.*)backup uploaded to GCS bucket"), string(out))
			assert.Regexp(t, regexp.MustCompile("neo4j(.*)backup.report.tar.gz uploaded to GCS bucket"), string(out))
			assert.Regexp(t, regexp.MustCompile("Inconsistencies found for neo4j database"), string(out))
			assert.Regexp(t, regexp.MustCompile("No inconsistencies found for system database !! No Inconsistency report generated."), string(out))
			assert.NotContains(t, string(out), "Deleting file")
			break
		}
	}
	assert.Equal(t, true, found, "no gcp backup pod found")

	err = revertInconsistency(standaloneReleaseName)
	assert.NoError(t, err, "error seen while reverting inconsistency")
	return nil
}

func InstallNeo4jBackupGCPHelmChartWithWorkloadIdentity(t *testing.T, standaloneReleaseName model.ReleaseName) error {
	if model.Neo4jEdition == "community" {
		t.Skip()
		return nil
	}
	shortName := standaloneReleaseName.ShortName()
	currentUnixTime := time.Now().Unix()
	backupReleaseName := model.NewReleaseName(fmt.Sprintf("%s-gcp-workload-%s", shortName, TestRunIdentifier))
	gcpServiceAccountName := fmt.Sprintf("%s-%d", gcpServiceAccountNamePrefix, currentUnixTime)
	k8sServiceAccountName := fmt.Sprintf("%s-%d", k8sServiceAccountNamePrefix, currentUnixTime)
	namespace := string(standaloneReleaseName.Namespace())

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
				"iam.gke.io/gcp-service-account": fmt.Sprintf("%s@%s.iam.gserviceaccount.com", gcpServiceAccountName, string(gcloud.CurrentProject())),
			},
		},
	}

	_, err := Clientset.CoreV1().ServiceAccounts(namespace).Create(context.Background(), &serviceAccount, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("error seen while creating k8s service account %s. \n Err := %v", k8sServiceAccountName, err)
	}

	err = createGCPServiceAccount(k8sServiceAccountName, namespace, gcpServiceAccountName)
	if err != nil {
		return fmt.Errorf("error seen while creating GCP service account. \n Err := %v", err)
	}

	bucketName := model.BucketName
	helmClient := model.NewHelmClient(model.DefaultNeo4jBackupChartName)
	helmValues := model.DefaultNeo4jBackupValues
	helmValues.Backup = model.Backup{
		BucketName:               bucketName,
		DatabaseAdminServiceName: fmt.Sprintf("%s-admin", standaloneReleaseName.String()),
		DatabaseNamespace:        string(standaloneReleaseName.Namespace()),
		Database:                 "neo4j,system",
		CloudProvider:            "gcp",
		Verbose:                  true,
		Type:                     "FULL",
		KeepBackupFiles:          true,
	}
	helmValues.ServiceAccountName = k8sServiceAccountName

	_, err = helmClient.Install(t, backupReleaseName.String(), namespace, helmValues)
	if err != nil {
		return err
	}

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
			assert.Contains(t, string(out), "Backup Completed for database neo4j system !!")
			assert.Regexp(t, regexp.MustCompile("neo4j(.*)backup uploaded to GCS bucket"), string(out))
			assert.Regexp(t, regexp.MustCompile("system(.*)backup uploaded to GCS bucket"), string(out))
			assert.Regexp(t, regexp.MustCompile("No inconsistencies found"), string(out))
			assert.NotContains(t, string(out), "Deleting file")
			break
		}
	}
	assert.Equal(t, true, found, "no gcp workload backup pod found")

	return nil
}

func InstallReverseProxyHelmChart(t *testing.T, standaloneReleaseName model.ReleaseName) error {
	if model.Neo4jEdition == "community" {
		t.Skip()
		return nil
	}
	reverseProxyReleaseName := model.NewReleaseName("rp-" + TestRunIdentifier)
	namespace := string(standaloneReleaseName.Namespace())

	t.Cleanup(func() {
		_ = runAll(t, "helm", [][]string{
			{"uninstall", reverseProxyReleaseName.String(), "--wait", "--timeout", "3m", "--namespace", namespace},
		}, false)
	})

	helmClient := model.NewHelmClient(model.DefaultNeo4jReverseProxyChartName)
	helmValues := model.DefaultNeo4jReverseProxyValues
	helmValues.ReverseProxy.ServiceName = fmt.Sprintf("%s-admin", standaloneReleaseName.String())
	helmValues.ReverseProxy.Namespace = namespace

	//installing nginx ingress controller
	err := run(t, "helm", "upgrade", "--install", "ingress-nginx", "ingress-nginx", "--repo", "https://kubernetes.github.io/ingress-nginx", "--namespace", "ingress-nginx", "--create-namespace")
	assert.NoError(t, err)
	time.Sleep(1 * time.Minute)

	_, err = helmClient.Install(t, reverseProxyReleaseName.String(), namespace, helmValues)
	assert.NoError(t, err)

	time.Sleep(1 * time.Minute)

	reverseProxyDepName := fmt.Sprintf("%s-reverseproxy-dep", reverseProxyReleaseName.String())
	deployment, err := Clientset.AppsV1().Deployments(namespace).Get(context.Background(), reverseProxyDepName, metav1.GetOptions{})
	assert.NoError(t, err, "cannot retrieve reverse proxy pod")
	assert.NotNil(t, deployment, "no reverse proxy deployment found")

	pods, err := Clientset.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{
		LabelSelector: fmt.Sprintf("name=%s-reverseproxy", reverseProxyReleaseName.String()),
	})
	assert.NoError(t, err, "cannot retrieve reverse proxy pod")
	assert.NotNil(t, pods, "no reverse proxy pods found")
	assert.Equal(t, len(pods.Items), 1, "more than 1 reverse proxy pods found")

	cmd := []string{"ls", "-lst", "/go"}
	stdoutCmd, _, err := ExecInPod(standaloneReleaseName, cmd, pods.Items[0].Name)
	assert.NoError(t, err, "cannot exec in reverse proxy pod")
	assert.NotContains(t, stdoutCmd, "root")
	assert.Contains(t, stdoutCmd, "neo4j")

	ingressName := fmt.Sprintf("%s-reverseproxy-ingress", reverseProxyReleaseName.String())
	ingress, err := Clientset.NetworkingV1().Ingresses(namespace).Get(context.Background(), ingressName, metav1.GetOptions{})
	assert.NoError(t, err, "cannot retrieve reverse proxy ingress")
	assert.NotNil(t, ingress, "empty reverse proxy ingress found")
	ingressIP := ingress.Status.LoadBalancer.Ingress[0].IP
	assert.NotEmpty(t, ingressIP, "no ingress ip found")

	ingressURL := fmt.Sprintf("https://%s:443", ingressIP)
	stdout, _, err := RunCommand(exec.Command("curl", "-ivk", ingressURL))
	assert.NoError(t, err)
	assert.NotNil(t, string(stdout), "no curl output found")
	assert.Contains(t, string(stdout), "bolt_routing")
	assert.NotContains(t, string(stdout), "8443")

	return nil
}

func createGCPServiceAccount(k8sServiceAccountName string, namespace string, gcpServiceAccountName string) error {
	//mutex required since GCP does not allow you to create and add iam policies to service accounts concurrently
	log.Printf("k8sServiceAccountName = %s \n gcpServiceAccountName = %s", k8sServiceAccountName, gcpServiceAccountName)
	mutex.Lock()
	project := string(gcloud.CurrentProject())
	stdout, stderr, err := RunCommand(exec.Command("gcloud", "iam", "service-accounts", "create", gcpServiceAccountName,
		fmt.Sprintf("--project=%s", project)))
	if err != nil {
		return fmt.Errorf("error seen while trying to create gcp service account  %s \n Here's why err := %s \n stderr := %s", gcpServiceAccountName, err, string(stderr))
	}
	serviceAccountEmail := fmt.Sprintf("%s@%s.iam.gserviceaccount.com", gcpServiceAccountName, project)
	serviceAccountConfig := fmt.Sprintf("serviceAccount:%s", serviceAccountEmail)
	log.Printf("serviceAccountConfig %s serviceAccountEmail %s", serviceAccountConfig, serviceAccountEmail)
	log.Printf("GCP service account creation done \n Stdout = %s \n Stderr = %s", string(stdout), string(stderr))

	stdout, stderr, err = RunCommand(exec.Command("gcloud", "projects", "add-iam-policy-binding",
		project, "--member", serviceAccountConfig, "--role", "roles/storage.admin"))
	if err != nil {
		return fmt.Errorf("error seen while trying to add iam policy binding to gcp service account %s \n Here's why err := %s \n stderr := %s", gcpServiceAccountName, err, string(stderr))
	}
	log.Printf("Adding iam policy binding \n Stdout = %s \n Stderr = %s", string(stdout), string(stderr))

	stdout, stderr, err = RunCommand(exec.Command("gcloud", "projects", "add-iam-policy-binding",
		project, "--member", serviceAccountConfig, "--role", "roles/artifactregistry.repoAdmin"))
	if err != nil {
		return fmt.Errorf("error seen while trying to add artifact registry iam policy binding to gcp service account %s \n Here's why err := %s \n stderr := %s", gcpServiceAccountName, err, string(stderr))
	}
	log.Printf("Adding iam policy binding \n Stdout = %s \n Stderr = %s", string(stdout), string(stderr))

	stdout, stderr, err = RunCommand(exec.Command("gcloud", "iam", "service-accounts", "add-iam-policy-binding",
		serviceAccountEmail, "--role", "roles/iam.workloadIdentityUser",
		"--member", fmt.Sprintf("serviceAccount:%s.svc.id.goog[%s/%s]", string(gcloud.CurrentProject()), namespace, k8sServiceAccountName)))
	if err != nil {
		return fmt.Errorf("error seen while trying to add iam policy binding to k8s service account %s \n Here's why err := %s \n stderr := %s", k8sServiceAccountName, err, string(stderr))
	}
	log.Printf("Adding iam policy binding to service account \n Stdout = %s \n Stderr = %s", string(stdout), string(stderr))

	// sleep for few seconds to allow the settings be applied...immediate helm install after this step leads to failure
	time.Sleep(60 * time.Second)
	mutex.Unlock()
	return nil
}

// introduceInconsistency corrupts a neo4j database relationship store file to introduce inconsistency
func introduceInconsistency(t *testing.T, releaseName model.ReleaseName) error {

	err := createMoviesDataSet(t, releaseName)
	if !assert.NoError(t, err) {
		return err
	}

	err = stopDatabase(t, releaseName, "neo4j")
	if !assert.NoError(t, err) {
		return err
	}

	// corrupting the database
	// echo “” > /var/lib/neo4j/data/databases/neo4j/block.relationship.xd.db
	cmd := []string{
		"bash",
		"-c",
		"cp /var/lib/neo4j/data/databases/neo4j/block.relationship.xd.db /tmp/block.relationship.xd.db && echo '' > /var/lib/neo4j/data/databases/neo4j/block.relationship.xd.db",
	}
	stdout, stderr, err := ExecInPod(releaseName, cmd, "")
	if err != nil {
		return fmt.Errorf("error seen while executing command `echo \"\" > /var/lib/neo4j/data/databases/neo4j/block.relationship.xd.db' ,\n err :- %v", err)
	}
	if len(stderr) != 0 {
		return fmt.Errorf("found something in stderr while introducing inconsistency %v\n", stderr)
	}
	log.Printf("stdout of echo command for introducing inconsistency %s\n", stdout)

	err = startDatabase(t, releaseName, "neo4j")
	if !assert.NoError(t, err) {
		return err
	}

	return nil
}

// revertInconsistency replaces the corrupted file
func revertInconsistency(releaseName model.ReleaseName) error {

	cmd := []string{
		"bash",
		"-c",
		"mv /tmp/block.relationship.xd.db /var/lib/neo4j/data/databases/neo4j/block.relationship.xd.db",
	}
	stdout, stderr, err := ExecInPod(releaseName, cmd, "")
	if err != nil {
		return fmt.Errorf("error seen while executing command `mv /tmp/block.relationship.xd.db /var/lib/neo4j/data/databases/neo4j/block.relationship.xd.db' ,\n err :- %v", err)
	}
	if strings.TrimSpace(stderr) != "" {
		return fmt.Errorf("stderr is not empty while reverting inconsistency%v\n", stderr)
	}
	log.Printf("stdout while reverting inconsistency %s \n", stdout)
	return nil
}

func deleteGCPServiceAccount(gcpServiceAccountName string) error {
	log.Printf("Deleting GCP Service Account %s", gcpServiceAccountName)
	_, _, err := RunCommand(exec.Command("gcloud", "iam", "service-accounts", "delete", fmt.Sprintf("%s@%s.iam.gserviceaccount.com", gcpServiceAccountName, string(gcloud.CurrentProject()))))
	if err != nil {
		return fmt.Errorf("error seen while trying to add iam policy binding \n Here's why err := %s ", err)
	}
	return nil
}

func createAWSBucket(accessKey string, secretKey string, region string, bucketName string) error {
	// Create a custom credentials provider
	credProvider := credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")

	// Load the AWS configuration with the custom credentials provider
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(region),
		config.WithCredentialsProvider(credProvider),
	)
	if err != nil {
		log.Printf("Error loading AWS config: %v\n", err)
		return err
	}

	// Create an S3 client
	client := s3.NewFromConfig(cfg)

	// Create the bucket
	_, err = client.CreateBucket(context.TODO(), &s3.CreateBucketInput{
		Bucket: aws.String(bucketName),
	})

	if err != nil {
		log.Printf("Error creating bucket: %v\n", err)
		return err
	}
	log.Printf("AWS bucket %s created", bucketName)
	return nil
}

func deleteAWSBucket(accessKey string, secretKey string, region string, bucketName string) error {
	credProvider := credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")

	// Load the AWS configuration with the custom credentials provider
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(region),
		config.WithCredentialsProvider(credProvider),
	)
	if err != nil {
		log.Printf("Error loading AWS config: %v\n", err)
		return err
	}

	// Create an S3 client
	client := s3.NewFromConfig(cfg)

	paginator := s3.NewListObjectsV2Paginator(client, &s3.ListObjectsV2Input{
		Bucket: &bucketName,
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if err != nil {
			return fmt.Errorf("failed to list objects: %v", err)
		}

		var objectIds []types.ObjectIdentifier
		for _, object := range page.Contents {
			objectIds = append(objectIds, types.ObjectIdentifier{Key: object.Key})
		}

		_, err = client.DeleteObjects(context.TODO(), &s3.DeleteObjectsInput{
			Bucket: &bucketName,
			Delete: &types.Delete{Objects: objectIds},
		})
		if err != nil {
			return fmt.Errorf("failed to delete objects: %v", err)
		}
	}

	// Delete the bucket
	_, err = client.DeleteBucket(context.TODO(), &s3.DeleteBucketInput{
		Bucket: aws.String(bucketName),
	})

	if err != nil {
		log.Printf("Error deleting bucket: %v\n", err)
		return err
	}
	log.Printf("AWS bucket %s deleted", bucketName)
	return nil
}
