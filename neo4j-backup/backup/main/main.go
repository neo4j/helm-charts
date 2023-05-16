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
	default:
		log.Fatalf("Incorrect cloud provider %s", cloudProvider)
	}
	//
	//log.Println("Sleeping for 100 minutes")
	//time.Sleep(100 * time.Minute)
	//log.Println("sleep over .. wake up !!!")
}
