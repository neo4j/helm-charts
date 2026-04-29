package integration_tests

import (
	"fmt"
	"testing"
	"time"
)

func waitForClusterConnection(t *testing.T) error {
	maxRetries := 3
	for i := 0; i < maxRetries; i++ {
		err := run(t, "kubectl", "cluster-info")
		if err == nil {
			return nil
		}
		time.Sleep(10 * time.Second)
	}
	return fmt.Errorf("failed to connect to cluster after %d attempts", maxRetries)
}
