package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	neo4jAdmin "github.com/neo4j/helm-charts/neo4j-admin/backup/neo4j-admin"
	"k8s.io/utils/strings/slices"
)

// cloudOperations handles backup operations for all cloud providers using Neo4j native cloud storage
func cloudOperations() {
	log.Printf("Using Neo4j native cloud storage backup for provider: %s", os.Getenv("CLOUD_PROVIDER"))

	if aggregateEnabled := os.Getenv("AGGREGATE_BACKUP_ENABLED"); aggregateEnabled == "true" {
		err := aggregateBackupOperations()
		handleError(err)
		return
	}

	backupFileNames, consistencyCheckReports, err := backupOperations()
	handleError(err)

	// Only handle consistency check reports if they exist and need local processing
	enableConsistencyCheck := os.Getenv("CONSISTENCY_CHECK_ENABLE")
	if enableConsistencyCheck == "true" && len(consistencyCheckReports) > 0 {
		log.Printf("Consistency check reports generated: %v", consistencyCheckReports)
		// Note: Consistency checks still generate local reports that may need manual handling
	}

	log.Printf("Cloud backup completed successfully. Files: %v", backupFileNames)
}

// awsOperations
func awsOperations() {
	cloudOperations()
}

// gcpOperations
func gcpOperations() {
	cloudOperations()
}

// azureOperations
func azureOperations() {
	cloudOperations()
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

// Returns backup file names and consistency check reports
func backupOperations() ([]string, []string, error) {
	// Clean up any existing backup files first
	if err := deleteBackupFiles([]string{}, []string{}); err != nil {
		log.Printf("Warning: failed to cleanup existing backups: %v", err)
	}

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
	log.Printf("Backup completed successfully. Files: %v", backupFileNames)

	// Handle consistency checks if enabled
	if consistencyCheckEnabled == "true" {
		if len(consistencyCheckDBs) == 1 && consistencyCheckDBs[0] == "" {
			// Perform consistency check for all databases
			reportArchiveName, err := neo4jAdmin.PerformConsistencyCheck("")
			if err != nil {
				return nil, nil, err
			}
			if len(reportArchiveName) != 0 {
				consistencyCheckReports = append(consistencyCheckReports, reportArchiveName)
			}
		} else {
			// Perform consistency check for each specified database
			for _, consistencyCheckDB := range consistencyCheckDBs {
				if consistencyCheckDB == "" {
					continue // Skip empty entries
				}
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
	}
	return backupFileNames, consistencyCheckReports, nil
}

func aggregateBackupOperations() error {
	log.Printf("Performing aggregate backup")
	err := neo4jAdmin.PerformAggregateBackup()
	if err != nil {
		return err
	}
	log.Printf("Aggregate backup completed successfully")
	return nil
}

func startupOperations() {
	address, err := generateAddress()
	handleError(err)

	err = neo4jAdmin.CheckDatabaseConnectivity(address)
	handleError(err)

	// Set backup location - will be overridden for cloud storage in helpers.go
	backupPath := "/backups"
	if path := os.Getenv("BACKUP_PATH"); path != "" {
		backupPath = path
	}
	os.Setenv("LOCATION", backupPath)
}

func handleError(err error) {
	if err != nil {
		log.Fatal(err.Error())
	}
}

// generateAddress returns the backup address in the format <hostip:port> or <standalone-admin.default.svc.cluster.local:port>
func generateAddress() (string, error) {
	if endpoints := os.Getenv("DATABASE_BACKUP_ENDPOINTS"); len(endpoints) > 0 {
		return endpoints, nil
	}

	// Legacy support for single endpoint
	if ip := os.Getenv("DATABASE_SERVICE_IP"); len(ip) > 0 {
		return fmt.Sprintf("%s:%s", ip, os.Getenv("DATABASE_BACKUP_PORT")), nil
	}

	if serviceName := os.Getenv("DATABASE_SERVICE_NAME"); len(serviceName) > 0 {
		return fmt.Sprintf("%s.%s.svc.%s:%s",
			serviceName,
			os.Getenv("DATABASE_NAMESPACE"),
			os.Getenv("DATABASE_CLUSTER_DOMAIN"),
			os.Getenv("DATABASE_BACKUP_PORT")), nil
	}

	return "", fmt.Errorf("no valid backup endpoints specified")
}

func deleteBackupFiles(backupFileNames, consistencyCheckReports []string) error {
	if value, present := os.LookupEnv("KEEP_BACKUP_FILES"); present && value == "false" {
		backupPath := "/backups"
		if path := os.Getenv("BACKUP_PATH"); path != "" {
			backupPath = path
		}

		for _, backupFileName := range backupFileNames {
			log.Printf("Deleting file %s/%s", backupPath, backupFileName)
			err := os.Remove(fmt.Sprintf("%s/%s", backupPath, backupFileName))
			if err != nil {
				return err
			}
		}
		for _, consistencyCheckReportName := range consistencyCheckReports {
			log.Printf("Deleting file %s/%s", backupPath, consistencyCheckReportName)
			err := os.Remove(fmt.Sprintf("%s/%s", backupPath, consistencyCheckReportName))
			if err != nil {
				return err
			}
		}
	}
	return nil
}
