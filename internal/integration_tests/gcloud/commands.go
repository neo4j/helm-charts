package gcloud

import (
	. "github.com/neo4j/helm-charts/internal/helpers"
	"github.com/neo4j/helm-charts/internal/model"
	"os/exec"
	"testing"
)

func InstallGcloud(t *testing.T, zone Zone, project Project, releaseName *model.ReleaseName) (Closeable, *model.PersistentDiskName, error) {

	err := run(t, "gcloud", "container", "clusters", "get-credentials", string(CurrentCluster()))
	if err != nil {
		return nil, nil, err
	}

	diskName, cleanupDisk, err := createDisk(t, zone, project, releaseName)
	if err != nil {
		return cleanupDisk, nil, err
	}

	return cleanupDisk, &diskName, err
}

func run(t *testing.T, command string, args ...string) error {
	t.Logf("running: %s %s\n", command, args)
	out, err := exec.Command(command, args...).CombinedOutput()
	if out != nil {
		t.Logf("output: %s\n", out)
	}
	return err
}

func createDisk(t *testing.T, zone Zone, project Project, releaseName *model.ReleaseName) (model.PersistentDiskName, Closeable, error) {
	diskName := releaseName.DiskName()
	err := run(t, "gcloud", "compute", "disks", "create", "--size", model.StorageSize, "--type", "pd-ssd", string(diskName), "--zone="+string(zone), "--project="+string(project))
	return model.PersistentDiskName(diskName), func() error { return deleteDisk(t, zone, project, string(diskName)) }, err
}

func deleteDisk(t *testing.T, zone Zone, project Project, diskName string) error {
	return run(t, "gcloud", "compute", "disks", "delete", diskName, "--zone="+string(zone), "--project="+string(project))
}
