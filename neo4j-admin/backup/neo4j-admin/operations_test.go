package neo4j_admin

import (
	"bytes"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// mockNeo4jAdmin creates a mock neo4j-admin script that simulates a long-running backup
func createMockNeo4jAdmin(t *testing.T) string {
	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "neo4j-admin-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Create a mock neo4j-admin script
	scriptPath := fmt.Sprintf("%s/neo4j-admin", tmpDir)
	script := `#!/bin/bash
if [ "$1" = "database" ] && [ "$2" = "backup" ]; then
    echo "Starting backup operation..."
    for i in {1..5}; do
        echo "Backup progress: $i/5"
        sleep 0.5
    done
    echo "Finished artifact creation 'neo4j-2024-03-19T12-00-00.backup' for database 'neo4j', took 5s."
    exit 0
fi
exit 1
`
	err = os.WriteFile(scriptPath, []byte(script), 0755)
	if err != nil {
		t.Fatalf("Failed to write mock script: %v", err)
	}

	return scriptPath
}

func TestBackupLogStreaming(t *testing.T) {
	// Create mock neo4j-admin
	mockScript := createMockNeo4jAdmin(t)
	defer os.RemoveAll(strings.TrimSuffix(mockScript, "/neo4j-admin"))

	// Set up test environment
	os.Setenv("DATABASE", "neo4j")
	os.Setenv("INCLUDE_METADATA", "true")
	os.Setenv("KEEP_FAILED", "true")
	os.Setenv("PARALLEL_RECOVERY", "true")
	os.Setenv("TYPE", "full")
	os.Setenv("PAGE_CACHE", "100M")
	os.Setenv("VERBOSE", "true")

	// Capture log output with microsecond precision
	var logBuffer bytes.Buffer
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	log.SetOutput(&logBuffer)
	defer func() {
		log.SetFlags(log.LstdFlags)
		log.SetOutput(os.Stdout)
	}()

	// Record start time
	startTime := time.Now()

	// Replace neo4j-admin path temporarily
	origPath := os.Getenv("PATH")
	os.Setenv("PATH", fmt.Sprintf("%s:%s", strings.TrimSuffix(mockScript, "/neo4j-admin"), origPath))
	defer os.Setenv("PATH", origPath)

	// Perform backup
	backupFiles, err := PerformBackup("localhost:6362")
	assert.NoError(t, err)
	assert.Len(t, backupFiles, 1)
	assert.Equal(t, "neo4j-2024-03-19T12-00-00.backup", backupFiles[0])

	// Record end time
	endTime := time.Now()

	// Get captured logs
	logs := logBuffer.String()

	// Verify that logs contain progress messages in order
	assert.Contains(t, logs, "Starting backup operation...")

	// Split logs into lines and verify progress messages appear in order
	logLines := strings.Split(logs, "\n")
	var progressLines []string
	for _, line := range logLines {
		if strings.Contains(line, "Backup progress:") {
			progressLines = append(progressLines, line)
		}
	}

	// Verify we got all 5 progress messages
	assert.Len(t, progressLines, 5, "Should have exactly 5 progress messages")

	// Verify progress messages are in order
	for i := 0; i < 5; i++ {
		expectedMsg := fmt.Sprintf("Backup progress: %d/5", i+1)
		assert.Contains(t, progressLines[i], expectedMsg, "Progress messages should be in order")
	}

	assert.Contains(t, logs, "Finished artifact creation")

	// Verify that timestamps are monotonically increasing
	var timestamps []time.Time
	for _, line := range progressLines {
		// Parse timestamp with microsecond precision
		if ts, err := time.Parse("2006/01/02 15:04:05.000000", line[:26]); err == nil {
			timestamps = append(timestamps, ts)
		} else {
			t.Logf("Failed to parse timestamp from line: %s, error: %v", line, err)
		}
	}

	assert.Len(t, timestamps, 5, "Should have parsed 5 timestamps")

	// Verify timestamps are in order and have some delay between them
	for i := 1; i < len(timestamps); i++ {
		diff := timestamps[i].Sub(timestamps[i-1])
		assert.Greater(t, diff.Microseconds(), int64(0),
			"Each log entry should be after the previous one (line %d timestamp: %v, line %d timestamp: %v)",
			i-1, timestamps[i-1], i, timestamps[i])
	}

	// Verify total operation took some time (at least 2 seconds)
	totalTime := endTime.Sub(startTime).Seconds()
	assert.Greater(t, totalTime, 2.0, "Operation should take at least 2 seconds")
}

func TestAggregateBackupLogStreaming(t *testing.T) {
	// Create a mock neo4j-admin script for aggregate backup
	tmpDir, err := os.MkdirTemp("", "neo4j-admin-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create backup directory with mock backup files
	backupDir := fmt.Sprintf("%s/backups", tmpDir)
	err = os.MkdirAll(backupDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create backup dir: %v", err)
	}

	// Create at least 2 backup files so aggregate is triggered (count > 1)
	mockBackups := []string{
		"neo4j-2024-01-01T10-00-00.backup",
		"neo4j-2024-01-02T10-00-00.backup",
	}
	for _, backup := range mockBackups {
		filePath := fmt.Sprintf("%s/%s", backupDir, backup)
		err := os.WriteFile(filePath, []byte("mock backup data"), 0644)
		if err != nil {
			t.Fatalf("Failed to create mock backup file %s: %v", backup, err)
		}
	}

	scriptPath := fmt.Sprintf("%s/neo4j-admin", tmpDir)
	script := `#!/bin/bash
if [ "$1" = "backup" ] && [ "$2" = "aggregate" ]; then
    echo "Starting aggregate backup operation..."
    for i in {1..3}; do
        echo "Aggregating backup files: $i/3"
        sleep 1
    done
    echo "Successfully aggregated backup chain of database 'neo4j', new artifact: '/backups/neo4j-2024-03-19T12-00-00.backup'."
    exit 0
fi
exit 1
`
	err = os.WriteFile(scriptPath, []byte(script), 0755)
	if err != nil {
		t.Fatalf("Failed to write mock script: %v", err)
	}

	// Set up test environment
	os.Setenv("AGGREGATE_BACKUP_DATABASE", "neo4j")
	os.Setenv("AGGREGATE_BACKUP_FROM_PATH", "/backups")
	os.Setenv("AGGREGATE_BACKUP_KEEPOLDBACKUP", "true")
	os.Setenv("AGGREGATE_BACKUP_PARALLEL_RECOVERY", "true")
	os.Setenv("VERBOSE", "true")

	// Capture log output
	var logBuffer bytes.Buffer
	log.SetOutput(&logBuffer)
	defer log.SetOutput(os.Stdout)

	// Record start time
	startTime := time.Now()

	// Replace neo4j-admin path temporarily
	origPath := os.Getenv("PATH")
	os.Setenv("PATH", fmt.Sprintf("%s:%s", tmpDir, origPath))
	defer os.Setenv("PATH", origPath)

	// Perform aggregate backup
	err = PerformAggregateBackup()
	assert.NoError(t, err)

	// Record end time
	endTime := time.Now()

	// Get captured logs
	logs := logBuffer.String()

	// Verify that logs contain progress messages
	assert.Contains(t, logs, "Starting aggregate backup operation...")
	for i := 1; i <= 3; i++ {
		assert.Contains(t, logs, fmt.Sprintf("Aggregating backup files: %d/3", i))
	}
	assert.Contains(t, logs, "Successfully aggregated backup chain")

	// Verify that the operation took at least 3 seconds (our mock sleeps for 3 seconds total)
	assert.GreaterOrEqual(t, endTime.Sub(startTime).Seconds(), 3.0)

	// Split logs by newline and verify timestamps are properly spread out
	logLines := strings.Split(logs, "\n")
	var timestamps []time.Time
	for _, line := range logLines {
		if strings.Contains(line, "Aggregating backup files:") {
			// Parse timestamp from log line (assuming default log format)
			if ts, err := time.Parse("2006/01/02 15:04:05", line[:19]); err == nil {
				timestamps = append(timestamps, ts)
			}
		}
	}

		// Verify that timestamps are spread out by roughly 1 second each
		for i := 1; i < len(timestamps); i++ {
			diff := timestamps[i].Sub(timestamps[i-1]).Seconds()
			assert.InDelta(t, 1.0, diff, 0.5) // Allow 0.5s variance for system delays
		}
}

func TestGetCloudStoragePath(t *testing.T) {
	tests := []struct {
		name           string
		cloudProvider  string
		bucketName     string
		storageAccount string
		expectedPath   string
	}{
		// AWS S3 tests
		{
			name:          "AWS S3 - simple bucket name",
			cloudProvider: "aws",
			bucketName:    "my-bucket",
			expectedPath:  "s3://my-bucket/",
		},
		{
			name:          "AWS S3 - bucket with path prefix",
			cloudProvider: "aws",
			bucketName:    "my-bucket/backups",
			expectedPath:  "s3://my-bucket/backups/",
		},
		{
			name:          "AWS S3 - bucket with nested path prefix",
			cloudProvider: "aws",
			bucketName:    "my-bucket/folder1/folder2",
			expectedPath:  "s3://my-bucket/folder1/folder2/",
		},
		{
			name:          "AWS S3 - bucket with leading slash",
			cloudProvider: "aws",
			bucketName:    "/my-bucket/backups",
			expectedPath:  "s3://my-bucket/backups/",
		},
		{
			name:          "AWS S3 - bucket with trailing slash",
			cloudProvider: "aws",
			bucketName:    "my-bucket/backups/",
			expectedPath:  "s3://my-bucket/backups/",
		},
		{
			name:          "AWS S3 - bucket with both leading and trailing slashes",
			cloudProvider: "aws",
			bucketName:    "/my-bucket/backups/",
			expectedPath:  "s3://my-bucket/backups/",
		},
		// GCP GCS tests
		{
			name:          "GCP GCS - simple bucket name",
			cloudProvider: "gcp",
			bucketName:    "my-bucket",
			expectedPath:  "gs://my-bucket/",
		},
		{
			name:          "GCP GCS - bucket with path prefix",
			cloudProvider: "gcp",
			bucketName:    "my-bucket/backups",
			expectedPath:  "gs://my-bucket/backups/",
		},
		{
			name:          "GCP GCS - bucket with nested path prefix",
			cloudProvider: "gcp",
			bucketName:    "my-bucket/folder1/folder2",
			expectedPath:  "gs://my-bucket/folder1/folder2/",
		},
		// Azure Blob Storage tests - without storage account
		{
			name:          "Azure - simple container name without storage account",
			cloudProvider: "azure",
			bucketName:    "my-container",
			expectedPath:  "azb://my-container/",
		},
		{
			name:          "Azure - container with path prefix without storage account",
			cloudProvider: "azure",
			bucketName:    "my-container/backups",
			expectedPath:  "azb://my-container/backups/",
		},
		// Azure Blob Storage tests - with storage account
		{
			name:           "Azure - simple container name with storage account",
			cloudProvider:  "azure",
			bucketName:     "my-container",
			storageAccount: "mystorageaccount",
			expectedPath:   "azb://mystorageaccount/my-container/",
		},
		{
			name:           "Azure - container with path prefix with storage account",
			cloudProvider:  "azure",
			bucketName:     "my-container/backups",
			storageAccount: "mystorageaccount",
			expectedPath:   "azb://mystorageaccount/my-container/backups/",
		},
		{
			name:           "Azure - container with nested path prefix with storage account",
			cloudProvider:  "azure",
			bucketName:     "my-container/folder1/folder2",
			storageAccount: "mystorageaccount",
			expectedPath:   "azb://mystorageaccount/my-container/folder1/folder2/",
		},
		// Edge cases
		{
			name:          "Empty cloud provider - should fallback to local path",
			cloudProvider: "",
			bucketName:    "my-bucket",
			expectedPath:  "/backups", // Default backup path
		},
		{
			name:          "Unknown cloud provider - should fallback to local path",
			cloudProvider: "unknown",
			bucketName:    "my-bucket",
			expectedPath:  "/backups", // Default backup path
		},
		{
			name:          "Empty bucket name",
			cloudProvider: "aws",
			bucketName:    "",
			expectedPath:  "s3:///",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original environment variables
			origCloudProvider := os.Getenv("CLOUD_PROVIDER")
			origBucketName := os.Getenv("BUCKET_NAME")
			origStorageAccount := os.Getenv("AZURE_STORAGE_ACCOUNT")
			origBackupPath := os.Getenv("BACKUP_PATH")

			// Clean up environment variables
			defer func() {
				if origCloudProvider != "" {
					os.Setenv("CLOUD_PROVIDER", origCloudProvider)
				} else {
					os.Unsetenv("CLOUD_PROVIDER")
				}
				if origBucketName != "" {
					os.Setenv("BUCKET_NAME", origBucketName)
				} else {
					os.Unsetenv("BUCKET_NAME")
				}
				if origStorageAccount != "" {
					os.Setenv("AZURE_STORAGE_ACCOUNT", origStorageAccount)
				} else {
					os.Unsetenv("AZURE_STORAGE_ACCOUNT")
				}
				if origBackupPath != "" {
					os.Setenv("BACKUP_PATH", origBackupPath)
				} else {
					os.Unsetenv("BACKUP_PATH")
				}
			}()

			// Set test environment variables
			if tt.cloudProvider != "" {
				os.Setenv("CLOUD_PROVIDER", tt.cloudProvider)
			} else {
				os.Unsetenv("CLOUD_PROVIDER")
			}
			os.Setenv("BUCKET_NAME", tt.bucketName)
			if tt.storageAccount != "" {
				os.Setenv("AZURE_STORAGE_ACCOUNT", tt.storageAccount)
			} else {
				os.Unsetenv("AZURE_STORAGE_ACCOUNT")
			}
			os.Unsetenv("BACKUP_PATH") // Use default backup path

			// Call the function
			result := getCloudStoragePath()

			// Assert the result
			assert.Equal(t, tt.expectedPath, result, "getCloudStoragePath() should return the expected path")
		})
	}
}

// TestGetCloudStoragePathBackwardCompatibility ensures that existing behavior is preserved
func TestGetCloudStoragePathBackwardCompatibility(t *testing.T) {
	tests := []struct {
		name          string
		cloudProvider string
		bucketName    string
		expectedPath  string
		description   string
	}{
		{
			name:          "AWS S3 - backward compatible simple bucket",
			cloudProvider: "aws",
			bucketName:    "my-bucket",
			expectedPath:  "s3://my-bucket/",
			description:   "Simple bucket names should work as before",
		},
		{
			name:          "GCP GCS - backward compatible simple bucket",
			cloudProvider: "gcp",
			bucketName:    "my-bucket",
			expectedPath:  "gs://my-bucket/",
			description:   "Simple bucket names should work as before",
		},
		{
			name:          "Azure - backward compatible simple container",
			cloudProvider: "azure",
			bucketName:    "my-container",
			expectedPath:  "azb://my-container/",
			description:   "Simple container names should work as before",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original environment variables
			origCloudProvider := os.Getenv("CLOUD_PROVIDER")
			origBucketName := os.Getenv("BUCKET_NAME")
			origStorageAccount := os.Getenv("AZURE_STORAGE_ACCOUNT")

			// Clean up environment variables
			defer func() {
				if origCloudProvider != "" {
					os.Setenv("CLOUD_PROVIDER", origCloudProvider)
				} else {
					os.Unsetenv("CLOUD_PROVIDER")
				}
				if origBucketName != "" {
					os.Setenv("BUCKET_NAME", origBucketName)
				} else {
					os.Unsetenv("BUCKET_NAME")
				}
				if origStorageAccount != "" {
					os.Setenv("AZURE_STORAGE_ACCOUNT", origStorageAccount)
				} else {
					os.Unsetenv("AZURE_STORAGE_ACCOUNT")
				}
			}()

			// Set test environment variables
			os.Setenv("CLOUD_PROVIDER", tt.cloudProvider)
			os.Setenv("BUCKET_NAME", tt.bucketName)
			os.Unsetenv("AZURE_STORAGE_ACCOUNT")
			os.Unsetenv("BACKUP_PATH")

			// Call the function
			result := getCloudStoragePath()

			// Assert the result matches expected backward-compatible behavior
			assert.Equal(t, tt.expectedPath, result, tt.description)
		})
	}
}

// TestGetCloudStoragePathCustomerScenario tests the exact customer scenario
func TestGetCloudStoragePathCustomerScenario(t *testing.T) {
	// Save original environment variables
	origCloudProvider := os.Getenv("CLOUD_PROVIDER")
	origBucketName := os.Getenv("BUCKET_NAME")

	// Clean up environment variables
	defer func() {
		if origCloudProvider != "" {
			os.Setenv("CLOUD_PROVIDER", origCloudProvider)
		} else {
			os.Unsetenv("CLOUD_PROVIDER")
		}
		if origBucketName != "" {
			os.Setenv("BUCKET_NAME", origBucketName)
		} else {
			os.Unsetenv("BUCKET_NAME")
		}
	}()

	// Simulate customer scenario: BUCKET_NAME/BACKUP_FOLDER
	os.Setenv("CLOUD_PROVIDER", "aws")
	os.Setenv("BUCKET_NAME", "BUCKET_NAME/BACKUP_FOLDER")

	result := getCloudStoragePath()

	// Should parse correctly: bucket="BUCKET_NAME", pathPrefix="BACKUP_FOLDER"
	expected := "s3://BUCKET_NAME/BACKUP_FOLDER/"
	assert.Equal(t, expected, result, "Customer scenario should parse bucket name and path prefix correctly")

	// Verify that this path can be parsed back correctly by URL parser
	// (simulating what getBackupCount does)
	parsedURL, err := url.Parse(result)
	assert.NoError(t, err, "Resulting path should be a valid URL")
	assert.Equal(t, "s3", parsedURL.Scheme, "URL scheme should be s3")
	assert.Equal(t, "BUCKET_NAME", parsedURL.Host, "URL host should be the bucket name")
	assert.Equal(t, "/BACKUP_FOLDER/", parsedURL.Path, "URL path should be the path prefix")

	// Verify pathPrefix extraction for getBackupCount/countS3Backups flow
	// pathPrefix must not have trailing slash to avoid "backups//neo4j-" prefix
	pathPrefix := strings.Trim(strings.TrimPrefix(parsedURL.Path, "/"), "/")
	assert.Equal(t, "BACKUP_FOLDER", pathPrefix, "Path prefix should be trimmed for S3 ListObjectsV2 prefix")
	assert.NotContains(t, pathPrefix, "//", "Path prefix should not produce double slashes when combined with db name")
}
