package aws

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
)

type awsClient struct {
	cfg *aws.Config
}

func NewAwsClient(credentialPath string) (*awsClient, error) {
	cfg, err := config.LoadDefaultConfig(
		context.TODO(),
		config.WithSharedCredentialsFiles(
			[]string{credentialPath},
		))
	if err != nil {
		return nil, fmt.Errorf("error while creating aws client \n %v", err)
	}

	return &awsClient{
		cfg: &cfg,
	}, nil
}
