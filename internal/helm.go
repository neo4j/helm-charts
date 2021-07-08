package internal

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/hashicorp/go-multierror"
	"os/exec"
	"testing"
)

var DefaultHelmTemplateReleaseName = ReleaseName("my-release")

// RunCommand runs the command and returns its standard
// output and standard error.
func RunCommand(c *exec.Cmd) ([]byte, []byte, error) {
	if c.Stdout != nil {
		return nil, nil, errors.New("exec: Stdout already set")
	}
	if c.Stderr != nil {
		return nil, nil, errors.New("exec: Stderr already set")
	}
	var a bytes.Buffer
	var b bytes.Buffer
	c.Stdout = &a
	c.Stderr = &b
	err := c.Run()
	return a.Bytes(), b.Bytes(), err
}

func helmTemplateFromYamlFile(t *testing.T, filepath string) (*K8sResources, error) {
	return helmTemplate(t, minHelmCommand("template", &DefaultHelmTemplateReleaseName), "-f", filepath)
}

func helmTemplate(t *testing.T, helmTemplateArgs []string, moreHelmTemplateArgs ...string) (*K8sResources, error) {
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
