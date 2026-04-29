package integration_tests

import (
	"context"
	"testing"
	"time"

	"github.com/neo4j/helm-charts/internal/testutil/poll"
)

func waitForClusterConnection(t *testing.T) error {
	return poll.Until(context.Background(), t, poll.Opts{
		Interval:      10 * time.Second,
		Timeout:       35 * time.Second,
		Description:   "kubectl cluster-info to succeed",
		RetryableErrs: func(error) bool { return true },
	}, func(context.Context) (bool, error) {
		err := run(t, "kubectl", "cluster-info")
		return err == nil, err
	})
}
