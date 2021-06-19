package internal

import (
	"github.com/hashicorp/go-multierror"
	"testing"
)

func ResourcesCleanup(t *testing.T, releaseName *ReleaseName) error {
	//patch := exec.Command("kubectl", "patch", "pv", "neo4j-data-disk", "-p", "'{\"spec\":{\"persistentVolumeReclaimPolicy\":\"Retain\"}}'")
	//t.Log("patch:", patch)
	var errors *multierror.Error

	err := run(t, "helm", "uninstall", string(*releaseName), "--namespace", string(releaseName.namespace()))
	if err != nil {
		errors = multierror.Append(errors, err)
		t.Log("Helm Cleanup failed:", err)
	}

	err = run(t, "kubectl", "delete", "namespace", string(releaseName.namespace()), "--ignore-not-found")
	if err != nil {
		errors = multierror.Append(errors, err)
		t.Log("Namespace Cleanup failed:", err)
	}

	err = run(t, "helm", "uninstall", string(*releaseName) + "-pv")
	if err != nil {
		errors = multierror.Append(errors, err)
		t.Log("Remove PV failed:", err)
	}

	return errors.ErrorOrNil()
}

func ResourcesReinstall(t *testing.T, releaseName *ReleaseName) error {
	_, err := createNamespace(t, releaseName)
	if err != nil {
		t.Log("Creating namespace failed:", err)
		return err
	}

	createSecretCommands, cleanupCertificates, err := kCreateSecret(releaseName.namespace())
	defer cleanupCertificates()
	if err != nil {
		t.Log("Creating certificates failed:", err)
		return err
	}

	err = runAll(t, "kubectl", createSecretCommands, true)
	if err != nil {
		t.Log("Re-creating secrets failed:", err)
		return err
	}
	err = runAll(t, "helm", helmInstallCommands(releaseName, releaseName.diskName()), true)
	if err != nil {
		t.Log("Helm Install failed:", err)
		return err
	}
	err = run(t, "kubectl", "--namespace", string(releaseName.namespace()), "rollout", "status", "--watch", "--timeout=120s", "statefulset/" + string(*releaseName))
	if err != nil {
		t.Log("Helm Install failed:", err)
		return err
	}
	return err
}
