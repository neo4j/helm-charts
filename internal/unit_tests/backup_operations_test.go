package unit_tests

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/neo4j/helm-charts/internal/backup"
	"github.com/stretchr/testify/assert"
)

func TestDeleteBackupFiles(t *testing.T) {
	// Create a temporary directory and files
	tmpDir := t.TempDir()

	// Create test files
	testFiles := []string{"backup1.backup", "backup2.backup"}
	for _, file := range testFiles {
		filePath := filepath.Join(tmpDir, file)
		f, err := os.Create(filePath)
		assert.NoError(t, err)
		f.Close()
	}

	// Set environment variables
	os.Setenv("KEEP_BACKUP_FILES", "false")
	os.Setenv("BACKUP_DIR", tmpDir)

	// Test deletion
	err := backup.DeleteBackupFiles(testFiles, []string{})
	assert.NoError(t, err)

	// Verify files are deleted
	for _, file := range testFiles {
		filePath := filepath.Join(tmpDir, file)
		_, err := os.Stat(filePath)
		assert.True(t, os.IsNotExist(err), "File should be deleted")
	}

	// Clean up environment
	os.Unsetenv("KEEP_BACKUP_FILES")
	os.Unsetenv("BACKUP_DIR")
}

func TestKeepBackupFiles(t *testing.T) {
	// Create a temporary directory and files
	tmpDir := t.TempDir()

	// Create test files
	testFiles := []string{"backup1.backup", "backup2.backup"}
	for _, file := range testFiles {
		filePath := filepath.Join(tmpDir, file)
		f, err := os.Create(filePath)
		assert.NoError(t, err)
		f.Close()
	}

	// Set environment variables to keep files
	os.Setenv("KEEP_BACKUP_FILES", "true")
	os.Setenv("BACKUP_DIR", tmpDir)

	// Test keep files
	err := backup.DeleteBackupFiles(testFiles, []string{})
	assert.NoError(t, err)

	// Verify files are NOT deleted
	files, err := os.ReadDir(tmpDir)
	assert.NoError(t, err)
	assert.Equal(t, len(testFiles), len(files), "Files should not be deleted when KEEP_BACKUP_FILES=true")

	// Clean up environment
	os.Unsetenv("KEEP_BACKUP_FILES")
	os.Unsetenv("BACKUP_DIR")
}
