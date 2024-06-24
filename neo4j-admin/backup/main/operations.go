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

	credentialPath := os.Getenv("CREDENTIAL_PATH")
	awsClient, err := aws.NewAwsClient(credentialPath)
	handleError(err)

	if aggregateEnabled := os.Getenv("AGGREGATE_BACKUP_ENABLED"); aggregateEnabled == "true" {

		log.Println("credential path", credentialPath)
		//service account is NOT used hence env variables need to be set for aggregate backup operation
		if credentialPath != "/credentials/" {
			log.Println("generating env variables from creds")
			err = awsClient.GenerateEnvVariablesFromCredentials()
			handleError(err)
		}

		err = aggregateBackupOperations()
		handleError(err)
		return
	}

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

func onPrem() {

	if aggregateEnabled := os.Getenv("AGGREGATE_BACKUP_ENABLED"); aggregateEnabled == "true" {
		err := aggregateBackupOperations()
		handleError(err)
		return
	}

	backupFileNames, consistencyCheckReports, err := backupOperations()
	handleError(err)

	err = deleteBackupFiles(backupFileNames, consistencyCheckReports)
	handleError(err)

}

// backupOperations returns backupFileNames , consistencyCheckReports and error
// performs aggregate backup is aggregate backup is enabled
func backupOperations() ([]string, []string, error) {

	address, err := generateAddress()
	if err != nil {
		return nil, nil, err
	}
	databases := strings.Split(os.Getenv("DATABASE"), ",")
	consistencyCheckDBs := strings.Split(os.Getenv("CONSISTENCY_CHECK_DATABASE"), ",")
	consistencyCheckEnabled := os.Getenv("CONSISTENCY_CHECK_ENABLE")

	var consistencyCheckReports []string
	backupFileNames, err := neo4jAdmin.PerformBackup(address)
	if err != nil {
		return nil, nil, err
	}
	log.Printf("Backup File Name(s) %v", backupFileNames)

	if consistencyCheckEnabled == "true" {
		for _, consistencyCheckDB := range consistencyCheckDBs {
			if slices.Contains(databases, consistencyCheckDB) || slices.Contains(databases, "*") {
				reportArchiveName, err := neo4jAdmin.PerformConsistencyCheck(consistencyCheckDB)
				if err != nil {
					return nil, nil, err
				}
				if len(reportArchiveName) != 0 {
					consistencyCheckReports = append(consistencyCheckReports, reportArchiveName)
				}
			}
		}
	}
	return backupFileNames, consistencyCheckReports, nil
}

// aggregateBackupOperations perform aggregate backup
func aggregateBackupOperations() error {
	err := neo4jAdmin.PerformAggregateBackup()
	if err != nil {
		return err
	}
	return nil
}

func startupOperations() {
	address, err := generateAddress()
	handleError(err)

	err = neo4jAdmin.CheckDatabaseConnectivity(address)
	handleError(err)

	os.Setenv("LOCATION", "/backups")
}

func handleError(err error) {
	if err != nil {
		log.Fatal(err.Error())
	}
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

func deleteBackupFiles(backupFileNames, consistencyCheckReports []string) error {
	if value, present := os.LookupEnv("KEEP_BACKUP_FILES"); present && value == "false" {
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
	}
	return nil
}
