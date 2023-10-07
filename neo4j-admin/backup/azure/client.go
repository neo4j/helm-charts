package azure

import (
	"errors"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"log"
	"os"
	"regexp"
)

type azureClient struct {
	client *azblob.Client
}

func NewAzureClient(credentialPath string) (*azureClient, error) {

	var client *azblob.Client

	if credentialPath == "/credentials/" {
		storageAccountName := os.Getenv("AZURE_STORAGE_ACCOUNT_NAME")
		log.Printf("Azure storage account name %v", storageAccountName)
		serviceURL := fmt.Sprintf("https://%s.blob.core.windows.net/", storageAccountName)
		cred, err := azidentity.NewDefaultAzureCredential(nil)
		if err != nil {
			return nil, fmt.Errorf("Failed to create azure credential without sharedKeyCredentials: %v\n", err)
		}

		client, err = azblob.NewClient(serviceURL, cred, nil)
		if err != nil {
			return nil, fmt.Errorf("error while creating azblob client\n err = %v", err)
		}
	} else {

		dataBytes, err := os.ReadFile(credentialPath)
		if err != nil {
			return nil, fmt.Errorf("unable to open azure credential file \n credentialPath = %s \n err = %v", credentialPath, err)
		}
		storageAccountName, err := getStorageAccountName(string(dataBytes))
		if err != nil {
			return nil, err
		}
		storageAccountKey, err := getStorageAccountKey(string(dataBytes))
		if err != nil {
			return nil, err
		}
		serviceURL := fmt.Sprintf("https://%s.blob.core.windows.net/", storageAccountName)

		cred, err := azblob.NewSharedKeyCredential(storageAccountName, storageAccountKey)
		if err != nil {
			return nil, fmt.Errorf("Failed to create azure credential using sharedkeycredentials: %v\n", err)
		}
		client, err = azblob.NewClientWithSharedKeyCredential(serviceURL, cred, nil)
		if err != nil {
			return nil, fmt.Errorf("error while creating azblob client using sharedkeycredentials\n err = %v", err)
		}
	}

	return &azureClient{
		client: client,
	}, nil
}

func getStorageAccountName(data string) (string, error) {
	re := regexp.MustCompile(`AZURE_STORAGE_ACCOUNT_NAME=(.*)`)
	matches := re.FindStringSubmatch(data)
	if len(matches) == 0 {
		return "", errors.New("missing azure storage account name")
	} else if len(matches) > 2 {
		return "", fmt.Errorf("more than one azure storage account name found !! \n matches = %v", matches)
	}
	//findStringSubmatch returns the complete string and then the matches hence the index 1 is the first match
	return matches[1], nil
}

func getStorageAccountKey(data string) (string, error) {
	re := regexp.MustCompile(`AZURE_STORAGE_ACCOUNT_KEY=(.*)`)
	matches := re.FindStringSubmatch(data)
	if len(matches) == 0 {
		return "", errors.New("missing storage account key")
	} else if len(matches) > 2 {
		return "", fmt.Errorf("more than one storage account key found !! \n matches = %v", matches)
	}
	return matches[1], nil
}
