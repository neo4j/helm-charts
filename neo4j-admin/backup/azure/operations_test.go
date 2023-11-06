package azure

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestCheckContainerAccessForAzure(t *testing.T) {
	t.Parallel()
	client, err := NewAzureClient(os.Getenv("AZURE_CREDENTIAL_PATH"))
	assert.NoError(t, err)

	tests := []struct {
		name       string
		wantErr    bool
		bucketName string
	}{
		{
			name:       "valid bucket",
			wantErr:    false,
			bucketName: "helm-backup-test",
		},
		{
			name:       "valid bucket with subdirectory",
			wantErr:    false,
			bucketName: "helm-backup-test/test",
		},
		{
			name:       "valid bucket with subdirectories",
			wantErr:    false,
			bucketName: "helm-backup-test/test/test2",
		},
		{
			name:       "invalid bucket",
			wantErr:    true,
			bucketName: "does-not-exist-bucket",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := client.CheckContainerAccess(tt.bucketName); (err != nil) != tt.wantErr {
				t.Errorf("CheckContainerAccess() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUploadFileForAzure(t *testing.T) {
	t.Parallel()
	client, err := NewAzureClient(os.Getenv("AZURE_CREDENTIAL_PATH"))
	assert.NoError(t, err)

	currentDirectory, err := os.Getwd()
	assert.NoError(t, err)
	os.Setenv("LOCATION", fmt.Sprintf("%s/../testData", currentDirectory))

	tests := []struct {
		name       string
		wantErr    bool
		bucketName string
		fileNames  []string
	}{
		{
			name:       "single file upload",
			wantErr:    false,
			bucketName: "helm-backup-test",
			fileNames:  []string{"test.yaml"},
		},
		{
			name:       "multiple files upload to a subdirectory",
			wantErr:    false,
			bucketName: "helm-backup-test/test",
			fileNames:  []string{"test.yaml", "test2.yaml"},
		},
		{
			name:       "multiple files upload to a nested subdirectory",
			wantErr:    false,
			bucketName: "helm-backup-test/test/test2",
			fileNames:  []string{"test.yaml", "test2.yaml"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := client.UploadFile(tt.fileNames, tt.bucketName); (err != nil) != tt.wantErr {
				t.Errorf("UploadFile() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
