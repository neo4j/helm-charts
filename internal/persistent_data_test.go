package internal

import (
	"fmt"
	"github.com/hashicorp/go-multierror"
)

func ResourcesCleanup() error {
	//patch := exec.Command("kubectl", "patch", "pv", "neo4j-data-disk", "-p", "'{\"spec\":{\"persistentVolumeReclaimPolicy\":\"Retain\"}}'")
	//fmt.Println("patch:", patch)
	var errors *multierror.Error

	err := run("helm", "uninstall", "neo4j", "--namespace", "neo4j")
	if err != nil {
		errors = multierror.Append(errors, err)
		fmt.Println("Helm Cleanup failed:", err)
	}

	err = run("kubectl", "delete", "namespace", "neo4j", "--ignore-not-found")
	if err != nil {
		errors = multierror.Append(errors, err)
		fmt.Println("Namespace Cleanup failed:", err)
	}
	err = run("kubectl", "replace", "--force", "-f", "./yaml/neo4j-persistentvolume.yaml")
	if err != nil {
		errors = multierror.Append(errors, err)
		fmt.Println("Replace PV failed:", err)
	}
	return errors.ErrorOrNil()
}

func ResourcesReinstall() error {
	err := runAll("helm", helmInstallCommands(), true)
	if err != nil {
		fmt.Println("Helm Install failed:", err)
	}
	return err
}
