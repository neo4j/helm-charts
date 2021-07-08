package internal

import (
	"bufio"
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"github.com/hashicorp/go-multierror"
	"io"
	"io/ioutil"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"log"
	"math/big"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

var (
	Clientset *kubernetes.Clientset
	Config    *restclient.Config
)

const storageSize = "10Gi"
const cpuRequests = "50m"
const memoryRequests = "900Mi"
const cpuLimits = "1500m"
const memoryLimits = "900Mi"

// This changes the working directory to the parent directory if the current working directory doesn't contain a directory called "yaml"
func init() {
	_, filename, _, _ := runtime.Caller(0)
	currentDir := path.Dir(filename)
	files, err := ioutil.ReadDir(currentDir)
	CheckError(err)
	for _, file := range files {
		if file.Name() == "yaml" {
			return
		}
	}
	dir := path.Join(currentDir, "..")
	err = os.Chdir(dir)
	CheckError(err)

	os.Setenv("KUBECONFIG", path.Join(dir, ".kube/config"))

	// gets kubeconfig from env variable
	Config, err = clientcmd.BuildConfigFromFlags("", os.Getenv("KUBECONFIG"))
	CheckError(err)
	Clientset, err = kubernetes.NewForConfig(Config)
	CheckError(err)
}

func CheckError(err error) {
	if err != nil {
		log.Panic(err)
	}
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

var defaultPassword = fmt.Sprintf("a%da", RandomIntBetween(100000, 999999999))

func minHelmCommand(helmCommand string, releaseName *ReleaseName) []string {
	return []string{helmCommand, string(*releaseName), "./neo4j-standalone", "--namespace", string(releaseName.namespace())}
}

func baseHelmCommand(helmCommand string, releaseName *ReleaseName, extraHelmArguments ...string) []string {

	var helmArgs = minHelmCommand(helmCommand, releaseName)
	helmArgs = append(helmArgs,
		"--create-namespace",
		"--set", "volumes.data.mode=selector",
		"--set", "volumes.data.selector.requests.storage="+storageSize,
		"--set", "neo4j.password="+defaultPassword,
		"--set", "neo4j.resources.requests.cpu="+cpuRequests,
		"--set", "neo4j.resources.requests.memory="+memoryRequests,
		"--set", "neo4j.resources.limits.cpu="+cpuLimits,
		"--set", "neo4j.resources.limits.memory="+memoryLimits,
		"--set", "ssl.bolt.privateKey.secretName=bolt-key", "--set", "ssl.bolt.publicCertificate.secretName=bolt-cert",
		"--set", "ssl.bolt.trustedCerts.sources[0].secret.name=bolt-cert",
		"--set", "ssl.https.privateKey.secretName=https-key", "--set", "ssl.https.publicCertificate.secretName=https-cert",
		"--set", "ssl.https.trustedCerts.sources[0].secret.name=https-cert",
	)

	if value, found := os.LookupEnv("NEO4J_DOCKER_IMG"); found {
		helmArgs = append(helmArgs, "--set", "image.customImage="+value)
	}

	if value, found := os.LookupEnv("NEO4J_EDITION"); found {
		helmArgs = append(helmArgs, "--set", "neo4j.edition="+value)
		if strings.EqualFold(value, "enterprise") {
			helmArgs = append(helmArgs, "--set", "neo4j.acceptLicenseAgreement=yes")
		}
	}

	if extraHelmArguments != nil && len(extraHelmArguments) > 0 {
		helmArgs = append(helmArgs, extraHelmArguments...)
	}

	return helmArgs
}

func helmInstallCommands(releaseName *ReleaseName, diskName PersistentDiskName) [][]string {
	return [][]string{
		{"install", string(*releaseName + "-pv"), "./neo4j-gcloud-pv", "--wait", "--timeout", "120s",
			"--set", "neo4j.name=" + string(*releaseName),
			"--set", "data.capacity.storage=" + storageSize,
			"--set", "data.gcePersistentDisk=" + string(diskName)},
		baseHelmCommand("install", releaseName, "--wait", "--timeout", "300s"),
	}
}

func kCreateSecret(namespace Namespace) ([][]string, Closeable, error) {
	tempDir, err := os.MkdirTemp("", string(namespace))
	generateCerts(tempDir)

	return [][]string{
		{"create", "secret", "-n", string(namespace), "generic", "bolt-cert", fmt.Sprintf("--from-file=%s/public.crt", tempDir)},
		{"create", "secret", "-n", string(namespace), "generic", "https-cert", fmt.Sprintf("--from-file=%s/public.crt", tempDir)},
		{"create", "secret", "-n", string(namespace), "generic", "bolt-key", fmt.Sprintf("--from-file=%s/private.key", tempDir)},
		{"create", "secret", "-n", string(namespace), "generic", "https-key", fmt.Sprintf("--from-file=%s/private.key", tempDir)},
	}, func() error { return os.RemoveAll(tempDir) }, err
}

func helmCleanupCommands(releaseName *ReleaseName) [][]string {
	return [][]string{
		{"uninstall", string(*releaseName), "--namespace", string(releaseName.namespace())},
		{"uninstall", string(*releaseName) + "-pv"},
	}
}

func kCleanupCommands(namespace Namespace) [][]string {
	return [][]string{{"delete", "namespace", string(namespace), "--ignore-not-found", "--force", "--grace-period=0"}}
}

type Closeable func() error

var portOffset int32 = 0

func proxyBolt(t *testing.T, releaseName *ReleaseName) (int32, Closeable, error) {
	localHttpPort := 9000 + atomic.AddInt32(&portOffset, 1)
	localBoltPort := 9100 + atomic.AddInt32(&portOffset, 1)

	program := "kubectl"
	args := []string{"--namespace", string(releaseName.namespace()), "port-forward", fmt.Sprintf("service/%s", *releaseName), fmt.Sprintf("%d:7474", localHttpPort), fmt.Sprintf("%d:7687", localBoltPort)}
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

func InstallNeo4j(t *testing.T, zone Zone, project Project, releaseName *ReleaseName) (Closeable, error) {

	err := run(t, "gcloud", "container", "clusters", "get-credentials", string(CurrentCluster()))
	if err != nil {
		return nil, err
	}

	var closeables []Closeable
	addCloseable := func(closeable Closeable) {
		closeables = append([]Closeable{closeable}, closeables...)
	}

	cleanup := func() error {
		var combinedErrors error
		if closeables != nil {
			for _, closeable := range closeables {
				innerErr := closeable()
				if innerErr != nil {
					combinedErrors = combineErrors(combinedErrors, innerErr)
				}
			}
		}
		return combinedErrors
	}

	completed := false
	defer func() (err error) {
		if !completed {
			err = cleanup()
			t.Log(err)
		}
		return err
	}()

	diskName, cleanupDisk, err := createDisk(t, zone, project, releaseName)
	addCloseable(cleanupDisk)
	if err != nil {
		return cleanup, err
	}

	cleanupNamespace, err := createNamespace(t, releaseName)
	addCloseable(cleanupNamespace)
	if err != nil {
		return cleanup, err
	}

	createSecretCommands, cleanupCertificates, err := kCreateSecret(releaseName.namespace())
	addCloseable(cleanupCertificates)
	if err != nil {
		return cleanup, err
	}

	err = runAll(t, "kubectl", createSecretCommands, true)
	if err != nil {
		return cleanup, err
	}

	addCloseable(func() error { return runAll(t, "helm", helmCleanupCommands(releaseName), false) })
	err = runAll(t, "helm", helmInstallCommands(releaseName, diskName), true)
	if err != nil {
		return cleanup, err
	}

	err = run(t, "kubectl", "--namespace", string(releaseName.namespace()), "rollout", "status", "--watch", "--timeout=120s", "statefulset/"+string(*releaseName))
	if err != nil {
		return cleanup, err
	}

	completed = true
	return cleanup, err
}

func combineErrors(firstOrNil error, second error) error {
	if firstOrNil == nil {
		firstOrNil = second
	} else {
		firstOrNil = multierror.Append(firstOrNil, second)
	}
	return firstOrNil
}

func runAll(t *testing.T, bin string, commands [][]string, failFast bool) error {
	var combinedErrors error
	for _, command := range commands {
		err := run(t, bin, command...)
		if err != nil {
			if failFast {
				return err
			} else {
				combinedErrors = combineErrors(combinedErrors, fmt.Errorf("error: '%s' running %s %s", err, bin, command))
			}
		}
	}
	return combinedErrors
}

func createDisk(t *testing.T, zone Zone, project Project, releaseName *ReleaseName) (PersistentDiskName, Closeable, error) {
	diskName := releaseName.diskName()
	err := run(t, "gcloud", "compute", "disks", "create", "--size", storageSize, "--type", "pd-ssd", string(diskName), "--zone="+string(zone), "--project="+string(project))
	return PersistentDiskName(diskName), func() error { return deleteDisk(t, zone, project, string(diskName)) }, err
}

type ReleaseName string

func (r *ReleaseName) namespace() Namespace {
	return Namespace("neo4j-" + string(*r))
}

func (r *ReleaseName) diskName() PersistentDiskName {
	return PersistentDiskName(fmt.Sprintf("neo4j-data-disk-%s", *r))
}

func (r *ReleaseName) podName() string {
	return string(*r) + "-0"
}

func (r *ReleaseName) envConfigMapName() string {
	return string(*r) + "-env"
}

func (r *ReleaseName) userConfigMapName() string {
	return string(*r) + "-user-config"
}

type Namespace string
type PersistentDiskName string

func createNamespace(t *testing.T, releaseName *ReleaseName) (Closeable, error) {
	err := run(t, "kubectl", "create", "ns", string(releaseName.namespace()))
	return func() error {
		return runAll(t, "kubectl", kCleanupCommands(releaseName.namespace()), false)
	}, err
}

func deleteDisk(t *testing.T, zone Zone, project Project, diskName string) error {
	return run(t, "gcloud", "compute", "disks", "delete", diskName, "--zone="+string(zone), "--project="+string(project))
}

func run(t *testing.T, command string, args ...string) error {
	t.Logf("running: %s %s\n", command, args)
	out, err := exec.Command(command, args...).CombinedOutput()
	if out != nil {
		t.Logf("output: %s\n", out)
	}
	return err
}
