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

	err = run("helm", "uninstall", "neo4j-pv")
	if err != nil {
		errors = multierror.Append(errors, err)
		fmt.Println("Remove PV failed:", err)
	}

	return errors.ErrorOrNil()
}

func ResourcesReinstall() error {
	generateCerts()
	err := runAll("kubectl", kCreateSecret, true)
	if err != nil {
		fmt.Println("Re-creating secrets failed:", err)
	}
	err = runAll("helm", helmInstallCommands(), true)
	if err != nil {
		fmt.Println("Helm Install failed:", err)
	}
	return err
}
