package aws

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
)

// CheckBucketAccess checks if the given bucket name is accessible or not
func (g *gcpClient) CheckBucketAccess(bucketName string) error {

	ctx := context.Background()
	bucketAttrs, err := g.storageClient.Bucket(bucketName).Attrs(ctx)
	if err != nil {
		return fmt.Errorf("Unable to connect to GCS bucket %s \n Here's why: %v\n", bucketName, err)
	}
	if bucketAttrs.Name != bucketName {
		return fmt.Errorf("BucketName provided '%s' not matching with the name retrieved '%s'", bucketName, bucketAttrs.Name)
	}
	log.Printf("Connectivity with bucket %s established", bucketName)

	return nil
}

// UploadFile uploads the file present at the provided location to the gcs bucket
func (g *gcpClient) UploadFile(fileName string, location string, bucketName string) error {

	filePath := fmt.Sprintf("%s/%s", location, fileName)
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("Couldn't open file %v to upload. Here's why: %v\n", filePath, err)
	}
	defer file.Close()

	log.Printf("Starting upload of file %s", filePath)
	// create a new object handle
	object := g.storageClient.Bucket(bucketName).Object(fileName)

	// create a new writer for the object
	writer := object.NewWriter(context.Background())

	// copy the file contents to the object writer
	if _, err = io.Copy(writer, file); err != nil {
		return fmt.Errorf("Error writing file to gcs bucket %s\n Here's why: %v", bucketName, err)
	}

	// close the object writer
	if err := writer.Close(); err != nil {
		return fmt.Errorf("Error closing writer while uploading file %s to gcs bucket %s \n Here's why: %v", fileName, bucketName, err)
	}
	log.Printf("File %s uploaded to GCS bucket %s !!", fileName, bucketName)
	return nil
}
