package aws

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestCheckBucketAccessForAWS(t *testing.T) {
	t.Parallel()
	client, err := NewAwsClient(os.Getenv("AWS_CREDENTIAL_PATH"))
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
		{
			name:       "invalid subdirectory",
			wantErr:    true,
			bucketName: "helm-backup-test/invalid",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := client.CheckBucketAccess(tt.bucketName); (err != nil) != tt.wantErr {
				t.Errorf("CheckBucketAccess() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUploadFileForAWS(t *testing.T) {
	t.Parallel()
	client, err := NewAwsClient(os.Getenv("AWS_CREDENTIAL_PATH"))
	assert.NoError(t, err)

	currentDirectory, err := os.Getwd()
	assert.NoError(t, err)
	os.Setenv("LOCATION", fmt.Sprintf("%s/../testData", currentDirectory))

	tests := []struct {
		name       string
		wantErr    bool
		bucketName string
		fileName   string
	}{
		{
			name:       "single file upload",
			wantErr:    false,
			bucketName: "helm-backup-test",
			fileName:   "test.yaml",
		},
		{
			name:       "multiple files upload to a subdirectory",
			wantErr:    false,
			bucketName: "helm-backup-test/test",
			fileName:   "test.yaml",
		},
		{
			name:       "multiple files upload to a nested subdirectory",
			wantErr:    false,
			bucketName: "helm-backup-test/test/test2",
			fileName:   "test2.yaml",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := client.UploadFile(tt.fileName, os.Getenv("LOCATION"), tt.bucketName); (err != nil) != tt.wantErr {
				t.Errorf("UploadFile() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
