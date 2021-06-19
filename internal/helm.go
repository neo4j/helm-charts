package internal

import (
	"errors"
	"github.com/hashicorp/go-multierror"
	"os/exec"
)


var DefaultHelmTemplateReleaseName = ReleaseName("my-release")

func helmTemplate(helmTemplateArgs ...string) (error, *K8sResources) {
	if len(helmTemplateArgs) == 0 || helmTemplateArgs[0] != "template" {
		helmTemplateArgs = append(minHelmCommand("template", &DefaultHelmTemplateReleaseName), helmTemplateArgs...)
	}

	out, err := exec.Command("helm", helmTemplateArgs...).CombinedOutput()

	if err != nil {
		return multierror.Append(errors.New("Error running helm template"), err, errors.New(string(out))), nil
	}

	manifest, err := decodeK8s(out)
	return err, manifest
}
