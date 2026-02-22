package aws

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sso"
	"github.com/aws/aws-sdk-go-v2/service/sso/types"
)

type identityCenterStrategy struct {
	authToken            string
	ssoProvider          *fileCachedAuthProvider
	startUrl             string
	defaultRole          string
	defaultRegion        string
	identityCenterRegion string
}

func NewIdentityCenterStrategy(startUrl, defaultRole, defaultRegion, identityCenterRegion string) IAws {
	if startUrl == "" {
		panic("parameter `startUrl` must not be empty")
	}
	if defaultRole == "" {
		panic("parameter `defaultRole` must not be empty")
	}
	if defaultRegion == "" {
		panic("default region cannot be empty")
	}
	if identityCenterRegion == "" {
		panic("parameter `identityCenterRegion` must not be empty")
	}

	return &identityCenterStrategy{
		defaultRole: defaultRole,
		ssoProvider: &fileCachedAuthProvider{
			startUrl: startUrl,
			region:   identityCenterRegion,
		},
		defaultRegion:        defaultRegion,
		identityCenterRegion: identityCenterRegion,
	}
}

func (s *identityCenterStrategy) AssumeRole(role, accountName string) (*AwsAccountAccess, error) {
	if role == "" {
		role = s.defaultRole
	}

	cfg, err := config.LoadDefaultConfig(context.Background(), config.WithDefaultRegion(s.identityCenterRegion))
	if err != nil {
		return nil, fmt.Errorf("failed to create default config: %w", err)
	}

	token, err := s.ssoAuth()
	if err != nil {
		return nil, fmt.Errorf("failed to authenticate to AWS: %w", err)
	}

	ssoClient := sso.NewFromConfig(cfg)
	accountInfo, err := s.findAccount(accountName, cfg)
	if err != nil {
		return nil, err
	}

	credsInput := sso.GetRoleCredentialsInput{
		AccessToken: aws.String(token),
		AccountId:   accountInfo.AccountId,
		RoleName:    aws.String(role),
	}

	credsOutput, err := ssoClient.GetRoleCredentials(context.Background(), &credsInput)
	if err != nil {
		return nil, fmt.Errorf("failed to get role credentials for account %s and role %s: %w", *accountInfo.AccountId, role, err)
	}
	return &AwsAccountAccess{
		AccountId:    *accountInfo.AccountId,
		AccessKeyId:  *credsOutput.RoleCredentials.AccessKeyId,
		SecretKey:    *credsOutput.RoleCredentials.SecretAccessKey,
		SessionToken: *credsOutput.RoleCredentials.SessionToken,
	}, nil

}

func (s *identityCenterStrategy) AssumeRoleWithMfa(role, accountName string) (*AwsAccountAccess, error) {
	return nil, fmt.Errorf("identity center does not support mfa")
}

func (s *identityCenterStrategy) FindAccountId(alias string) (string, error) {
	return "", fmt.Errorf("not supported")
}

func (s *identityCenterStrategy) OrganizationInfo() (AwsOrganizationInfo, error) {
	return AwsOrganizationInfo{}, nil
}

func (s *identityCenterStrategy) Region() string {
	return s.defaultRegion
}

func (s *identityCenterStrategy) SessionToken(duration time.Duration) (*AwsAccountAccess, error) {
	return nil, fmt.Errorf("a direct session token is not supported by the identity center strategy")
}

func (s *identityCenterStrategy) ssoAuth() (string, error) {
	if s.authToken != "" {
		return s.authToken, nil
	}

	authToken, err := s.ssoProvider.Auth()
	if err != nil {
		return "", fmt.Errorf("failed to authenticate: %w", err)
	}

	s.authToken = authToken
	return authToken, nil
}

func (s *identityCenterStrategy) findAccount(name string, cfg aws.Config) (types.AccountInfo, error) {
	token, err := s.ssoAuth()
	if err != nil {
		return types.AccountInfo{}, err
	}

	ssoClient := sso.NewFromConfig(cfg)
	accountPaginator := sso.NewListAccountsPaginator(ssoClient, &sso.ListAccountsInput{
		AccessToken: aws.String(token),
	})

	for accountPaginator.HasMorePages() {
		page, err := accountPaginator.NextPage(context.Background())
		if err != nil {
			return types.AccountInfo{}, fmt.Errorf("failed to get account page: %w", err)
		}

		for _, account := range page.AccountList {
			if strings.ToLower(*account.AccountName) == strings.ToLower(name) {
				return account, nil
			}
		}
	}
	return types.AccountInfo{}, fmt.Errorf("no aws account %q found", name)
}
