package azure

import (
	"errors"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"os"
	"regexp"
)

type azureClient struct {
	credential *azblob.SharedKeyCredential
	serviceURL string
}

func NewAzureClient(credentialPath string) (*azureClient, error) {
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

	cred, err := azblob.NewSharedKeyCredential(storageAccountName, storageAccountKey)
	if err != nil {
		fmt.Printf("Failed to create azure credential: %v\n", err)
		return nil, err
	}

	return &azureClient{
		credential: cred,
		serviceURL: fmt.Sprintf("https://%s.blob.core.windows.net/", storageAccountName),
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
