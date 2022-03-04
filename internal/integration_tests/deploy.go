package integration_tests

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
	. "github.com/neo4j/helm-charts/internal/helpers"
	"github.com/neo4j/helm-charts/internal/integration_tests/gcloud"
	"github.com/neo4j/helm-charts/internal/model"
	"io"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"log"
	"math/big"
	"os"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func CheckError(err error) {
	if err != nil {
		log.Panic(err)
	}
}

var (
	Clientset *kubernetes.Clientset
	Config    *restclient.Config
)

func init() {
	os.Setenv("KUBECONFIG", ".kube/config")
	var err error
	// gets kubeconfig from env variable
	Config, err = clientcmd.BuildConfigFromFlags("", os.Getenv("KUBECONFIG"))
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

func kCreateSecret(namespace model.Namespace) ([][]string, Closeable, error) {
	tempDir, err := os.MkdirTemp("", string(namespace))
	generateCerts(tempDir)

	return [][]string{
		{"create", "secret", "-n", string(namespace), "generic", "bolt-cert", fmt.Sprintf("--from-file=%s/public.crt", tempDir)},
		{"create", "secret", "-n", string(namespace), "generic", "https-cert", fmt.Sprintf("--from-file=%s/public.crt", tempDir)},
		{"create", "secret", "-n", string(namespace), "generic", "bolt-key", fmt.Sprintf("--from-file=%s/private.key", tempDir)},
		{"create", "secret", "-n", string(namespace), "generic", "https-key", fmt.Sprintf("--from-file=%s/private.key", tempDir)},
	}, func() error { return os.RemoveAll(tempDir) }, err
}

func helmCleanupCommands(releaseName model.ReleaseName) [][]string {
	return [][]string{
		{"uninstall", releaseName.String(), "--namespace", string(releaseName.Namespace())},
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
		args = []string{"--namespace", string(releaseName.Namespace()), "port-forward", fmt.Sprintf("service/%s-neo4j", releaseName), fmt.Sprintf("%d:7474", localHttpPort), fmt.Sprintf("%d:7687", localBoltPort)}
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

func InstallNeo4jInGcloud(t *testing.T, zone gcloud.Zone, project gcloud.Project, releaseName model.ReleaseName, chart model.Neo4jHelmChart, extraHelmInstallArgs ...string) (Closeable, error) {

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
	addCloseable(cleanupGcloud)
	if err != nil {
		return AsCloseable(closeables), err
	}

	addCloseable(func() error { return runAll(t, "helm", helmCleanupCommands(releaseName), false) })
	// delete the statefulset like this otherwise the pods will hang around for their termination grace period
	addCloseable(func() error {
		return runAll(t, "kubectl", [][]string{
			{"delete", "statefulset", releaseName.String(), "--namespace", string(releaseName.Namespace()), "--grace-period=0", "--force", "--ignore-not-found"},
			{"delete", "pod", releaseName.PodName(), "--namespace", string(releaseName.Namespace()), "--grace-period=0", "--wait", "--timeout=120s", "--ignore-not-found"},
		}, false)
	})
	err = run(t, "helm", model.BaseHelmCommand("install", releaseName, chart, model.Neo4jEdition, diskName, extraHelmInstallArgs...)...)

	if err != nil {
		return AsCloseable(closeables), err
	}

	completed = true
	return AsCloseable(closeables), err
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
