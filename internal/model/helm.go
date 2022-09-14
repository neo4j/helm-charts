package model

import (
	"errors"
	"fmt"
	"github.com/hashicorp/go-multierror"
	. "github.com/neo4j/helm-charts/internal/helpers"
	"github.com/neo4j/helm-charts/internal/resources"
	"io/ioutil"
	"k8s.io/utils/env"
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

var DefaultHelmTemplateReleaseName = releaseName("my-release")
var Neo4jEdition = strings.ToLower(env.GetString("NEO4J_EDITION", "enterprise"))

func HelmTemplateForRelease(t *testing.T, releaseName ReleaseName, chart HelmChart, helmTemplateArgs []string, moreHelmTemplateArgs ...string) (*K8sResources, error) {

	helmTemplateArgs = append(minHelmCommand("template", releaseName, chart), helmTemplateArgs...)

	return RunHelmCommand(t, helmTemplateArgs, moreHelmTemplateArgs...)
}

func HelmTemplate(t *testing.T, chart HelmChart, helmTemplateArgs []string, moreHelmTemplateArgs ...string) (*K8sResources, error) {

	return HelmTemplateForRelease(t, &DefaultHelmTemplateReleaseName, chart, helmTemplateArgs, moreHelmTemplateArgs...)
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

func BaseHelmCommand(helmCommand string, releaseName ReleaseName, chart Neo4jHelmChart, edition string, diskName *PersistentDiskName, extraHelmArguments ...string) []string {

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

	if edition != "" {
		if !chart.SupportsEdition(edition) {
			panic(fmt.Sprintf("Helm chart %s does not support neo4j edition %s", chart.Name(), edition))
		}
		helmArgs = append(helmArgs, "--set", "neo4j.edition="+edition)
	}

	if strings.EqualFold(edition, "enterprise") {
		helmArgs = append(helmArgs, "--set", "neo4j.acceptLicenseAgreement=yes")
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

// HeadlessServiceHelmCommand will perform helm install or helm uninstall on headless service chart
func HeadlessServiceHelmCommand(helmCommand string, releaseName ReleaseName, extraHelmArguments ...string) []string {

	var helmArgs []string
	if helmCommand == "uninstall" {
		helmArgs = []string{"uninstall", releaseName.String(), "--namespace", string(releaseName.Namespace())}
	} else {
		helmArgs = minHelmCommand(helmCommand, releaseName, HeadlessServiceHelmChart)
	}

	if extraHelmArguments != nil && len(extraHelmArguments) > 0 {
		helmArgs = append(helmArgs, extraHelmArguments...)
	}

	return helmArgs
}
