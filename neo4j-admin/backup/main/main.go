package main

import (
	"log"
	"os"
)

var errorLogger = log.New(os.Stderr, "", log.LstdFlags)

func configureLogging() {
	log.SetOutput(os.Stdout)
}

func main() {
	configureLogging()

	if aggregateEnabled := os.Getenv("AGGREGATE_BACKUP_ENABLED"); aggregateEnabled != "true" {
		startupOperations()
	}

	cloudProvider := os.Getenv("CLOUD_PROVIDER")
	switch cloudProvider {
	case "aws":
		awsOperations()
		break
	case "azure":
		azureOperations()
		break
	case "gcp":
		gcpOperations()
		break
	case "":
		onPrem()
		break
	default:
		errorLogger.Fatalf("Incorrect cloud provider %s", cloudProvider)
	}

}
