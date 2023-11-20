package aws

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"os"
)

type awsClient struct {
	cfg *aws.Config
}

func NewAwsClient(credentialPath string) (*awsClient, error) {
	var cfg aws.Config
	var err error
	if credentialPath == "/credentials/" {
		_, present := os.LookupEnv("AWS_WEB_IDENTITY_TOKEN_FILE")
		if !present {
			return nil, fmt.Errorf("error while creating aws client without credentials file\n Missing AWS_WEB_IDENTITY_TOKEN_FILE")
		}
		cfg, err = config.LoadDefaultConfig(
			context.TODO(),
			config.WithRegion(os.Getenv("AWS_REGION")))
		if err != nil {
			return nil, fmt.Errorf("error while creating aws client without credentials file\n %v", err)
		}
	} else {

		cfg, err = config.LoadDefaultConfig(
			context.TODO(),
			config.WithSharedCredentialsFiles(
				[]string{credentialPath},
			))
		if err != nil {
			return nil, fmt.Errorf("error while creating aws client \n %v", err)
		}
	}

	return &awsClient{
		cfg: &cfg,
	}, nil
}
