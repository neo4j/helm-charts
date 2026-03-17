package gcloud

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"testing"
	"time"

	. "github.com/neo4j/helm-charts/internal/helpers"
	"github.com/neo4j/helm-charts/internal/model"
)

func InstallGcloud(t *testing.T, zone Zone, project Project, releaseName model.ReleaseName) (Closeable, *model.PersistentDiskName, error) {

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

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, command, args...)
	out, err := cmd.CombinedOutput()

	if ctx.Err() == context.DeadlineExceeded {
		t.Logf("Command timed out after 5 minutes: %s %s", command, args)
		return fmt.Errorf("command timed out after 5 minutes: %s %v", command, args)
	}

	if out != nil {
		t.Logf("output: %s\n", out)
	}
	if err != nil {
		return fmt.Errorf("%w: %s", err, string(out))
	}
	return nil
}

func createDisk(t *testing.T, zone Zone, project Project, releaseName model.ReleaseName) (model.PersistentDiskName, Closeable, error) {
	diskName := releaseName.DiskName()
	err := run(t, "gcloud", "compute", "disks", "create", "--size", model.StorageSize, "--type", "pd-ssd", string(diskName), "--zone="+string(zone), "--project="+string(project))
	return diskName, func() error { return deleteDisk(t, zone, project, string(diskName)) }, err
}

func deleteDisk(t *testing.T, zone Zone, project Project, diskName string) error {
	deleteFn := func() error {
		return run(t, "gcloud", "compute", "disks", "delete", diskName, "--quiet", "--zone="+string(zone), "--project="+string(project))
	}
	err := deleteFn()
	if err != nil {
		deadline := time.Now().Add(1 * time.Minute)
		for time.Now().Before(deadline) {
			time.Sleep(5 * time.Second)
			if err = deleteFn(); err == nil {
				return nil
			}
			if strings.Contains(err.Error(), "was not found") {
				return nil
			}
		}
	}
	return err
}
