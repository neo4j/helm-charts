package integration_tests

import (
	"bytes"
	"context"
	"fmt"
	"github.com/neo4j/helm-charts/internal/model"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	coreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"time"

	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"
	"os"
	"strings"
	"testing"
)

func init() {
	var err error
	// gets kubeconfig from env variable
	Config, err = clientcmd.BuildConfigFromFlags("", os.Getenv("KUBECONFIG"))
	CheckError(err)
	Clientset, err = kubernetes.NewForConfig(Config)
	CheckError(err)
}
func CheckProbes(t *testing.T, releaseName *model.ReleaseName) error {

	// getting Probes values from values.yaml
	type ValuesYaml struct {
		ReadinessProbe struct {
			FailureThreshold int32 `yaml:"failureThreshold"`
			TimeoutSeconds   int32 `yaml:"timeoutSeconds"`
			PeriodSeconds    int32 `yaml:"periodSeconds"`
		} `yaml:"readinessProbe"`
		LivenessProbe struct {
			FailureThreshold int32 `yaml:"failureThreshold"`
			TimeoutSeconds   int32 `yaml:"timeoutSeconds"`
			PeriodSeconds    int32 `yaml:"periodSeconds"`
		} `yaml:"livenessProbe"`
	}

	var fileName = "neo4j-standalone/values.yaml"

	yamlFile, err := ioutil.ReadFile(fileName)
	if err != nil {
		return fmt.Errorf("Error reading YAML file: %v", err)
	}

	var yamlConfig ValuesYaml
	err = yaml.Unmarshal(yamlFile, &yamlConfig)
	if err != nil {
		return fmt.Errorf("Error parsing YAML file: %v", err)
	}
	pods_map := make(map[string]int32)

	podsLiveness := "neo4j_LivenessProbe"
	podsReadiness := "neo4j_ReadinessProbe"

	for start := time.Now(); time.Since(start) < 60*time.Second; {
		pods, err := Clientset.CoreV1().Pods(string(releaseName.Namespace())).List(context.TODO(), v1.ListOptions{})
		if err != nil {
			return fmt.Errorf("Failed to get Pods options: %v", err)
		}

		var emptyProbesFound = false
		for _, opt := range pods.Items {
			for _, container := range opt.Spec.Containers {
				if container.LivenessProbe.PeriodSeconds == 0 || container.ReadinessProbe.PeriodSeconds == 0 {
					emptyProbesFound = true
				}
				pods_map[container.Name+"_LivenessProbe"] = container.LivenessProbe.PeriodSeconds
				pods_map[container.Name+"_ReadinessProbe"] = container.ReadinessProbe.PeriodSeconds
			}
		}
		if emptyProbesFound {
			continue
		} else {
			break
		}
	}

	yamlConfigLiveness := yamlConfig.LivenessProbe.PeriodSeconds
	// TODO: these assertions seem slightly flaky. I think we might need to wait for the pod to be running before checking them or something
	assert.Equal(t, yamlConfigLiveness, pods_map[podsLiveness], "LivenessProbe mismatch")

	yamlConfigReadiness := yamlConfig.ReadinessProbe.PeriodSeconds
	assert.Equal(t, yamlConfigReadiness, pods_map[podsReadiness], "ReadinessProbe mismatch")
	return nil
}

func CheckServiceAnnotations(t *testing.T, releaseName *model.ReleaseName) (err error) {
	var services = getAllServices(t, releaseName)
	assert.Equal(t, 3, len(services.Items))

	// by default they should have no annotations
	for _, service := range services.Items {
		assert.Empty(t, getOurAnnotations(service))
	}

	// when we add annotations via helm
	err = runAll(t, "helm", [][]string{
		model.BaseHelmCommand("upgrade", releaseName,
			"--set", "services.neo4j.annotations.foo=bar",
			"--set", "services.admin.annotations.foo=bar",
			"--set", "services.default.annotations.foo=bar",
		),
	}, true)
	if err != nil {
		return err
	}

	// then the services get annotations
	services = getAllServices(t, releaseName)
	assert.Equal(t, 3, len(services.Items))

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

func getAllServices(t *testing.T, releaseName *model.ReleaseName) *coreV1.ServiceList {
	services, err := Clientset.CoreV1().Services(string(releaseName.Namespace())).List(context.TODO(), v1.ListOptions{})
	assert.NoError(t, err)
	return services
}

func RunAsNonRoot(t *testing.T, releaseName *model.ReleaseName) error {
	pods, err := Clientset.CoreV1().Pods(string(releaseName.Namespace())).List(context.TODO(), v1.ListOptions{})
	if err != nil {
		return fmt.Errorf("Failed to get Pods options: %v", err)
	}
	assert.NotEmpty(t, pods.Items, "pods.Items is empty")
	for _, opt := range pods.Items {
		assert.Equal(t, true, *opt.Spec.SecurityContext.RunAsNonRoot)
	}
	return nil
}

func CheckExecInPod(t *testing.T, releaseName *model.ReleaseName) error {
	cmd := []string{
		"bash",
		"-c",
		"id -u",
	}

	stdout, stderr, err := ExecInPod(releaseName, cmd)

	assert.NoError(t, err)
	assert.Equal(t, "7474", stdout, "UID is different than expected")
	assert.Empty(t, stderr, "stderr is not empty")

	return err
}

func ExecInPod(releaseName *model.ReleaseName, cmd []string) (string, string, error) {

	var (
		stdout bytes.Buffer
		stderr bytes.Buffer
	)
	req := Clientset.CoreV1().RESTClient().Post().Resource("pods").Name(releaseName.PodName()).
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
