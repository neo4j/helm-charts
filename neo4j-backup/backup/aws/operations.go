package aws

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"log"
	"os"
)

// CheckBucketAccess checks if the given bucket name is accessible or not
func (a *awsClient) CheckBucketAccess(bucketName string) error {

	// Create an Amazon S3 service client
	client := s3.NewFromConfig(*a.cfg)

	// Get the first page of results for ListObjectsV2 for a bucket
	_, err := client.ListObjectsV2(context.TODO(), &s3.ListObjectsV2Input{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		return fmt.Errorf("Unable to connect to s3 bucket %s \n Here's why: %v\n", bucketName, err)
	}
	log.Printf("Connectivity with S3 Bucket '%s' established", bucketName)
	return nil
}

// UploadFile uploads the file present at the provided location to the s3 bucket
func (a *awsClient) UploadFile(fileName string, location string, bucketName string) error {

	filePath := fmt.Sprintf("%s/%s", location, fileName)
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("Couldn't open file %v to upload. Here's why: %v\n", filePath, err)
	}
	defer file.Close()

	s3Client := s3.NewFromConfig(*a.cfg)
	_, err = s3Client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(fileName),
		Body:   file,
	})
	if err != nil {
		return fmt.Errorf("Couldn't upload file %v to %v:%v. Here's why: %v\n", filePath, bucketName, fileName, err)
	}
	log.Printf("File %s uploaded to s3 bucket %s !!", fileName, bucketName)

	return nil
}