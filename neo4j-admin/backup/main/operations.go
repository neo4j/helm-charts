package main

import (
	"fmt"
	"github.com/neo4j/helm-charts/neo4j-admin/backup/aws"
	"github.com/neo4j/helm-charts/neo4j-admin/backup/azure"
	gcp "github.com/neo4j/helm-charts/neo4j-admin/backup/gcp"
	neo4jAdmin "github.com/neo4j/helm-charts/neo4j-admin/backup/neo4j-admin"
	"k8s.io/utils/strings/slices"
	"log"
	"os"
	"strings"
)

func awsOperations() {
	awsClient, err := aws.NewAwsClient(os.Getenv("CREDENTIAL_PATH"))
	handleError(err)

	bucketName := os.Getenv("BUCKET_NAME")
	err = awsClient.CheckBucketAccess(bucketName)
	handleError(err)

	backupFileNames, consistencyCheckReports, err := backupOperations()
	handleError(err)

	err = awsClient.UploadFile(backupFileNames, bucketName)
	handleError(err)

	enableConsistencyCheck := os.Getenv("CONSISTENCY_CHECK_ENABLE")
	if enableConsistencyCheck == "true" {
		err = awsClient.UploadFile(consistencyCheckReports, bucketName)
		handleError(err)
	}
	err = deleteBackupFiles(backupFileNames, consistencyCheckReports)
	handleError(err)
}

func gcpOperations() {
	gcpClient, err := gcp.NewGCPClient(os.Getenv("CREDENTIAL_PATH"))
	handleError(err)

	bucketName := os.Getenv("BUCKET_NAME")
	err = gcpClient.CheckBucketAccess(bucketName)
	handleError(err)

	backupFileNames, consistencyCheckReports, err := backupOperations()
	handleError(err)

	err = gcpClient.UploadFile(backupFileNames, bucketName)
	handleError(err)

	enableConsistencyCheck := os.Getenv("CONSISTENCY_CHECK_ENABLE")
	if enableConsistencyCheck == "true" {
		err = gcpClient.UploadFile(consistencyCheckReports, bucketName)
		handleError(err)
	}
	err = deleteBackupFiles(backupFileNames, consistencyCheckReports)
	handleError(err)
}

func azureOperations() {
	azureClient, err := azure.NewAzureClient(os.Getenv("CREDENTIAL_PATH"))
	handleError(err)

	containerName := os.Getenv("BUCKET_NAME")
	err = azureClient.CheckContainerAccess(containerName)
	handleError(err)

	backupFileNames, consistencyCheckReports, err := backupOperations()
	handleError(err)

	err = azureClient.UploadFile(backupFileNames, containerName)
	handleError(err)

	enableConsistencyCheck := os.Getenv("CONSISTENCY_CHECK_ENABLE")
	if enableConsistencyCheck == "true" {
		err = azureClient.UploadFile(consistencyCheckReports, containerName)
		handleError(err)
	}
	err = deleteBackupFiles(backupFileNames, consistencyCheckReports)
	handleError(err)
}

func backupOperations() ([]string, []string, error) {

	address, err := generateAddress()
	if err != nil {
		return nil, nil, err
	}
	databases := strings.Split(os.Getenv("DATABASE"), ",")
	consistencyCheckDBs := strings.Split(os.Getenv("CONSISTENCY_CHECK_DATABASE"), ",")
	var fileNames, consistencyCheckReports []string
	for _, database := range databases {
		fileName, err := neo4jAdmin.PerformBackup(address, database)
		if err != nil {
			return nil, nil, err
		}
		log.Printf("Backup File Name is %s", fileName)
		fileNames = append(fileNames, fileName)

		if slices.Contains(consistencyCheckDBs, database) {
			if os.Getenv("CONSISTENCY_CHECK_ENABLE") == "true" {
				reportArchiveName, err := neo4jAdmin.PerformConsistencyCheck(fileName, database)
				if err != nil {
					return nil, nil, err
				}
				consistencyCheckReports = append(consistencyCheckReports, reportArchiveName)
			}
		}
	}

	return fileNames, consistencyCheckReports, nil
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

func deleteBackupFiles(backupFileNames, consistencyCheckReports []string) error {
	for _, backupFileName := range backupFileNames {
		log.Printf("Deleting file /backups/%s", backupFileName)
		err := os.Remove(fmt.Sprintf("/backups/%s", backupFileName))
		if err != nil {
			return err
		}
	}
	for _, consistencyCheckReportName := range consistencyCheckReports {
		log.Printf("Deleting file /backups/%s", consistencyCheckReportName)
		err := os.Remove(fmt.Sprintf("/backups/%s", consistencyCheckReportName))
		if err != nil {
			return err
		}
	}
	return nil
}
