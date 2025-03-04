package aws

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
)

type awsClient struct {
	cfg *aws.Config
}

func NewAwsClient(credentialPath string) (*awsClient, error) {
	var cfg aws.Config
	var err error

	// Check if a custom region is specified
	customRegion := os.Getenv("S3_REGION")

	if credentialPath == "/credentials/" {
		_, present := os.LookupEnv("AWS_WEB_IDENTITY_TOKEN_FILE")
		if !present {
			return nil, fmt.Errorf("error while creating aws client without credentials file\n Missing AWS_WEB_IDENTITY_TOKEN_FILE")
		}

		// Use custom region if provided, otherwise use AWS_REGION
		region := os.Getenv("AWS_REGION")
		if customRegion != "" {
			region = customRegion
		}

		cfg, err = config.LoadDefaultConfig(
			context.TODO(),
			config.WithRegion(region))
		if err != nil {
			return nil, fmt.Errorf("error while creating aws client without credentials file\n %v", err)
		}
	} else {
		// Load options for config
		var options []func(*config.LoadOptions) error

		// Add credentials file
		options = append(options, config.WithSharedCredentialsFiles([]string{credentialPath}))

		// Add custom region if provided
		if customRegion != "" {
			options = append(options, config.WithRegion(customRegion))
		}

		cfg, err = config.LoadDefaultConfig(context.TODO(), options...)
		if err != nil {
			return nil, fmt.Errorf("error while creating aws client \n %v", err)
		}
	}

	return &awsClient{
		cfg: &cfg,
	}, nil
}
