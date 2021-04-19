package internal

import (
	"bytes"
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	coreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"
	"os"
	"strings"
	"testing"
)
func init() {
	var err error
	// gets kubeconfig from env variable
	config, err = clientcmd.BuildConfigFromFlags("", os.Getenv("KUBECONFIG"))
	CheckError(err)
	clientset, err = kubernetes.NewForConfig(config)
	CheckError(err)
}
func CheckProbes(t *testing.T) error {
	pods, err := clientset.CoreV1().Pods("neo4j").List(context.TODO(), v1.ListOptions{})
	if err != nil {
		return fmt.Errorf("Failed to get Pods options: %v", err)
	}
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

	var fileName string = "neo4j/values.yaml"

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
	for _, opt := range pods.Items {
		for _, container := range opt.Spec.Containers {
			pods_map[container.Name + "_LivenessProbe"] = container.LivenessProbe.PeriodSeconds
			pods_map[container.Name + "_ReadinessProbe"] = container.ReadinessProbe.PeriodSeconds
		}
	}
	podsLiveness := "neo4j_LivenessProbe"
	yamlConfigLiveness := yamlConfig.LivenessProbe.PeriodSeconds
	assert.Equal(t, pods_map[podsLiveness], yamlConfigLiveness, "LivenessProbe mismatch")
	podsReadiness := "neo4j_ReadinessProbe"
	yamlConfigReadiness := yamlConfig.ReadinessProbe.PeriodSeconds
	assert.Equal(t, pods_map[podsReadiness], yamlConfigReadiness, "ReadinessProbe mismatch")
	return nil
}
func RunAsNonRoot(t *testing.T) error {
	pods, err := clientset.CoreV1().Pods("neo4j").List(context.TODO(), v1.ListOptions{})
	if err != nil {
		return fmt.Errorf("Failed to get Pods options: %v", err)
	}
	assert.NotEmpty(t, pods.Items, "pods.Items is empty")
	for _, opt := range pods.Items {
		assert.Equal(t, true, *opt.Spec.SecurityContext.RunAsNonRoot)
	}
		return nil
}
func ExecInPod(t *testing.T) error {
	cmd := []string{
		"bash",
		"-c",
		"id -u",
	}
	var (
	stdout bytes.Buffer
	stderr bytes.Buffer
	)
	req := clientset.CoreV1().RESTClient().Post().Resource("pods").Name("neo4j-0").
		Namespace("neo4j").SubResource("exec")
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
	exec, err := remotecommand.NewSPDYExecutor(config, "POST", req.URL())
	if err != nil {
		return err
	}
	err = exec.Stream(remotecommand.StreamOptions{
		Stdout: &stdout,
		Stderr: &stderr,
	})
	if err != nil {
		return err
	}
	s := stdout.String()
	s = strings.TrimSuffix(s, "\n")
	assert.Equal(t, "7474", s, "UID is different than expected")
	e :=stderr.String()
	assert.Empty(t, e, "stderr is not empty")

	return nil
}