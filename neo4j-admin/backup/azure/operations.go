package azure

import (
	"errors"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"golang.org/x/net/context"
	"log"
	"os"
)

func (a *azureClient) CheckContainerAccess(containerName string) error {

	pager := a.client.NewListBlobsFlatPager(containerName, &azblob.ListBlobsFlatOptions{
		Include: azblob.ListBlobsInclude{Snapshots: true, Versions: true},
	})

	_, err := pager.NextPage(context.TODO())
	if err != nil {
		var azureResponseError *azcore.ResponseError
		if errors.As(err, &azureResponseError) && azureResponseError.ErrorCode == "ContainerNotFound" {
			return errors.New(fmt.Sprintf("ContainerName = %s , ErrorMessage = %s", containerName, azureResponseError.RawResponse.Status))
		}
		log.Printf("error %v", err)
		return err
	}
	log.Printf("Connectivity with Azure container '%s' established", containerName)
	return nil
}

// UploadFile uploads the file present at the provided location to the azure container
func (a *azureClient) UploadFile(fileNames []string, containerName string) error {
	location := "/backups"
	for _, fileName := range fileNames {

		filePath := fmt.Sprintf("%s/%s", location, fileName)
		file, err := os.Open(filePath)
		if err != nil {
			return fmt.Errorf("Couldn't open file %v to upload. Here's why: %v\n", filePath, err)
		}

		log.Printf("Starting upload of file %s", filePath)
		_, err = a.client.UploadFile(context.TODO(), containerName, fileName, file, nil)
		if err != nil {
			return fmt.Errorf("Couldn't upload file %v to %v Here's why: %v\n", filePath, containerName, err)
		}
		log.Printf("File %s uploaded to azure container %s !!", fileName, containerName)
		file.Close()
	}
	return nil
}
