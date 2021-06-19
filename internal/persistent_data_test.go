package internal

import (
	"fmt"
	"github.com/hashicorp/go-multierror"
)

func ResourcesCleanup(releaseName *ReleaseName) error {
	//patch := exec.Command("kubectl", "patch", "pv", "neo4j-data-disk", "-p", "'{\"spec\":{\"persistentVolumeReclaimPolicy\":\"Retain\"}}'")
	//fmt.Println("patch:", patch)
	var errors *multierror.Error

	err := run("helm", "uninstall", string(*releaseName), "--namespace", string(releaseName.namespace()))
	if err != nil {
		errors = multierror.Append(errors, err)
		fmt.Println("Helm Cleanup failed:", err)
	}

	err = run("kubectl", "delete", "namespace", string(releaseName.namespace()), "--ignore-not-found")
	if err != nil {
		errors = multierror.Append(errors, err)
		fmt.Println("Namespace Cleanup failed:", err)
	}

	err = run("helm", "uninstall", string(*releaseName) + "-pv")
	if err != nil {
		errors = multierror.Append(errors, err)
		fmt.Println("Remove PV failed:", err)
	}

	return errors.ErrorOrNil()
}

func ResourcesReinstall(releaseName *ReleaseName) error {
	_, err := createNamespace(releaseName)
	if err != nil {
		fmt.Println("Creating namespace failed:", err)
		return err
	}

	createSecretCommands, cleanupCertificates, err := kCreateSecret(releaseName.namespace())
	defer cleanupCertificates()
	if err != nil {
		fmt.Println("Creating certificates failed:", err)
		return err
	}

	err = runAll("kubectl", createSecretCommands, true)
	if err != nil {
		fmt.Println("Re-creating secrets failed:", err)
		return err
	}
	err = runAll("helm", helmInstallCommands(releaseName, releaseName.diskName()), true)
	if err != nil {
		fmt.Println("Helm Install failed:", err)
		return err
	}
	err = run("kubectl", "--namespace", string(releaseName.namespace()), "rollout", "status", "--watch", "--timeout=120s", "statefulset/" + string(*releaseName))
	if err != nil {
		fmt.Println("Helm Install failed:", err)
		return err
	}
	return err
}
