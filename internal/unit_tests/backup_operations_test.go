package unit_tests

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/neo4j/helm-charts/internal/backup"
	"github.com/stretchr/testify/assert"
)

func TestDeleteBackupFiles(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	backupDir := filepath.Join(tmpDir, "backups")
	err := os.MkdirAll(backupDir, 0755)
	assert.NoError(t, err)

	originalBackupDir := "/backups"
	os.Setenv("BACKUP_DIR", backupDir)
	defer os.Setenv("BACKUP_DIR", originalBackupDir)

	os.Setenv("KEEP_BACKUP_FILES", "false")

	testFiles := []string{
		"neo4j-2024-01-01.backup",
		"system-2024-01-01.backup",
		"neo4j-2024-01-01.backup.report.tar.gz",
	}

	for _, file := range testFiles {
		filePath := filepath.Join(backupDir, file)
		err := os.WriteFile(filePath, []byte("test"), 0644)
		assert.NoError(t, err)
	}

	err = backup.DeleteBackupFiles(testFiles, []string{})
	assert.NoError(t, err)

	files, err := os.ReadDir(backupDir)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(files), "Backup directory should be empty")

	os.Setenv("KEEP_BACKUP_FILES", "true")

	for _, file := range testFiles {
		filePath := filepath.Join(backupDir, file)
		err := os.WriteFile(filePath, []byte("test"), 0644)
		assert.NoError(t, err)
	}

	err = backup.DeleteBackupFiles(testFiles, []string{})
	assert.NoError(t, err)

	files, err = os.ReadDir(backupDir)
	assert.NoError(t, err)
	assert.Equal(t, len(testFiles), len(files), "Files should not be deleted when KEEP_BACKUP_FILES=true")
}
