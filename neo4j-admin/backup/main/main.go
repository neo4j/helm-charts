package main

import (
	"log"
	"os"
)

func main() {

	startupOperations()

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
		log.Fatalf("Incorrect cloud provider %s", cloudProvider)
	}
}
