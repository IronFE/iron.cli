package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

func CreateConfig(access *AwsAccountAccess, region string) (aws.Config, error) {

	credsProvider := credentials.NewStaticCredentialsProvider(access.AccessKeyId, access.SecretKey, access.SessionToken)

	return config.LoadDefaultConfig(context.Background(), config.WithRegion(region), config.WithCredentialsProvider(credsProvider))
}
