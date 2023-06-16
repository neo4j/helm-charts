package main

import (
	"fmt"
	"github.com/neo4j/helm-charts/neo4j-admin/backup/aws"
	"github.com/neo4j/helm-charts/neo4j-admin/backup/azure"
	gcp "github.com/neo4j/helm-charts/neo4j-admin/backup/gcp"
	neo4jAdmin "github.com/neo4j/helm-charts/neo4j-admin/backup/neo4j-admin"
	"log"
	"os"
)

func awsOperations() {
	awsClient, err := aws.NewAwsClient(os.Getenv("CREDENTIAL_PATH"))
	handleError(err)

	bucketName := os.Getenv("BUCKET_NAME")
	err = awsClient.CheckBucketAccess(bucketName)
	handleError(err)

	backupFileNames, err := backupOperations()
	handleError(err)

	for _, backupFileName := range backupFileNames {
		err = awsClient.UploadFile(backupFileName, "/backups", bucketName)
		handleError(err)
	}

	err = deleteBackupFiles(backupFileNames)
	handleError(err)
}

func gcpOperations() {
	gcpClient, err := gcp.NewGCPClient(os.Getenv("CREDENTIAL_PATH"))
	handleError(err)

	bucketName := os.Getenv("BUCKET_NAME")
	err = gcpClient.CheckBucketAccess(bucketName)
	handleError(err)

	backupFileNames, err := backupOperations()
	handleError(err)

	for _, backupFileName := range backupFileNames {
		err = gcpClient.UploadFile(backupFileName, "/backups", bucketName)
		handleError(err)
	}

	err = deleteBackupFiles(backupFileNames)
	handleError(err)
}

func azureOperations() {
	azureClient, err := azure.NewAzureClient(os.Getenv("CREDENTIAL_PATH"))
	handleError(err)

	containerName := os.Getenv("BUCKET_NAME")
	err = azureClient.CheckContainerAccess(containerName)
	handleError(err)

	backupFileNames, err := backupOperations()
	handleError(err)

	for _, backupFileName := range backupFileNames {
		err = azureClient.UploadFile(backupFileName, "/backups", containerName)
		handleError(err)
	}

	err = deleteBackupFiles(backupFileNames)
	handleError(err)
}

func backupOperations() ([]string, error) {

	address, err := generateAddress()
	if err != nil {
		return nil, err
	}
	fileNames, err := neo4jAdmin.PerformBackup(address)
	if err != nil {
		return nil, err
	}
	log.Printf("Backup File Names are %v", fileNames)

	return fileNames, nil
}

// startupOperations includes the following
func startupOperations() {
	dir, err := os.Getwd()
	handleError(err)
	log.Printf("printing current directory %s", dir)

	address, err := generateAddress()
	handleError(err)

	err = neo4jAdmin.CheckDatabaseConnectivity(address)
	handleError(err)
}

// generateAddress returns the backup address in the format <hostip:port> or <standalone-admin.default.svc.cluster.local:port>
func generateAddress() (string, error) {
	if ip := os.Getenv("DATABASE_SERVICE_IP"); len(ip) > 0 {
		address := fmt.Sprintf("%s:%s", ip, os.Getenv("DATABASE_BACKUP_PORT"))
		log.Printf("Address := %s", address)
		return address, nil
	}
	if serviceName := os.Getenv("DATABASE_SERVICE_NAME"); len(serviceName) > 0 {
		address := fmt.Sprintf("%s.%s.svc.%s:%s", serviceName, os.Getenv("DATABASE_NAMESPACE"), os.Getenv("DATABASE_CLUSTER_DOMAIN"), os.Getenv("DATABASE_BACKUP_PORT"))
		log.Printf("Address := %s", address)
		return address, nil
	}
	return "", fmt.Errorf("cannot generate address. Invalid DATABASE_SERVICE_IP = %s or DATABASE_SERVICE_NAME = %s", os.Getenv("DATABASE_SERVICE_IP"), os.Getenv("DATABASE_SERVICE_NAME"))
}

func handleError(err error) {
	if err != nil {
		log.Fatal(err.Error())
	}
}

func deleteBackupFiles(backupFileNames []string) error {
	for _, backupFileName := range backupFileNames {
		log.Printf("Deleting file /backups/%s", backupFileName)
		err := os.Remove(fmt.Sprintf("/backups/%s", backupFileName))
		if err != nil {
			return err
		}
	}
	return nil
}
