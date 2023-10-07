package aws

import (
	"cloud.google.com/go/storage"
	"context"
	"fmt"
	"google.golang.org/api/option"
	"log"
)

type gcpClient struct {
	storageClient *storage.Client
}

func NewGCPClient(credentialPath string) (*gcpClient, error) {
	ctx := context.Background()
	var client *storage.Client
	var err error

	if credentialPath == "/credentials/" {
		log.Printf("Credential Path is %s", credentialPath)
		client, err = storage.NewClient(ctx)
		if err != nil {
			return nil, fmt.Errorf("Unable to create gcs storage client . Here's why: %v", err)
		}
	} else {
		client, err = storage.NewClient(ctx, option.WithCredentialsFile(credentialPath))
		if err != nil {
			return nil, fmt.Errorf("Unable to create gcs storage client with credentials file. Here's why: %v", err)
		}
	}

	return &gcpClient{
		storageClient: client,
	}, nil
}
