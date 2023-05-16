package main

import (
	"github.com/neo4j/helm-charts/neo4j-backup/backup/aws"
	"github.com/neo4j/helm-charts/neo4j-backup/backup/azure"
	gcp "github.com/neo4j/helm-charts/neo4j-backup/backup/gcp"
	neo4jAdmin "github.com/neo4j/helm-charts/neo4j-backup/backup/neo4j-admin"
	"log"
	"os"
)

func awsOperations() {
	awsClient, err := aws.NewAwsClient(os.Getenv("CREDENTIAL_PATH"))
	handleError(err)

	bucketName := os.Getenv("BUCKET_NAME")
	err = awsClient.CheckBucketAccess(bucketName)
	handleError(err)

	backupFileName, consistencyCheckReport, err := backupOperations()
	handleError(err)

	err = awsClient.UploadFile(backupFileName, "/backups", bucketName)
	handleError(err)

	enableConsistencyCheck := os.Getenv("CONSISTENCY_CHECK_ENABLE")
	if enableConsistencyCheck == "true" {
		err = awsClient.UploadFile(consistencyCheckReport, "/backups", bucketName)
		handleError(err)
	}
}

func gcpOperations() {
	gcpClient, err := gcp.NewGCPClient(os.Getenv("CREDENTIAL_PATH"))
	handleError(err)

	bucketName := os.Getenv("BUCKET_NAME")
	err = gcpClient.CheckBucketAccess(bucketName)
	handleError(err)

	backupFileName, consistencyCheckReport, err := backupOperations()
	handleError(err)

	err = gcpClient.UploadFile(backupFileName, "/backups", bucketName)
	handleError(err)

	enableConsistencyCheck := os.Getenv("CONSISTENCY_CHECK_ENABLE")
	if enableConsistencyCheck == "true" {
		err = gcpClient.UploadFile(consistencyCheckReport, "/backups", bucketName)
		handleError(err)
	}
}

func azureOperations() {
	azureClient, err := azure.NewAzureClient(os.Getenv("CREDENTIAL_PATH"))
	handleError(err)

	containerName := os.Getenv("BUCKET_NAME")
	err = azureClient.CheckContainerAccess(containerName)
	handleError(err)

	backupFileName, consistencyCheckReport, err := backupOperations()
	handleError(err)

	err = azureClient.UploadFile(backupFileName, "/backups", containerName)
	handleError(err)

	enableConsistencyCheck := os.Getenv("CONSISTENCY_CHECK_ENABLE")
	if enableConsistencyCheck == "true" {
		err = azureClient.UploadFile(consistencyCheckReport, "/backups", containerName)
		handleError(err)
	}
}

func backupOperations() (string, string, error) {
	fileName, err := neo4jAdmin.PerformBackup()
	if err != nil {
		return "", "", err
	}
	log.Printf("Backup File Name is %s", fileName)

	enableConsistencyCheck := os.Getenv("CONSISTENCY_CHECK_ENABLE")
	var reportArchiveName string
	if enableConsistencyCheck == "true" {
		reportArchiveName, err = neo4jAdmin.PerformConsistencyCheck(fileName)
		if err != nil {
			return "", "", err
		}
	}
	return fileName, reportArchiveName, nil
}

// startupOperations includes the following
func startupOperations() {
	dir, err := os.Getwd()
	handleError(err)
	log.Printf("printing current directory %s", dir)

	err = neo4jAdmin.CheckDatabaseConnectivity(os.Getenv("ADDRESS"))
	handleError(err)
}

func handleError(err error) {
	if err != nil {
		log.Fatal(err.Error())
	}
}
