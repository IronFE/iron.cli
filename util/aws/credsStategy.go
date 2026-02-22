package aws

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/IronFE/iron.cli/util"
	"github.com/apex/log"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/organizations"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/pkg/errors"
)

type AwsOrganizationInfo struct {
	Id string
}

type awsAbstraction struct {
	config aws.Config
	region string
	mfaSerial string
}

func newCredsStrategyAws(profileName string, region string, mfaSerial string) (IAws, error) {
	defaultConfig, err := config.LoadDefaultConfig(context.Background(), config.WithSharedConfigProfile(profileName))
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}
	return &awsAbstraction{
		config: defaultConfig,
		region: region,
		mfaSerial: mfaSerial,
	}, nil
}

func (a *awsAbstraction) FindAccountId(alias string) (string, error) {
	client := organizations.NewFromConfig(a.config)
	response, err := client.ListAccounts(context.Background(), &organizations.ListAccountsInput{})
	if err != nil {
		return "", errors.Wrap(err, "failed to list accounts")
	}

	for _, account := range response.Accounts {
		if strings.ToLower(*account.Name) == strings.ToLower(alias) {
			return *account.Id, nil
		}
	}

	return "", errors.Errorf("Could not find account with alias %s", alias)
}

func (a *awsAbstraction) OrganizationInfo() (AwsOrganizationInfo, error) {

	client := organizations.NewFromConfig(a.config)

	output, err := client.DescribeOrganization(context.Background(), &organizations.DescribeOrganizationInput{})
	if err != nil {
		return AwsOrganizationInfo{}, fmt.Errorf("failed to describe organization: %:w", err)
	}

	return AwsOrganizationInfo{
		Id: *output.Organization.Id,
	}, nil
}

func (a *awsAbstraction) AssumeRole(role, accountName string) (*AwsAccountAccess, error) {

	return a.assumeRole(role, accountName, false)
}

func (a *awsAbstraction) AssumeRoleWithMfa(role, accountName string) (*AwsAccountAccess, error) {
	return a.assumeRole(role, accountName, true)
}

func (a *awsAbstraction) SessionToken(duration time.Duration) (*AwsAccountAccess, error) {
	client := sts.NewFromConfig(a.config)

	fmt.Printf("Enter MFA: ")
	reader := bufio.NewReader(os.Stdin)
	mfa, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read MFA")
	}

	input := &sts.GetSessionTokenInput{
		DurationSeconds: aws.Int32(int32(duration.Seconds())),
		SerialNumber:    aws.String(a.mfaSerial),
		TokenCode:       aws.String(strings.Trim(mfa, "\n")),
	}

	output, err := client.GetSessionToken(context.Background(), input)
	if err != nil {
		return nil, fmt.Errorf("failed to get session token: %w", err)
	}

	return &AwsAccountAccess{
		AccessKeyId:  *output.Credentials.AccessKeyId,
		SecretKey:    *output.Credentials.SecretAccessKey,
		SessionToken: *output.Credentials.SessionToken,
	}, nil
}

func (a *awsAbstraction) Region() string {
	return a.region
}

func (a *awsAbstraction) assumeRole(role, accountName string, mfa bool) (*AwsAccountAccess, error) {
	accountId, err := a.FindAccountId(accountName)
	if err != nil {
		return nil, fmt.Errorf("failed to get account id for account %q: %w", accountName, err)
	}

	hostname, err := os.Hostname()
	if err != nil {
		hostname = "Unknown"
	}

	client := sts.NewFromConfig(a.config)
	input := &sts.AssumeRoleInput{
		RoleArn:         aws.String(fmt.Sprintf("arn:aws:iam::%s:role/%s", accountId, role)),
		RoleSessionName: aws.String(fmt.Sprintf("IronCLI@%s", hostname)),
	}

	if mfa {
		mfaSerial, err := util.AskUser("Enter MFA Serial")
		if err != nil {
			return nil, fmt.Errorf("user did not enter mfa serial: %w", err)
		}

		token, err := util.AskUser("Enter Token")
		if err != nil {
			return nil, fmt.Errorf("user did not enter mfa token: %w", err)
		}

		input.SerialNumber = aws.String(mfaSerial)
		input.TokenCode = aws.String(token)
		log.Infof("Assuming role %s with mfa %s", *input.RoleArn, mfaSerial)
	} else {
		log.Infof("Assuming role %s", *input.RoleArn)
	}

	response, err := client.AssumeRole(context.Background(), input)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to assume role %s in account %s", role, accountId)
	}

	return &AwsAccountAccess{
		AccountId:    accountId,
		AccessKeyId:  *response.Credentials.AccessKeyId,
		SecretKey:    *response.Credentials.SecretAccessKey,
		SessionToken: *response.Credentials.SessionToken,
	}, nil
}
