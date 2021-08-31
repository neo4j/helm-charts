package model

import (
	"errors"
	"fmt"
	"github.com/hashicorp/go-multierror"
	. "neo4j.com/helm-charts-tests/internal/helpers"
	"neo4j.com/helm-charts-tests/internal/resources"
	"os"
	"os/exec"
	"strings"
	"testing"
)

var DefaultHelmTemplateReleaseName = releaseName("my-release")

func HelmTemplate(t *testing.T, chart HelmChart, helmTemplateArgs []string, moreHelmTemplateArgs ...string) (*K8sResources, error) {

	helmTemplateArgs = append(minHelmCommand("template", &DefaultHelmTemplateReleaseName, chart), helmTemplateArgs...)

	return RunHelmCommand(t, helmTemplateArgs, moreHelmTemplateArgs...)
}

func RunHelmCommand(t *testing.T, helmTemplateArgs []string, moreHelmTemplateArgs ...string) (*K8sResources, error) {
	if helmTemplateArgs != nil && moreHelmTemplateArgs != nil {
		helmTemplateArgs = append(helmTemplateArgs, moreHelmTemplateArgs...)
	} else if helmTemplateArgs == nil {
		helmTemplateArgs = moreHelmTemplateArgs
	}

	if helmTemplateArgs == nil {
		helmTemplateArgs = make([]string, 0)
	}

	program := "helm"
	t.Logf("running: %s %s\n", program, helmTemplateArgs)
	stdout, stderr, err := RunCommand(exec.Command(program, helmTemplateArgs...))

	if err != nil {
		return nil, multierror.Append(errors.New("Error running helm template"), err, fmt.Errorf("stdout: %s\nstderr: %s", stdout, stderr))
	}

	return decodeK8s(stdout)
}

func minHelmCommand(helmCommand string, releaseName ReleaseName, chart HelmChart) []string {
	return []string{helmCommand, releaseName.String(), chart.getPath(), "--namespace", string(releaseName.Namespace())}
}

func BaseHelmCommand(helmCommand string, releaseName ReleaseName, chart Neo4jHelmChart, diskName *PersistentDiskName, extraHelmArguments ...string) []string {

	var helmArgs = minHelmCommand(helmCommand, releaseName, chart)
	helmArgs = append(helmArgs,
		"--set", "volumes.data.mode=volume",
		"--set", "volumes.data.volume.gcePersistentDisk.pdName="+string(*diskName),
		"--set", "neo4j.password="+DefaultPassword,
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

func HelmTemplateFromYamlFile(t *testing.T, chart HelmChart, values resources.YamlFile, extraHelmArgs ...string) (*K8sResources, error) {
	args := minHelmCommand("template", &DefaultHelmTemplateReleaseName, chart)
	return RunHelmCommand(t, args, append(extraHelmArgs, values.HelmArgs()...)...)
}

func LoadBalancerHelmCommand(helmCommand string, releaseName ReleaseName, extraHelmArguments ...string) []string {

	var helmArgs []string
	if helmCommand == "uninstall" {
		helmArgs = []string{"uninstall", releaseName.String(), "--namespace", string(releaseName.Namespace())}
	} else {
		helmArgs = minHelmCommand(helmCommand, releaseName, LoadBalancerHelmChart)
	}

	if extraHelmArguments != nil && len(extraHelmArguments) > 0 {
		helmArgs = append(helmArgs, extraHelmArguments...)
	}

	return helmArgs
}
