package model

import (
	"errors"
	"fmt"
	"github.com/hashicorp/go-multierror"
	. "github.com/neo4j/helm-charts/internal/helpers"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strings"
	"testing"
)

func CheckError(err error) {
	if err != nil {
		log.Panic(err)
	}
}

// This changes the working directory to the parent directory if the current working directory doesn't contain a directory called "internal"
func setWorkingDir() {

	var _, thisFile, _, _ = runtime.Caller(0)
	var thisDir = path.Dir(thisFile)

	files, err := ioutil.ReadDir(".")
	CheckError(err)
	for _, file := range files {
		if file.Name() == "internal" {
			return
		}
	}

	files, err = ioutil.ReadDir(thisDir)
	CheckError(err)
	for _, file := range files {
		if file.Name() == "internal" {
			return
		}
	}
	dir := path.Join(thisDir, "../..")
	err = os.Chdir(dir)
	CheckError(err)
	files, err = ioutil.ReadDir(".")
	CheckError(err)
	for _, file := range files {
		if file.Name() == "internal" {
			return
		}
	}
	panic("unable to set current dir correctly")
}

func init() {
	setWorkingDir()

	os.Setenv("KUBECONFIG", ".kube/config")
}

var DefaultHelmTemplateReleaseName = ReleaseName("my-release")

func HelmTemplate(t *testing.T, helmTemplateArgs []string, moreHelmTemplateArgs ...string) (*K8sResources, error) {
	if helmTemplateArgs != nil {
		helmTemplateArgs = append(helmTemplateArgs, moreHelmTemplateArgs...)
	}

	if len(helmTemplateArgs) == 0 || helmTemplateArgs[0] != "template" {
		helmTemplateArgs = append(minHelmCommand("template", &DefaultHelmTemplateReleaseName), helmTemplateArgs...)
	}

	program := "helm"
	t.Logf("running: %s %s\n", program, helmTemplateArgs)
	stdout, stderr, err := RunCommand(exec.Command(program, helmTemplateArgs...))

	if err != nil {
		return nil, multierror.Append(errors.New("Error running helm template"), err, fmt.Errorf("stdout: %s\nstderr: %s", stdout, stderr))
	}

	return decodeK8s(stdout)
}

func minHelmCommand(helmCommand string, releaseName *ReleaseName) []string {
	return []string{helmCommand, string(*releaseName), "./neo4j-standalone", "--namespace", string(releaseName.Namespace())}
}

func BaseHelmCommand(helmCommand string, releaseName *ReleaseName, extraHelmArguments ...string) []string {

	var helmArgs = minHelmCommand(helmCommand, releaseName)
	helmArgs = append(helmArgs,
		"--create-namespace",
		"--set", "volumes.data.mode=selector",
		"--set", "volumes.data.selector.requests.storage="+StorageSize,
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

func HelmTemplateFromYamlFile(t *testing.T, filepath string) (*K8sResources, error) {
	return HelmTemplate(t, minHelmCommand("template", &DefaultHelmTemplateReleaseName), "-f", filepath)
}
