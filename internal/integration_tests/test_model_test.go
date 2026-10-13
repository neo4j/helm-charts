package integration_tests

import (
	"strings"
	"testing"

	"github.com/neo4j/helm-charts/internal/model"
	"github.com/stretchr/testify/require"
)

func TestBackupDestinationIsolation(t *testing.T) {
	var destinations []string
	for _, name := range []string{"initial and aggregate backup", "other backup"} {
		t.Run(name, func(t *testing.T) {
			destination := TestBackupDestination(t)
			bucket, prefix, found := strings.Cut(destination, "/")
			require.True(t, found)
			require.Equal(t, model.BucketName, bucket)
			require.NotEmpty(t, prefix)
			require.Contains(t, prefix, TestRunIdentifier)
			require.Equal(t, destination, TestBackupDestination(t))
			destinations = append(destinations, destination)
		})
	}
	require.Len(t, destinations, 2)
	require.NotEqual(t, destinations[0], destinations[1])
}
