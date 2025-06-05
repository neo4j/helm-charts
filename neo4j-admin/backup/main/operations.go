package main

import (
	"fmt"
	"log"
	"os"
	"regexp"
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
		if len(backupFileNames) == 0 {
			return nil, nil, fmt.Errorf("no backup files found, cannot perform consistency check")
		}

		if len(consistencyCheckDBs) == 1 && consistencyCheckDBs[0] == "" {
			// Perform consistency check for all databases using the first backup file as reference
			firstBackupFile := backupFileNames[0]
			baseName := strings.TrimSuffix(firstBackupFile, ".backup")
			reportArchiveName, err := neo4jAdmin.PerformConsistencyCheck("", baseName)
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
					// Find the corresponding backup file for this database
					var matchingBackupFile string
					for _, backupFile := range backupFileNames {
						if strings.HasPrefix(backupFile, consistencyCheckDB+"-") {
							matchingBackupFile = strings.TrimSuffix(backupFile, ".backup")
							break
						}
					}

					if matchingBackupFile != "" {
						reportArchiveName, err := neo4jAdmin.PerformConsistencyCheck(consistencyCheckDB, matchingBackupFile)
						if err != nil {
							return nil, nil, err
						}
						if len(reportArchiveName) != 0 {
							consistencyCheckReports = append(consistencyCheckReports, reportArchiveName)
						}
					} else {
						log.Printf("Warning: No backup file found for database %s, skipping consistency check", consistencyCheckDB)
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
	// Load Azure credentials from file if using secret-based authentication
	if os.Getenv("CLOUD_PROVIDER") == "azure" && os.Getenv("CREDENTIAL_PATH") != "" {
		loadAzureCredentialsFromFile()
	}

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

// loadAzureCredentialsFromFile reads Azure credentials from the mounted secret file
// and sets the appropriate environment variables
func loadAzureCredentialsFromFile() {
	credentialPath := os.Getenv("CREDENTIAL_PATH")
	if credentialPath == "" {
		log.Printf("Warning: CREDENTIAL_PATH not set for Azure, skipping credential file loading")
		return
	}

	log.Printf("Loading Azure credentials from file: %s", credentialPath)

	// Read the credentials file
	content, err := os.ReadFile(credentialPath)
	if err != nil {
		log.Printf("Warning: Failed to read Azure credentials file %s: %v", credentialPath, err)
		return
	}

	// Parse AZURE_STORAGE_ACCOUNT_NAME using regex (consistent with dev branch approach)
	storageAccountName, err := extractAzureStorageAccountName(string(content))
	if err != nil {
		log.Printf("Warning: Failed to extract AZURE_STORAGE_ACCOUNT_NAME: %v", err)
	} else {
		os.Setenv("AZURE_STORAGE_ACCOUNT_NAME", storageAccountName)
		log.Printf("Set AZURE_STORAGE_ACCOUNT_NAME from credentials file")
	}

	// Parse AZURE_STORAGE_ACCOUNT_KEY using regex (consistent with dev branch approach)
	storageAccountKey, err := extractAzureStorageAccountKey(string(content))
	if err != nil {
		log.Printf("Warning: Failed to extract AZURE_STORAGE_ACCOUNT_KEY: %v", err)
	} else {
		os.Setenv("AZURE_STORAGE_ACCOUNT_KEY", storageAccountKey)
		log.Printf("Set AZURE_STORAGE_ACCOUNT_KEY from credentials file")
	}
}

// extractAzureStorageAccountName extracts the storage account name from Azure credentials file
func extractAzureStorageAccountName(data string) (string, error) {
	re := regexp.MustCompile(`AZURE_STORAGE_ACCOUNT_NAME=(.*)`)
	matches := re.FindStringSubmatch(data)
	if len(matches) == 0 {
		return "", fmt.Errorf("missing azure storage account name")
	} else if len(matches) > 2 {
		return "", fmt.Errorf("more than one azure storage account name found: %v", matches)
	}
	// FindStringSubmatch returns the complete string and then the matches, hence index 1 is the first match
	return strings.TrimSpace(matches[1]), nil
}

// extractAzureStorageAccountKey extracts the storage account key from Azure credentials file
func extractAzureStorageAccountKey(data string) (string, error) {
	re := regexp.MustCompile(`AZURE_STORAGE_ACCOUNT_KEY=(.*)`)
	matches := re.FindStringSubmatch(data)
	if len(matches) == 0 {
		return "", fmt.Errorf("missing storage account key")
	} else if len(matches) > 2 {
		return "", fmt.Errorf("more than one storage account key found: %v", matches)
	}
	return strings.TrimSpace(matches[1]), nil
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
