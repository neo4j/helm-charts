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

// getNodeWithLabel returns the first node that carries the given "key=value" label.
func getNodeWithLabel(labelName string) (*coreV1.Node, error) {
	matches, err := getNodesWithLabel(labelName)
	if err != nil {
		return nil, err
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf("No node with the label %s found", labelName)
	}
	return matches[0], nil
}

// getNodesWithLabel returns every node that carries the given "key=value" label.
func getNodesWithLabel(labelName string) ([]*coreV1.Node, error) {
	nodes, err := getNodesList()
	if err != nil {
		return nil, err
	}
	parts := strings.SplitN(labelName, "=", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("label %q is not in key=value form", labelName)
	}
	labelKey, labelValue := parts[0], parts[1]
	var matches []*coreV1.Node
	for i := range nodes.Items {
		node := &nodes.Items[i]
		if v, ok := node.ObjectMeta.Labels[labelKey]; ok && v == labelValue {
			matches = append(matches, node)
		}
	}
	return matches, nil
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

func ExecInPod(releaseName model.ReleaseName, cmd []string, podName string) (string, string, error) {
	name := releaseName.PodName()
	if podName != "" {
		name = podName
	}

	var lastErr error
	for attempt := 0; attempt < 6; attempt++ {
		var stdout, stderr bytes.Buffer
		req := Clientset.CoreV1().RESTClient().Post().Resource("pods").Name(name).
			Namespace(string(releaseName.Namespace())).SubResource("exec")
		option := &coreV1.PodExecOptions{
			Command: cmd,
			Stdin:   false,
			Stdout:  true,
			Stderr:  true,
			TTY:     false,
		}
		req.VersionedParams(option, scheme.ParameterCodec)
		executor, err := remotecommand.NewSPDYExecutor(Config, "POST", req.URL())
		if err != nil {
			return "", "", err
		}
		err = executor.Stream(remotecommand.StreamOptions{
			Stdout: &stdout,
			Stderr: &stderr,
		})
		if err != nil {
			if strings.Contains(err.Error(), "container not found") {
				lastErr = err
				time.Sleep(10 * time.Second)
				continue
			}
			return "", "", err
		}
		s := strings.TrimSuffix(stdout.String(), "\n")
		return s, stderr.String(), nil
	}
	return "", "", fmt.Errorf("ExecInPod failed after 6 attempts: %w", lastErr)
}
