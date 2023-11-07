package aws

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/neo4j/helm-charts/neo4j-admin/backup/common"
	"log"
	"os"
	"strings"
)

// CheckBucketAccess checks if the given bucket name is accessible or not
func (a *awsClient) CheckBucketAccess(bucketName string) error {

	//Create an Amazon S3 service client
	client := s3.NewFromConfig(*a.cfg)

	s3Input := &s3.ListObjectsV2Input{
		Bucket: aws.String(bucketName),
	}
	if strings.Contains(bucketName, "/") {
		firstIndex := strings.Index(bucketName, "/")
		name := bucketName[:firstIndex]
		prefix := bucketName[firstIndex+1:]
		log.Printf("Name = %s , Prefix = %s", name, prefix)
		s3Input = &s3.ListObjectsV2Input{
			Bucket: aws.String(name),
			Prefix: aws.String(prefix),
		}
	}

	// Get the first page of results for ListObjectsV2 for a bucket
	objects, err := client.ListObjectsV2(context.TODO(), s3Input)
	if err != nil {
		return fmt.Errorf("Unable to connect to s3 bucket %s \n Here's why: %v\n", bucketName, err)
	}
	if strings.Contains(bucketName, "/") {
		if len(objects.Contents) == 0 {
			return fmt.Errorf("s3 Bucket %s does not exist", bucketName)
		}
	}
	log.Printf("Connectivity with S3 Bucket '%s' established", bucketName)
	return nil
}

// UploadFile uploads the file present at the provided location to the s3 bucket
func (a *awsClient) UploadFile(fileName string, bucketName string) error {

	location := os.Getenv("LOCATION")
	keyName := fileName
	parentBucketName := bucketName
	// if bucketName is demo/test/test2
	// parentBucketName will be "demo"
	if strings.Contains(bucketName, "/") {
		index := strings.Index(bucketName, "/")
		parentBucketName = bucketName[:index]
		keyName = fmt.Sprintf("%s/%s", bucketName[index+1:], fileName)
	}
	filePath := fmt.Sprintf("%s/%s", location, fileName)
	yes, err := common.IsFileBigger(filePath)
	if err != nil {
		return err
	}

	//use UploadLargeObject if file size is more than 4GB
	if yes {
		return a.UploadLargeObject(fileName, bucketName, parentBucketName)
	}

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("Couldn't open file %v to upload. Here's why: %v\n", filePath, err)
	}

	defer file.Close()

	s3Client := s3.NewFromConfig(*a.cfg)

	log.Printf("Starting upload of file %s", filePath)
	_, err = s3Client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(parentBucketName),
		Key:    aws.String(keyName),
		Body:   file,
	})
	if err != nil {
		return fmt.Errorf("Couldn't upload file %v to %v:%v. Here's why: %v\n", filePath, bucketName, fileName, err)
	}
	log.Printf("File %s uploaded to s3 bucket %s !!", fileName, bucketName)

	return nil
}

func (a *awsClient) UploadLargeObject(fileName string, bucketName string, parentBucketName string) error {

	location := os.Getenv("LOCATION")
	keyName := fileName
	// if bucketName is demo/test/test2
	// parentBucketName will be "demo"
	if strings.Contains(bucketName, "/") {
		index := strings.Index(bucketName, "/")
		keyName = fmt.Sprintf("%s/%s", bucketName[index+1:], fileName)
	}
	filePath := fmt.Sprintf("%s/%s", location, fileName)

	//divide the file into 1GB parts
	var partGiBs int64 = 1
	s3Client := s3.NewFromConfig(*a.cfg)
	uploader := manager.NewUploader(s3Client, func(u *manager.Uploader) {
		u.PartSize = partGiBs * 1024 * 1024 * 1024
	})

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("Couldn't open large file %v to upload. Here's why: %v\n", filePath, err)
	}

	defer file.Close()

	log.Printf("Starting upload of file %s", filePath)
	_, err = uploader.Upload(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(parentBucketName),
		Key:    aws.String(keyName),
		Body:   file,
	})
	if err != nil {
		return fmt.Errorf("Couldn't upload large file %v to %v:%v. Here's why: %v\n", filePath, bucketName, fileName, err)
	}
	log.Printf("File (Large) %s uploaded to s3 bucket %s !!", fileName, bucketName)
	return err
}
