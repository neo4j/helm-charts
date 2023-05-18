package aws

import (
	"cloud.google.com/go/storage"
	"context"
	"fmt"
	"google.golang.org/api/option"
)

type gcpClient struct {
	storageClient *storage.Client
}

func NewGCPClient(credentialPath string) (*gcpClient, error) {
	ctx := context.Background()
	client, err := storage.NewClient(ctx, option.WithCredentialsFile(credentialPath))
	if err != nil {
		return nil, fmt.Errorf("Unable to create gcs storage client. Here's why: %v", err)
	}

	return &gcpClient{
		storageClient: client,
	}, nil
}
