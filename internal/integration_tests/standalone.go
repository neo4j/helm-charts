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
	"io"
	"log"
	"math"
	"math/big"
	"os"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	. "github.com/neo4j/helm-charts/internal/helpers"
	"github.com/neo4j/helm-charts/internal/integration_tests/gcloud"
	"github.com/neo4j/helm-charts/internal/model"
	"github.com/neo4j/helm-charts/internal/testutil/poll"
	"github.com/neo4j/helm-charts/internal/testutil/timeouts"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
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
	namespace := string(releaseName.Namespace())

	// Try to delete the namespace if it exists
	_ = run(t, "kubectl", "delete", "ns", namespace, "--ignore-not-found=true")

	// Wait for the namespace to be fully deleted
	maxRetries := 30
	for i := 0; i < maxRetries; i++ {
		err := run(t, "kubectl", "get", "ns", namespace)
		if err != nil {
			// Namespace doesn't exist, we can proceed
			break
		}
		t.Logf("Waiting for namespace %s to be deleted... (%d/%d)", namespace, i+1, maxRetries)
		time.Sleep(5 * time.Second)
		if i == maxRetries-1 {
			return func() error {
				return runAll(t, "kubectl", kCleanupCommands(releaseName.Namespace()), false)
			}, fmt.Errorf("timed out waiting for namespace %s to be deleted", namespace)
		}
	}

	err := run(t, "kubectl", "create", "ns", namespace)
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
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	out, err := exec.CommandContext(ctx, command, args...).CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		t.Logf("Command timed out after 10 minutes: %s %s", command, args)
		return fmt.Errorf("command timed out after 10 minutes: %s %v", command, args)
	}
	if out != nil {
		t.Logf("output: %s\n", out)
	}
	return err
}

func kubectlLogs(t *testing.T, podName string, namespace string) (string, error) {
	return poll.UntilValue(context.Background(), t, poll.Opts{
		Interval:      10 * time.Second,
		Timeout:       25 * time.Second,
		Description:   fmt.Sprintf("kubectl logs for pod %s in %s", podName, namespace),
		RetryableErrs: func(error) bool { return true },
	}, func(context.Context) (string, bool, error) {
		out, err := exec.Command("kubectl", "logs", podName, "--namespace", namespace).CombinedOutput()
		if err != nil {
			return "", false, err
		}
		return string(out), true, nil
	})
}

func waitForPodLogsMatching(t *testing.T, namespace string, podNamePart string, timeout time.Duration, ready func(string) error) (string, error) {
	return poll.UntilValue(context.Background(), t, poll.Opts{
		Interval:      10 * time.Second,
		Timeout:       timeout,
		Description:   fmt.Sprintf("logs for pod containing %s in %s", podNamePart, namespace),
		RetryableErrs: func(error) bool { return true },
	}, func(ctx context.Context) (string, bool, error) {
		pods, err := Clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			return "", false, err
		}

		for _, pod := range pods.Items {
			if !strings.Contains(pod.Name, podNamePart) {
				continue
			}
			out, err := exec.Command("kubectl", "logs", pod.Name, "--namespace", namespace).CombinedOutput()
			if err != nil {
				return "", false, err
			}
			logOutput := string(out)
			if err := ready(logOutput); err != nil {
				return logOutput, false, err
			}
			return logOutput, true, nil
		}

		return "", false, fmt.Errorf("no pod containing %s found", podNamePart)
	})
}

func waitForPodsTerminated(t *testing.T, namespace string, timeout time.Duration) {
	err := poll.Until(context.Background(), t, poll.Opts{
		Interval:      5 * time.Second,
		Timeout:       timeout,
		Description:   fmt.Sprintf("all pods in namespace %s to terminate", namespace),
		RetryableErrs: func(error) bool { return true },
	}, func(ctx context.Context) (bool, error) {
		pods, err := Clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			return false, err
		}
		return len(pods.Items) == 0, nil
	})
	if err != nil {
		t.Logf("%v; force-deleting remaining pods", err)
		_ = run(t, "kubectl", "delete", "pod", "--all", "--namespace", namespace, "--force", "--grace-period=0", "--ignore-not-found")
	}
}

func AsCloseable(closeables []Closeable) Closeable {
	return func() error {
		var combinedErrors error
		for _, closeable := range closeables {
			innerErr := closeable()
			if innerErr != nil {
				combinedErrors = CombineErrors(combinedErrors, innerErr)
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
	err := waitForClusterConnection(t)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to cluster: %v", err)
	}

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
	if err != nil {
		return AsCloseable(closeables), err
	}
	err = run(t, "kubectl", "--namespace", string(releaseName.Namespace()), "wait", "--for=condition=Ready", "pod/"+releaseName.PodName(), timeouts.KubectlPodReady())
	return AsCloseable(closeables), err
}

func TestBackupLogStreamingIntegration(t *testing.T, releaseName model.ReleaseName) error {
	if model.Neo4jEdition == "community" {
		t.Skip()
		return nil
	}

	backupReleaseName := model.NewReleaseName("standalone-backup-logs-" + TestRunIdentifier)
	namespace := string(releaseName.Namespace())

	t.Cleanup(func() {
		_ = runAll(t, "helm", [][]string{
			{"uninstall", backupReleaseName.String(), "--wait", "--timeout", "3m", "--namespace", namespace},
		}, false)
	})

	// Install backup chart without cloud provider to use local volume
	helmClient := model.NewHelmClient(model.DefaultNeo4jBackupChartName)
	helmValues := model.DefaultNeo4jBackupValues
	helmValues.Backup = model.Backup{
		DatabaseAdminServiceName: fmt.Sprintf("%s-admin", releaseName.String()),
		DatabaseNamespace:        namespace,
		Database:                 "neo4j,system",
		CloudProvider:            "",
		Verbose:                  true,
		Type:                     "FULL",
		KeepBackupFiles:          true,
	}
	helmValues.Neo4J.JobSchedule = "* * * * *"

	_, err := helmClient.Install(t, backupReleaseName.String(), namespace, helmValues)
	if err != nil {
		return fmt.Errorf("helm install failed: %v", err)
	}

	// Wait for backup job to complete
	time.Sleep(2 * time.Minute)

	// Get all pods in the namespace
	pods, err := Clientset.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error while retrieving pod list during backup operation: %v", err)
	}

	var found bool
	var logOutput string
	for _, pod := range pods.Items {
		if strings.Contains(pod.Name, "standalone-backup-logs") {
			found = true
			out, err := exec.Command("kubectl", "logs", pod.Name, "--namespace", namespace).CombinedOutput()
			if err != nil {
				return fmt.Errorf("error while getting backup pod logs: %v", err)
			}
			logOutput = string(out)
			break
		}
	}
	if !found {
		return fmt.Errorf("no backup pod found")
	}

	// Verify log content
	expectedLogEntries := []string{
		"Backup Completed",
	}

	for _, expectedLog := range expectedLogEntries {
		if !strings.Contains(logOutput, expectedLog) {
			return fmt.Errorf("expected log entry '%s' not found in logs", expectedLog)
		}
	}

	return nil
}

func k8sTests(name model.ReleaseName, chart model.Neo4jHelmChartBuilder) ([]SubTest, error) {
	expectedConfiguration, err := (&model.Neo4jConfiguration{}).PopulateFromFile(Neo4jConfFile)
	if err != nil {
		return nil, err
	}
	log.Printf("%v", expectedConfiguration)
	return []SubTest{
		{name: "Check Neo4j Logs For Any Errors", test: func(t *testing.T) {
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
			assert.NoError(t, InstallNeo4jBackupGCPHelmChartWithWorkloadIdentity(t, name), "Backup to GCP with workload identity should succeed")
		}},
		{name: "Install Backup Helm Chart For AWS", test: func(t *testing.T) {
			assert.NoError(t, InstallNeo4jBackupAWSHelmChart(t, name), "Backup to AWS should succeed")
		}},
		{name: "Install Backup Helm Chart For Azure", test: func(t *testing.T) {
			assert.NoError(t, InstallNeo4jBackupAzureHelmChart(t, name), "Backup to Azure should succeed")
		}},
		{name: "Install Backup Helm Chart For GCP", test: func(t *testing.T) {
			assert.NoError(t, InstallNeo4jBackupGCPHelmChart(t, name), "Backup to GCP should succeed")
		}},
		{name: "Install Reverse Proxy Helm Chart", test: func(t *testing.T) {
			assert.NoError(t, InstallReverseProxyHelmChart(t, name), "Reverse Proxy installation with ingress should succeed")
		}},
		{name: "Install Backup With File Cleanup", test: func(t *testing.T) {
			assert.NoError(t, InstallNeo4jBackupWithFileCleanup(t, name), "Backup with file cleanup should succeed")
		}},
		{name: "Check Backup Log Streaming", test: func(t *testing.T) {
			assert.NoError(t, TestBackupLogStreamingIntegration(t, name), "Backup log streaming should work correctly")
		}},
	}, nil
}

func waitForServiceAccountCreation(projectID, serviceAccountEmail string, maxRetries int) error {
	for i := 0; i < maxRetries; i++ {
		cmd := exec.Command("gcloud", "iam", "service-accounts", "describe",
			serviceAccountEmail,
			"--project", projectID)

		if err := cmd.Run(); err == nil {
			return nil
		}

		time.Sleep(time.Duration(math.Pow(2, float64(i))) * time.Second)
	}
	return fmt.Errorf("service account %s was not created after %d retries",
		serviceAccountEmail, maxRetries)
}

func InstallNeo4jBackupAWSHelmChart(t *testing.T, standaloneReleaseName model.ReleaseName) error {
	if model.Neo4jEdition == "community" {
		t.Skip()
		return nil
	}
	backupReleaseName := model.NewReleaseName("standalone-backup-aws-" + TestRunIdentifier)
	backupBucketName := fmt.Sprintf("helm-charts-%s", TestRunIdentifier)
	namespace := string(standaloneReleaseName.Namespace())

	t.Logf("Using namespace: %s for AWS backup test", namespace)

	t.Cleanup(func() {
		_ = runAll(t, "kubectl", [][]string{
			{"delete", "secret", "awscred", "--namespace", namespace, "--ignore-not-found"},
		}, false)
		_ = runAll(t, "helm", [][]string{
			{"uninstall", backupReleaseName.String(), "--wait", "--timeout", "3m", "--namespace", namespace},
		}, false)
		_ = deleteAWSBucket(os.Getenv("AWS_ACCESS_KEY_ID"), os.Getenv("AWS_SECRET_ACCESS_KEY"), "us-east-1", backupBucketName)
	})

	_, err := Clientset.CoreV1().Secrets(namespace).Get(context.TODO(), "awscred", metav1.GetOptions{})
	if err == nil {
		t.Logf("Found existing secret 'awscred' in namespace %s, deleting it", namespace)
		err = Clientset.CoreV1().Secrets(namespace).Delete(context.TODO(), "awscred", metav1.DeleteOptions{})
		if err != nil {
			return fmt.Errorf("failed to delete existing AWS credentials secret: %v", err)
		}
	}

	secretKey := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "awscred",
			Namespace: namespace,
		},
		Data: map[string][]byte{
			"credentials": []byte(fmt.Sprintf("[default]\naws_access_key_id=%s\naws_secret_access_key=%s",
				os.Getenv("AWS_ACCESS_KEY_ID"),
				os.Getenv("AWS_SECRET_ACCESS_KEY"))),
		},
		Type: "Opaque",
	}

	_, err = Clientset.CoreV1().Secrets(namespace).Create(context.TODO(), secretKey, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create AWS credentials secret: %v", err)
	}

	_, err = Clientset.CoreV1().Secrets(namespace).Get(context.TODO(), "awscred", metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to verify AWS credentials secret exists: %v", err)
	}

	t.Logf("Successfully created and verified secret 'awscred' in namespace %s", namespace)

	err = createAWSBucket(os.Getenv("AWS_ACCESS_KEY_ID"), os.Getenv("AWS_SECRET_ACCESS_KEY"), "us-east-1", backupBucketName)
	if err != nil {
		return fmt.Errorf("failed to create AWS bucket: %v", err)
	}

	helmClient := model.NewHelmClient(model.DefaultNeo4jBackupChartName)
	helmValues := model.DefaultNeo4jBackupValues
	helmValues.Backup = model.Backup{
		BucketName:               backupBucketName,
		DatabaseAdminServiceName: fmt.Sprintf("%s-admin", standaloneReleaseName.String()),
		DatabaseNamespace:        namespace,
		Database:                 "neo4j,system",
		CloudProvider:            "aws",
		SecretName:               "awscred",
		SecretKeyName:            "credentials",
		Verbose:                  true,
		KeepBackupFiles:          true,
		Type:                     "FULL",
		S3ForcePathStyle:         true,
		S3Region:                 "us-east-1",
		S3SignatureVersion:       "4",
	}
	helmValues.ConsistencyCheck.Database = "neo4j"

	t.Logf("Installing helm chart in namespace %s with secret 'awscred'", namespace)
	_, err = helmClient.Install(t, backupReleaseName.String(), namespace, helmValues)
	if err != nil {
		secret, getErr := Clientset.CoreV1().Secrets(namespace).Get(context.TODO(), "awscred", metav1.GetOptions{})
		if getErr != nil {
			t.Logf("Debug: Failed to get secret after helm error: %v", getErr)
		} else {
			t.Logf("Debug: Secret exists with keys: %v", secret.Data)
		}
		return fmt.Errorf("helm install failed: %v", err)
	}

	return nil
}

func InstallNeo4jBackupAzureHelmChart(t *testing.T, standaloneReleaseName model.ReleaseName) error {
	if model.Neo4jEdition == "community" {
		t.Skip()
		return nil
	}
	backupReleaseName := model.NewReleaseName("standalone-backup-azure-" + TestRunIdentifier)
	namespace := string(standaloneReleaseName.Namespace())

	t.Log("Starting Azure backup test")

	t.Cleanup(func() {
		t.Log("Running cleanup for Azure backup test")
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

	t.Logf("Installing Azure backup helm chart with values: %+v", helmValues)
	_, err := helmClient.Install(t, backupReleaseName.String(), namespace, helmValues)
	if err != nil {
		t.Logf("Failed to install Azure backup helm chart: %v", err)
		return err
	}

	t.Log("Waiting for Azure backup job to complete")
	time.Sleep(2 * time.Minute)

	cronjob, err := Clientset.BatchV1().CronJobs(namespace).Get(context.Background(), backupReleaseName.String(), metav1.GetOptions{})
	if err != nil {
		t.Logf("Failed to get Azure backup cronjob: %v", err)
		return fmt.Errorf("cannot retrieve azure backup cronjob: %v", err)
	}
	if cronjob.Spec.Schedule != helmValues.Neo4J.JobSchedule {
		t.Logf("Azure cronjob schedule mismatch. Got %s, want %s", cronjob.Spec.Schedule, helmValues.Neo4J.JobSchedule)
		return fmt.Errorf("azure cronjob schedule %s not matching with the schedule defined in values.yaml %s", cronjob.Spec.Schedule, helmValues.Neo4J.JobSchedule)
	}

	t.Log("Getting Azure backup pod logs")
	pods, err := Clientset.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		t.Logf("Failed to list pods: %v", err)
		return fmt.Errorf("error while retrieving pod list during azure backup operation: %v", err)
	}

	var found bool
	for _, pod := range pods.Items {
		if strings.Contains(pod.Name, "standalone-backup-azure") {
			found = true
			t.Logf("Found Azure backup pod: %s", pod.Name)
			t.Logf("Pod status: %s", pod.Status.Phase)

			out, err := exec.Command("kubectl", "logs", pod.Name, "--namespace", namespace).CombinedOutput()
			if err != nil {
				t.Logf("Failed to get Azure backup pod logs: %v", err)
				return fmt.Errorf("error while getting azure backup pod logs: %v", err)
			}
			if out == nil {
				t.Log("Azure backup pod logs are empty")
				return fmt.Errorf("azure backup logs cannot be retrieved")
			}

			logOutput := string(out)
			t.Logf("Azure backup pod logs:\n%s", logOutput)

			// Check for connectivity and initialization logs
			requiredLogs := []string{
				"Connectivity established with Database",
				"Printing backup flags",
				"--include-metadata=all",
				"--type=FULL",
				"neo4j system",
				"Backup command completed",
				"Backup Completed",
				"uploaded to azure container",
			}

			for _, requiredLog := range requiredLogs {
				if !strings.Contains(logOutput, requiredLog) {
					t.Logf("Required log entry not found in Azure backup: %s", requiredLog)
					return fmt.Errorf("required log entry not found in Azure backup: %s", requiredLog)
				}
			}
			break
		}
	}

	if !found {
		t.Log("No Azure backup pod found")
		return fmt.Errorf("no azure backup pod found")
	}

	t.Log("Azure backup test completed successfully")
	return nil
}

func InstallNeo4jBackupGCPHelmChart(t *testing.T, standaloneReleaseName model.ReleaseName) error {
	if model.Neo4jEdition == "community" {
		t.Skip()
		return nil
	}
	backupReleaseName := model.NewReleaseName("standalone-backup-gcp-" + TestRunIdentifier)
	namespace := string(standaloneReleaseName.Namespace())

	t.Log("Starting GCP backup test")

	t.Cleanup(func() {
		t.Log("Running cleanup for GCP backup test")
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

	t.Logf("Installing GCP backup helm chart with values: %+v", helmValues)
	_, err := helmClient.Install(t, backupReleaseName.String(), namespace, helmValues)
	if err != nil {
		t.Logf("Failed to install GCP backup helm chart: %v", err)
		return err
	}

	t.Log("Waiting for GCP backup job to complete")
	time.Sleep(2 * time.Minute)

	cronjob, err := Clientset.BatchV1().CronJobs(namespace).Get(context.Background(), backupReleaseName.String(), metav1.GetOptions{})
	if err != nil {
		t.Logf("Failed to get GCP backup cronjob: %v", err)
		return fmt.Errorf("cannot retrieve gcp backup cronjob: %v", err)
	}
	if cronjob.Spec.Schedule != helmValues.Neo4J.JobSchedule {
		t.Logf("GCP cronjob schedule mismatch. Got %s, want %s", cronjob.Spec.Schedule, helmValues.Neo4J.JobSchedule)
		return fmt.Errorf("gcp cronjob schedule %s not matching with the schedule defined in values.yaml %s", cronjob.Spec.Schedule, helmValues.Neo4J.JobSchedule)
	}

	t.Log("Getting GCP backup pod logs")
	pods, err := Clientset.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		t.Logf("Failed to list pods: %v", err)
		return fmt.Errorf("error while retrieving pod list during gcp backup operation: %v", err)
	}

	var found bool
	for _, pod := range pods.Items {
		if strings.Contains(pod.Name, "standalone-backup-gcp") {
			found = true
			t.Logf("Found GCP backup pod: %s", pod.Name)
			t.Logf("Pod status: %s", pod.Status.Phase)

			out, err := exec.Command("kubectl", "logs", pod.Name, "--namespace", namespace).CombinedOutput()
			if err != nil {
				t.Logf("Failed to get GCP backup pod logs: %v", err)
				return fmt.Errorf("error while getting gcp backup pod logs: %v", err)
			}
			if out == nil {
				t.Log("GCP backup pod logs are empty")
				return fmt.Errorf("gcp backup logs cannot be retrieved")
			}

			logOutput := string(out)
			t.Logf("GCP backup pod logs:\n%s", logOutput)

			// Check for connectivity and initialization logs
			requiredLogs := []string{
				"Connectivity established with Database",
				"Connectivity with bucket",
				"Printing backup flags",
				"--include-metadata=all",
				"--type=FULL",
				"neo4j",
				"Backup command completed",
				"Backup Completed",
			}

			for _, requiredLog := range requiredLogs {
				if !strings.Contains(logOutput, requiredLog) {
					t.Logf("Required log entry not found in GCP backup: %s", requiredLog)
					return fmt.Errorf("required log entry not found in GCP backup: %s", requiredLog)
				}
			}
			break
		}
	}

	if !found {
		t.Log("No GCP backup pod found")
		return fmt.Errorf("no gcp backup pod found")
	}

	t.Log("GCP backup test completed successfully")
	return nil
}

func InstallNeo4jBackupGCPHelmChartWithInconsistencies(t *testing.T, standaloneReleaseName model.ReleaseName) error {
	if model.Neo4jEdition == "community" {
		t.Skip()
		return nil
	}

	t.Log("Starting backup test with inconsistencies")
	err := introduceInconsistency(t, standaloneReleaseName)
	if err != nil {
		t.Logf("Failed to introduce inconsistency: %v", err)
		return err
	}

	backupReleaseName := model.NewReleaseName("standalone-backup-gcp-incon-" + TestRunIdentifier)
	namespace := string(standaloneReleaseName.Namespace())

	t.Cleanup(func() {
		t.Log("Running cleanup for backup test")
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

	t.Logf("Installing backup helm chart with values: %+v", helmValues)
	_, err = helmClient.Install(t, backupReleaseName.String(), namespace, helmValues)
	if err != nil {
		t.Logf("Failed to install backup helm chart: %v", err)
		return err
	}

	t.Log("Waiting for backup job to complete")
	time.Sleep(2 * time.Minute)

	cronjob, err := Clientset.BatchV1().CronJobs(namespace).Get(context.Background(), backupReleaseName.String(), metav1.GetOptions{})
	if err != nil {
		t.Logf("Failed to get cronjob: %v", err)
		return fmt.Errorf("cannot retrieve gcp backup cronjob: %v", err)
	}
	if cronjob.Spec.Schedule != helmValues.Neo4J.JobSchedule {
		t.Logf("Cronjob schedule mismatch. Got %s, want %s", cronjob.Spec.Schedule, helmValues.Neo4J.JobSchedule)
		return fmt.Errorf("gcp cronjob schedule %s not matching with the schedule defined in values.yaml %s", cronjob.Spec.Schedule, helmValues.Neo4J.JobSchedule)
	}

	t.Log("Getting backup pod logs")
	pods, err := Clientset.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		t.Logf("Failed to list pods: %v", err)
		return fmt.Errorf("error while retrieving pod list during gcp backup operation: %v", err)
	}

	var found bool
	for _, pod := range pods.Items {
		if strings.Contains(pod.Name, "standalone-backup-gcp-incon-") {
			found = true
			t.Logf("Found backup pod: %s", pod.Name)
			t.Logf("Pod status: %s", pod.Status.Phase)

			out, err := exec.Command("kubectl", "logs", pod.Name, "--namespace", namespace).CombinedOutput()
			if err != nil {
				t.Logf("Failed to get pod logs: %v", err)
				return fmt.Errorf("error while getting gcp backup pod logs: %v", err)
			}
			if out == nil {
				t.Log("Pod logs are empty")
				return fmt.Errorf("gcp backup logs cannot be retrieved")
			}

			logOutput := string(out)
			t.Logf("Pod logs:\n%s", logOutput)

			// Check for connectivity and initialization logs
			requiredLogs := []string{
				"Connectivity established with Database",
				"Printing backup flags",
				"--include-metadata=all",
				"--type=FULL",
				"neo4j system",
				"Printing consistency check flags",
				"--check-indexes=true",
				"--check-graph=true",
				"--check-counts=true",
				"--check-property-owners=true",
				"Backup command completed",
				"Backup Completed",
			}

			for _, requiredLog := range requiredLogs {
				if !strings.Contains(logOutput, requiredLog) {
					t.Logf("Required log entry not found: %s", requiredLog)
					return fmt.Errorf("required log entry not found: %s", requiredLog)
				}
			}
			break
		}
	}

	if !found {
		t.Log("No backup pod found")
		return fmt.Errorf("no gcp backup pod found")
	}

	t.Log("Reverting inconsistency")
	err = revertInconsistency(standaloneReleaseName)
	if err != nil {
		t.Logf("Failed to revert inconsistency: %v", err)
		return fmt.Errorf("error seen while reverting inconsistency: %v", err)
	}

	t.Log("Backup test completed successfully")
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

	cronjob, err := Clientset.BatchV1().CronJobs(namespace).Get(context.Background(), backupReleaseName.String(), metav1.GetOptions{})
	assert.NoError(t, err, "cannot retrieve gcp backup cronjob")
	assert.Equal(t, cronjob.Spec.Schedule, helmValues.Neo4J.JobSchedule, fmt.Sprintf("gcp cronjob schedule %s not matching with the schedule defined in values.yaml %s", cronjob.Spec.Schedule, helmValues.Neo4J.JobSchedule))

	requiredLogs := []string{
		"Connectivity established with Database",
		"Credential Path is /credentials/",
		"Connectivity with bucket",
		"Printing backup flags",
		"--include-metadata=all",
		"--type=FULL",
		"neo4j system",
	}
	_, err = waitForPodLogsMatching(t, namespace, backupReleaseName.String(), 5*time.Minute, func(logOutput string) error {
		for _, requiredLog := range requiredLogs {
			if !strings.Contains(logOutput, requiredLog) {
				return fmt.Errorf("required log entry not found: %s", requiredLog)
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("gcp workload backup logs did not reach expected content: %v", err)
	}

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
	if err != nil {
		return err
	}

	_, err = helmClient.Install(t, reverseProxyReleaseName.String(), namespace, helmValues)
	if err != nil {
		return err
	}

	reverseProxyDepName := fmt.Sprintf("%s-reverseproxy-dep", reverseProxyReleaseName.String())
	err = run(t, "kubectl", "--namespace", namespace, "rollout", "status", "deployment/"+reverseProxyDepName, "--timeout=3m")
	if err != nil {
		return err
	}
	deployment, err := Clientset.AppsV1().Deployments(namespace).Get(context.Background(), reverseProxyDepName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("cannot retrieve reverse proxy deployment: %v", err)
	}
	assert.NotNil(t, deployment, "no reverse proxy deployment found")

	pod, err := poll.UntilValue(context.Background(), t, poll.Opts{
		Interval:      5 * time.Second,
		Timeout:       3 * time.Minute,
		Description:   fmt.Sprintf("reverse proxy pod for %s to be Ready", reverseProxyReleaseName.String()),
		RetryableErrs: func(error) bool { return true },
	}, func(ctx context.Context) (v1.Pod, bool, error) {
		pods, err := Clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
			LabelSelector: fmt.Sprintf("name=%s-reverseproxy", reverseProxyReleaseName.String()),
		})
		if err != nil {
			return v1.Pod{}, false, err
		}
		if len(pods.Items) != 1 {
			return v1.Pod{}, false, fmt.Errorf("expected 1 reverse proxy pod, got %d", len(pods.Items))
		}
		pod := pods.Items[0]
		for _, condition := range pod.Status.Conditions {
			if condition.Type == v1.PodReady && condition.Status == v1.ConditionTrue {
				return pod, true, nil
			}
		}
		return pod, false, fmt.Errorf("reverse proxy pod %s is not Ready", pod.Name)
	})
	if err != nil {
		return err
	}

	cmd := []string{"ls", "-lst", "/reverse-proxy"}
	stdoutCmd, _, err := ExecInPod(standaloneReleaseName, cmd, pod.Name)
	if err != nil {
		return fmt.Errorf("cannot exec in reverse proxy pod: %v", err)
	}
	assert.NotContains(t, stdoutCmd, "root")
	assert.Contains(t, stdoutCmd, "neo4j")

	ingressName := fmt.Sprintf("%s-reverseproxy-ingress", reverseProxyReleaseName.String())
	ingressIP, err := poll.UntilValue(context.Background(), t, poll.Opts{
		Interval:      10 * time.Second,
		Timeout:       5 * time.Minute,
		Description:   fmt.Sprintf("ingress IP for %s", ingressName),
		RetryableErrs: func(error) bool { return true },
	}, func(ctx context.Context) (string, bool, error) {
		ingress, err := Clientset.NetworkingV1().Ingresses(namespace).Get(ctx, ingressName, metav1.GetOptions{})
		if err != nil {
			return "", false, err
		}
		if len(ingress.Status.LoadBalancer.Ingress) == 0 || ingress.Status.LoadBalancer.Ingress[0].IP == "" {
			return "", false, fmt.Errorf("no ingress IP assigned")
		}
		return ingress.Status.LoadBalancer.Ingress[0].IP, true, nil
	})
	if err != nil {
		return err
	}

	ingressURL := fmt.Sprintf("https://%s:443", ingressIP)
	stdout, err := poll.UntilValue(context.Background(), t, poll.Opts{
		Interval:      10 * time.Second,
		Timeout:       3 * time.Minute,
		Description:   fmt.Sprintf("reverse proxy endpoint %s to return Neo4j routing metadata", ingressURL),
		RetryableErrs: func(error) bool { return true },
	}, func(context.Context) ([]byte, bool, error) {
		stdout, _, err := RunCommand(exec.Command("wget", "-qO-", "--no-check-certificate", ingressURL))
		if err != nil {
			return nil, false, err
		}
		if !strings.Contains(string(stdout), "bolt_routing") {
			return stdout, false, fmt.Errorf("reverse proxy response missing bolt_routing")
		}
		return stdout, true, nil
	})
	if err != nil {
		return err
	}
	assert.NotNil(t, string(stdout), "no wget output found")
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
	time.Sleep(10 * time.Second)

	if err := waitForServiceAccountCreation(project, serviceAccountEmail, 5); err != nil {
		return fmt.Errorf("failed waiting for service account creation: %v", err)
	}

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
	// echo "" > /var/lib/neo4j/data/databases/neo4j/block.relationship.xd.db
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

func InstallNeo4jBackupWithFileCleanup(t *testing.T, standaloneReleaseName model.ReleaseName) error {
	backupReleaseName := model.NewReleaseName(fmt.Sprintf("backup-%s", standaloneReleaseName))
	namespace := string(backupReleaseName.Namespace())

	t.Cleanup(func() {
		_ = runAll(t, "helm", [][]string{
			{"uninstall", backupReleaseName.String(), "--wait", "--timeout", "3m", "--namespace", namespace},
			{"delete", "namespace", namespace},
		}, false)
	})

	if _, err := createNamespace(t, backupReleaseName); err != nil {
		return err
	}

	helmClient := model.NewHelmClient(model.DefaultNeo4jBackupChartName)
	helmValues := model.DefaultNeo4jBackupValues
	helmValues.Backup.SecretName = "backup-secret"
	helmValues.Backup.SecretKeyName = "credentials"
	helmValues.Backup.CloudProvider = "gcp"
	helmValues.Backup.BucketName = "backup-bucket"
	helmValues.Backup.DatabaseAdminServiceName = fmt.Sprintf("%s-admin", standaloneReleaseName)
	helmValues.Backup.Database = "neo4j,system"
	helmValues.Backup.KeepBackupFiles = false

	secretKey := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "backup-secret",
			Namespace: namespace,
		},
		Data: map[string][]byte{
			"credentials": []byte("demo-credentials"),
		},
		Type: "Opaque",
	}

	_, err := Clientset.CoreV1().Secrets(namespace).Create(context.TODO(), secretKey, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	_, err = helmClient.Install(t, backupReleaseName.String(), namespace, helmValues)
	return err
}

func TestProbeConfigurations(t *testing.T) {
	releaseName := model.NewReleaseName("neo4j-probes")
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
			closeable, err := installNeo4j(t, releaseName, chart)
			assert.NoError(t, err)
			t.Cleanup(func() { _ = closeable() })

			err = run(t, "kubectl", "--namespace", string(releaseName.Namespace()),
				"wait", "--for=condition=ready", "pod", releaseName.PodName(),
				"--timeout=300s")
			assert.NoError(t, err)

			err = CheckProbes(t, releaseName)
			assert.NoError(t, err)
		})
	}
}
