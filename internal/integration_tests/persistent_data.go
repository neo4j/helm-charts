package integration_tests

import (
	"github.com/hashicorp/go-multierror"
	"github.com/neo4j/helm-charts/internal/model"
	"testing"
)

func ResourcesCleanup(t *testing.T, releaseName model.ReleaseName) error {
	//patch := exec.Command("kubectl", "patch", "pv", "neo4j-data-disk", "-p", "'{\"spec\":{\"persistentVolumeReclaimPolicy\":\"Retain\"}}'")
	//t.Log("patch:", patch)
	var errors *multierror.Error

	err := run(t, "kubectl", "delete", "--all", "statefulsets", "--namespace", string(releaseName.Namespace()), "--force", "--grace-period=0", "--wait=false")
	if err != nil {
		errors = multierror.Append(errors, err)
		t.Log("StatefulSet delete failed:", err)
	}

	err = run(t, "kubectl", "delete", "--all", "pods", "--namespace", string(releaseName.Namespace()), "--force", "--grace-period=0", "--wait=false")
	if err != nil {
		errors = multierror.Append(errors, err)
		t.Log("Pod delete failed:", err)
	}

	// This value is set manually
	tasksToRun := 2
	// semaphore
	sem := make(chan error, tasksToRun)

	go func() {
		err = run(t, "helm", "uninstall", releaseName.String(), "--namespace", string(releaseName.Namespace()))
		if err != nil {
			t.Log("Helm Cleanup failed:", err)
		}
		sem <- err

		err = run(t, "kubectl", "delete", "namespace", string(releaseName.Namespace()), "--ignore-not-found")
		if err != nil {
			t.Log("Namespace Cleanup failed:", err)
		}
		sem <- err
	}()

	for i := 0; i < tasksToRun; i++ {
		errInGoRoutine := <-sem
		if errInGoRoutine != nil {
			errors = multierror.Append(errors, err)
		}
	}

	return errors.ErrorOrNil()
}

func ResourcesReinstall(t *testing.T, releaseName model.ReleaseName, chart model.Neo4jHelmChart) error {
	_, err := createNamespace(t, releaseName)
	if err != nil {
		t.Log("Creating namespace failed:", err)
		return err
	}

	createSecretCommands, cleanupCertificates, err := kCreateSecret(releaseName.Namespace())
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

	diskName := releaseName.DiskName()
	err = run(t, "helm", model.BaseHelmCommand("install", releaseName, chart, model.Neo4jEdition, &diskName, "--wait", "--timeout", "300s")...)
	if err != nil {
		t.Log("Helm Install failed:", err)
		return err
	}
	err = run(t, "kubectl", "--namespace", string(releaseName.Namespace()), "rollout", "status", "--watch", "--timeout=120s", "statefulset/"+releaseName.String())
	if err != nil {
		t.Log("Helm Install failed:", err)
		return err
	}
	return err
}
