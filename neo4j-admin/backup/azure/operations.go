package azure

import (
	"errors"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"golang.org/x/net/context"
	"log"
	"os"
	"strings"
)

func (a *azureClient) CheckContainerAccess(containerName string) error {

	prefix := ""
	parentContainerName := containerName
	options := &azblob.ListBlobsFlatOptions{
		Include: azblob.ListBlobsInclude{Snapshots: true, Versions: true},
	}
	if strings.Contains(containerName, "/") {
		index := strings.Index(containerName, "/")
		prefix = containerName[index+1:]
		parentContainerName = containerName[:index]
		options.Prefix = &prefix
	}
	pager := a.client.NewListBlobsFlatPager(parentContainerName, options)

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
func (a *azureClient) UploadFile(fileName string, location string, containerName string) error {

	prefix := ""
	parentContainerName := containerName
	// if bucketName is demo/test/test2
	// parentBucketName will be "demo"
	if strings.Contains(containerName, "/") {
		index := strings.Index(containerName, "/")
		parentContainerName = containerName[:index]
		prefix = containerName[index+1:]
	}
	filePath := fmt.Sprintf("%s/%s", location, fileName)
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("Couldn't open file %v to upload. Here's why: %v\n", filePath, err)
	}
	defer file.Close()

	name := fileName
	if prefix != "" {
		name = fmt.Sprintf("%s/%s", prefix, fileName)
	}
	log.Printf("Starting upload of file %s", filePath)
	_, err = a.client.UploadFile(context.TODO(), parentContainerName, name, file, nil)
	if err != nil {
		return fmt.Errorf("Couldn't upload file %v to %v Here's why: %v\n", filePath, containerName, err)
	}
	log.Printf("File %s uploaded to azure container %s !!", fileName, containerName)
	return nil
}
