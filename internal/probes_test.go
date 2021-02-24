package internal

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/clientcmd"
	"path"
	"runtime"
	"testing"
)
var clientset *kubernetes.Clientset
func init() {
	// uses the current context in kubeconfig
	// path-to-kubeconfig -- for example, /root/.kube/config
	_, filename, _, _ := runtime.Caller(0)
	currentDir := path.Dir(filename)
	dir := path.Join(currentDir, "..")
	config, err := clientcmd.BuildConfigFromFlags("", path.Join(dir, ".kube/config"))
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