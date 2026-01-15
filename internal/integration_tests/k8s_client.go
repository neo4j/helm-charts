package integration_tests

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/neo4j/helm-charts/internal/model"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
	coreV1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"
)

func init() {
	var err error
	// gets kubeconfig from env variable
	Config, err = clientcmd.BuildConfigFromFlags("", os.Getenv("KUBECONFIG"))
	CheckError(err)
	Clientset, err = kubernetes.NewForConfig(Config)
	CheckError(err)
}

type ProbeConfig struct {
	HTTPGet             *HTTPGetAction   `yaml:"httpGet,omitempty"`
	TCPSocket           *TCPSocketAction `yaml:"tcpSocket,omitempty"`
	FailureThreshold    int32            `yaml:"failureThreshold"`
	TimeoutSeconds      int32            `yaml:"timeoutSeconds"`
	PeriodSeconds       int32            `yaml:"periodSeconds"`
	InitialDelaySeconds int32            `yaml:"initialDelaySeconds,omitempty"`
}

type HTTPGetAction struct {
	Path string `yaml:"path"`
	Port int32  `yaml:"port"`
}

type TCPSocketAction struct {
	Port int32 `yaml:"port"`
}

type ValuesYaml struct {
	ReadinessProbe ProbeConfig `yaml:"readinessProbe"`
	LivenessProbe  ProbeConfig `yaml:"livenessProbe"`
	StartupProbe   ProbeConfig `yaml:"startupProbe"`
}

func CheckProbes(t *testing.T, releaseName model.ReleaseName) error {
	var fileName = "neo4j/values.yaml"
	yamlFile, err := ioutil.ReadFile(fileName)
	if err != nil {
		return fmt.Errorf("Error reading YAML file: %v", err)
	}

	var yamlConfig ValuesYaml
	err = yaml.Unmarshal(yamlFile, &yamlConfig)
	if err != nil {
		return fmt.Errorf("Error parsing YAML file: %v", err)
	}

	for start := time.Now(); time.Since(start) < 60*time.Second; {
		options := v1.ListOptions{
			LabelSelector: fmt.Sprintf("helm.neo4j.com/instance=%s", releaseName.String()),
		}
		pods, err := Clientset.CoreV1().Pods(string(releaseName.Namespace())).List(context.TODO(), options)
		if err != nil {
			return fmt.Errorf("Failed to get Pods options: %v", err)
		}

		for _, pod := range pods.Items {
			for _, container := range pod.Spec.Containers {
				// Check readiness probe
				if yamlConfig.ReadinessProbe.HTTPGet != nil {
					assert.Equal(t, yamlConfig.ReadinessProbe.HTTPGet.Path, container.ReadinessProbe.HTTPGet.Path)
					assert.Equal(t, yamlConfig.ReadinessProbe.HTTPGet.Port, container.ReadinessProbe.HTTPGet.Port.IntVal)
				}
				if yamlConfig.ReadinessProbe.TCPSocket != nil {
					assert.Equal(t, yamlConfig.ReadinessProbe.TCPSocket.Port, container.ReadinessProbe.TCPSocket.Port.IntVal)
				}
				assert.Equal(t, yamlConfig.ReadinessProbe.FailureThreshold, container.ReadinessProbe.FailureThreshold)
				assert.Equal(t, yamlConfig.ReadinessProbe.TimeoutSeconds, container.ReadinessProbe.TimeoutSeconds)
				assert.Equal(t, yamlConfig.ReadinessProbe.PeriodSeconds, container.ReadinessProbe.PeriodSeconds)

				// Check liveness probe
				if yamlConfig.LivenessProbe.HTTPGet != nil {
					assert.Equal(t, yamlConfig.LivenessProbe.HTTPGet.Path, container.LivenessProbe.HTTPGet.Path)
					assert.Equal(t, yamlConfig.LivenessProbe.HTTPGet.Port, container.LivenessProbe.HTTPGet.Port.IntVal)
				}
				if yamlConfig.LivenessProbe.TCPSocket != nil {
					assert.Equal(t, yamlConfig.LivenessProbe.TCPSocket.Port, container.LivenessProbe.TCPSocket.Port.IntVal)
				}
				assert.Equal(t, yamlConfig.LivenessProbe.FailureThreshold, container.LivenessProbe.FailureThreshold)
				assert.Equal(t, yamlConfig.LivenessProbe.TimeoutSeconds, container.LivenessProbe.TimeoutSeconds)
				assert.Equal(t, yamlConfig.LivenessProbe.PeriodSeconds, container.LivenessProbe.PeriodSeconds)
			}
		}
		return nil
	}
	return fmt.Errorf("Timed out waiting for probes to be configured")
}

func CheckServiceAnnotations(t *testing.T, releaseName model.ReleaseName, chart model.Neo4jHelmChartBuilder) (err error) {
	services, err := getAllServices(releaseName.Namespace())
	if err != nil {
		return err
	}
	expectedServiceCount := 3

	assert.Equal(t, expectedServiceCount, len(services.Items))

	// by default they should have no annotations
	for _, service := range services.Items {
		assert.Empty(t, getOurAnnotations(service))
	}

	// when we add annotations via helm
	err = runAll(t, "helm", [][]string{
		model.BaseHelmCommand("upgrade", releaseName, chart, model.Neo4jEdition,
			"--set", "services.admin.annotations.foo=bar",
			"--set", "services.neo4j.annotations.foo=bar",
			"--set", "services.default.annotations.foo=bar", "--set", "neo4j.name="+model.DefaultNeo4jName),
	}, true)
	if err != nil {
		return err
	}

	// then the services get annotations
	services, err = getAllServices(releaseName.Namespace())
	if err != nil {
		return err
	}

	assert.Equal(t, expectedServiceCount, len(services.Items))

	for _, service := range services.Items {
		assert.Equal(t, "bar", getOurAnnotations(service)["foo"])
	}
	return err
}

func getOurAnnotations(service coreV1.Service) map[string]string {
	ourAnnotations := map[string]string{}
	prefixesToIgnore := []string{
		"cloud.google.com/",
		"meta.helm.sh/",
		"helm.sh/",
	}
	for key, value := range service.Annotations {
		if !matchesAnyPrefix(prefixesToIgnore, key) {
			ourAnnotations[key] = value
		}
	}
	return ourAnnotations
}

func matchesAnyPrefix(knownPrefixes []string, key string) bool {
	for _, prefix := range knownPrefixes {
		if strings.HasPrefix(key, prefix) {
			return true
		}
	}
	return false
}

func getAllPods(namespace model.Namespace) (*coreV1.PodList, error) {
	return Clientset.CoreV1().Pods(string(namespace)).List(context.TODO(), v1.ListOptions{})
}

func getSpecificPod(namespace model.Namespace, podName string) (*coreV1.Pod, error) {
	return Clientset.CoreV1().Pods(string(namespace)).Get(context.TODO(), podName, v1.GetOptions{})
}

func getPodsWithSpecificLabel(namespace model.Namespace, label string) (*coreV1.PodList, error) {
	return Clientset.CoreV1().Pods(string(namespace)).List(context.TODO(), v1.ListOptions{LabelSelector: label})
}

func getNodesList() (*coreV1.NodeList, error) {
	return Clientset.CoreV1().Nodes().List(context.TODO(), v1.ListOptions{})
}

// getNodeWithLabel returns the node with the given label
func getNodeWithLabel(labelName string) (*coreV1.Node, error) {
	nodes, err := getNodesList()
	if err != nil {
		return nil, err
	}
	labelKey := strings.Split(labelName, "=")[0]
	labelValue := strings.Split(labelName, "=")[1]
	var nodeSelectorNode *coreV1.Node
	for _, node := range nodes.Items {
		if value, present := node.ObjectMeta.Labels[labelKey]; present {
			if value == labelValue {
				nodeSelectorNode = &node
				break
			}
		}
	}
	if nodeSelectorNode == nil {
		return nil, fmt.Errorf("No node with the label %s found", labelName)
	}
	return nodeSelectorNode, nil
}

func getManifest(namespace model.Namespace) (*model.K8sResources, error) {

	pods, err := getAllPods(namespace)
	if err != nil {
		return nil, err
	}

	services, err := getAllServices(namespace)
	if err != nil {
		return nil, err
	}

	endpoints, err := getAllEndpoints(namespace)
	if err != nil {
		return nil, err
	}

	manifest := model.NewK8sResources(nil, []schema.GroupVersionKind{
		pods.GroupVersionKind(),
		services.GroupVersionKind(),
		endpoints.GroupVersionKind(),
	})

	manifest.AddPods(pods.Items)
	manifest.AddServices(services.Items)
	manifest.AddEndpoints(endpoints.Items)

	return manifest, err
}

func getAllEndpoints(namespace model.Namespace) (*coreV1.EndpointsList, error) {
	return Clientset.CoreV1().Endpoints(string(namespace)).List(context.TODO(), v1.ListOptions{})
}

func getAllServices(namespace model.Namespace) (*coreV1.ServiceList, error) {
	return Clientset.CoreV1().Services(string(namespace)).List(context.TODO(), v1.ListOptions{})
}

func getAllSecrets(namespace model.Namespace) (*coreV1.SecretList, error) {
	return Clientset.CoreV1().Secrets(string(namespace)).List(context.TODO(), v1.ListOptions{})
}

func getSpecificSecret(namespace model.Namespace, secretName string) (*coreV1.Secret, error) {
	return Clientset.CoreV1().Secrets(string(namespace)).Get(context.TODO(), secretName, v1.GetOptions{})
}

func RunAsNonRoot(t *testing.T, releaseName model.ReleaseName) error {
	options := v1.ListOptions{
		LabelSelector: fmt.Sprintf("helm.neo4j.com/instance=%s", releaseName.String()),
	}
	pods, err := Clientset.CoreV1().Pods(string(releaseName.Namespace())).List(context.TODO(), options)
	if err != nil {
		return fmt.Errorf("Failed to get Pods options: %v", err)
	}
	assert.NotEmpty(t, pods.Items, "pods.Items is empty")
	for _, opt := range pods.Items {
		assert.Equal(t, true, *opt.Spec.SecurityContext.RunAsNonRoot)
	}
	return nil
}

func CheckExecInPod(t *testing.T, releaseName model.ReleaseName) error {
	cmd := []string{
		"bash",
		"-c",
		"id -u",
	}

	stdout, stderr, err := ExecInPod(releaseName, cmd, "")

	assert.NoError(t, err)
	assert.Equal(t, "7474", stdout, "UID is different than expected")
	assert.Empty(t, stderr, "stderr is not empty")

	return err
}

// waitForPodRunning waits for a pod to be in Running state with at least one container running
// This is sufficient for exec operations that don't require readiness probes to pass
func waitForPodRunning(namespace model.Namespace, podName string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			pod, err := Clientset.CoreV1().Pods(string(namespace)).Get(context.Background(), podName, v1.GetOptions{})
			if err != nil {
				return fmt.Errorf("timeout waiting for pod %s/%s to be running: %v", namespace, podName, err)
			}
			return fmt.Errorf("timeout waiting for pod %s/%s to be running. Current phase: %s, Conditions: %v",
				namespace, podName, pod.Status.Phase, pod.Status.Conditions)
		case <-ticker.C:
			pod, err := Clientset.CoreV1().Pods(string(namespace)).Get(context.Background(), podName, v1.GetOptions{})
			if err != nil {
				// Pod might not exist yet, continue waiting
				continue
			}

			// Check if pod is running
			if pod.Status.Phase != coreV1.PodRunning {
				continue
			}

			// Check if at least one container is running (not necessarily ready)
			// For exec operations, we just need the container to be running
			hasRunningContainer := false
			for _, containerStatus := range pod.Status.ContainerStatuses {
				if containerStatus.State.Running != nil {
					hasRunningContainer = true
					break
				}
			}

			if hasRunningContainer && len(pod.Status.ContainerStatuses) > 0 {
				return nil
			}
		}
	}
}

// waitForPodReady waits for a pod to be in Running state with all containers ready
// This requires readiness probes to pass, which is needed for operations that require Neo4j to be fully ready
func waitForPodReady(namespace model.Namespace, podName string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			pod, err := Clientset.CoreV1().Pods(string(namespace)).Get(context.Background(), podName, v1.GetOptions{})
			if err != nil {
				return fmt.Errorf("timeout waiting for pod %s/%s to be ready: %v", namespace, podName, err)
			}
			return fmt.Errorf("timeout waiting for pod %s/%s to be ready. Current phase: %s, Conditions: %v",
				namespace, podName, pod.Status.Phase, pod.Status.Conditions)
		case <-ticker.C:
			pod, err := Clientset.CoreV1().Pods(string(namespace)).Get(context.Background(), podName, v1.GetOptions{})
			if err != nil {
				// Pod might not exist yet, continue waiting
				continue
			}

			// Check if pod is running
			if pod.Status.Phase != coreV1.PodRunning {
				continue
			}

			// Check if all containers are ready
			allContainersReady := true
			for _, containerStatus := range pod.Status.ContainerStatuses {
				if !containerStatus.Ready {
					allContainersReady = false
					break
				}
			}

			if allContainersReady && len(pod.Status.ContainerStatuses) > 0 {
				return nil
			}
		}
	}
}

func ExecInPod(releaseName model.ReleaseName, cmd []string, podName string) (string, string, error) {
	return ExecInPodWithWait(releaseName, cmd, podName, true, false, 5*time.Minute)
}

// ExecInPodWithWait executes a command in a pod, optionally waiting for pod readiness or running state
// waitForReady: if true, waits for pod before executing; if false, executes immediately
// requireReadiness: if true and waitForReady is true, waits for readiness probe to pass; if false, waits only for container to be running
func ExecInPodWithWait(releaseName model.ReleaseName, cmd []string, podName string, waitForReady bool, requireReadiness bool, timeout time.Duration) (string, string, error) {
	name := releaseName.PodName()
	if podName != "" {
		name = podName
	}

	// Wait for pod before executing command
	if waitForReady {
		var err error
		if requireReadiness {
			// Wait for readiness probe to pass (for operations that need Neo4j to be fully ready)
			err = waitForPodReady(releaseName.Namespace(), name, timeout)
			if err != nil {
				return "", "", fmt.Errorf("pod %s/%s not ready: %v", releaseName.Namespace(), name, err)
			}
		} else {
			// Wait only for container to be running (for simple operations like ls, cat, etc.)
			err = waitForPodRunning(releaseName.Namespace(), name, timeout)
			if err != nil {
				return "", "", fmt.Errorf("pod %s/%s not running: %v", releaseName.Namespace(), name, err)
			}
		}
	}

	var (
		stdout bytes.Buffer
		stderr bytes.Buffer
	)
	req := Clientset.CoreV1().RESTClient().Post().Resource("pods").Name(name).
		Namespace(string(releaseName.Namespace())).SubResource("exec")
	option := &coreV1.PodExecOptions{
		Command: cmd,
		Stdin:   false,
		Stdout:  true,
		Stderr:  true,
		TTY:     false,
	}
	req.VersionedParams(
		option,
		scheme.ParameterCodec,
	)
	exec, err := remotecommand.NewSPDYExecutor(Config, "POST", req.URL())
	if err != nil {
		return "", "", err
	}
	err = exec.Stream(remotecommand.StreamOptions{
		Stdout: &stdout,
		Stderr: &stderr,
	})
	if err != nil {
		return "", "", err
	}
	s := stdout.String()
	s = strings.TrimSuffix(s, "\n")
	e := stderr.String()
	return s, e, nil
}
