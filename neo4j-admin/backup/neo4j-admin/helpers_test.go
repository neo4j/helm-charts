package neo4j_admin

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetAggregateBackupCommandFlags(t *testing.T) {
	t.Parallel()

	// Set up test environment
	testCases := []struct {
		name     string
		env      map[string]string
		expected []string
	}{
		{
			name: "all flags set",
			env: map[string]string{
				"AGGREGATE_BACKUP_DATABASE":          "neo4j",
				"AGGREGATE_BACKUP_TEMP_DIR":          "/tmp/backup",
				"AGGREGATE_BACKUP_FROM_PATH":         "s3://bucket/path",
				"AGGREGATE_BACKUP_KEEPOLDBACKUP":     "true",
				"AGGREGATE_BACKUP_PARALLEL_RECOVERY": "true",
				"VERBOSE":                            "true",
			},
			expected: []string{
				"database",
				"aggregate-backup",
				"--temp-path=/tmp/backup",
				"--from-path=s3://bucket/path",
				"--keep-old-backup=true",
				"--parallel-recovery=true",
				"--verbose",
				"neo4j",
			},
		},
		{
			name: "minimal flags",
			env: map[string]string{
				"AGGREGATE_BACKUP_DATABASE":  "neo4j",
				"AGGREGATE_BACKUP_FROM_PATH": "s3://bucket/path",
			},
			expected: []string{
				"database",
				"aggregate-backup",
				"--from-path=s3://bucket/path",
				"--keep-old-backup=",
				"--parallel-recovery=",
				"neo4j",
			},
		},
		{
			name: "multiple databases",
			env: map[string]string{
				"AGGREGATE_BACKUP_DATABASE":  "neo4j,system",
				"AGGREGATE_BACKUP_FROM_PATH": "s3://bucket/path",
			},
			expected: []string{
				"database",
				"aggregate-backup",
				"--from-path=s3://bucket/path",
				"--keep-old-backup=",
				"--parallel-recovery=",
				"neo4j",
				"system",
			},
		},
	}

	for _, tc := range testCases {
		tc := tc // capture range variable
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Clear any existing env vars
			os.Unsetenv("AGGREGATE_BACKUP_DATABASE")
			os.Unsetenv("AGGREGATE_BACKUP_TEMP_DIR")
			os.Unsetenv("AGGREGATE_BACKUP_FROM_PATH")
			os.Unsetenv("AGGREGATE_BACKUP_KEEPOLDBACKUP")
			os.Unsetenv("AGGREGATE_BACKUP_PARALLEL_RECOVERY")
			os.Unsetenv("VERBOSE")

			// Set test env vars
			for k, v := range tc.env {
				os.Setenv(k, v)
			}

			// Clean up after test
			defer func() {
				for k := range tc.env {
					os.Unsetenv(k)
				}
			}()

			flags := GetAggregateBackupCommandFlags()
			assert.Equal(t, tc.expected, flags, "flags should match expected values")
		})
	}
}
