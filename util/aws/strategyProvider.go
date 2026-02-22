package aws

import (
	"fmt"
	"time"

	"github.com/IronFE/iron.cli/util/config"
)

type AwsAccountAccess struct {
	AccountId    string
	AccessKeyId  string
	SecretKey    string
	SessionToken string
}

type IAws interface {
	FindAccountId(alias string) (string, error)
	OrganizationInfo() (AwsOrganizationInfo, error)
	AssumeRole(role, accountName string) (*AwsAccountAccess, error)
	AssumeRoleWithMfa(role, accountName string) (*AwsAccountAccess, error)
	SessionToken(duration time.Duration) (*AwsAccountAccess, error)
	Region() string
}

func NewAws(profileName string) (IAws, error) {
	profileProvider := config.NewProfileProvider()
	var selectedProfile config.Profile
	var err error
	if profileName != "" {
		selectedProfile, err = profileProvider.Profile(profileName)
		if err != nil {
			return nil, fmt.Errorf("could not load profile %s: %w", profileName, err)
		}
	} else {
		selectedProfile, err = profileProvider.DefaultProfile()
		if err != nil {
			return nil, fmt.Errorf("could not load default profile: %w", err)
		}
	}

	switch selectedProfile.AuthStrategy {
	case "identityCenter":
		idcRegion := selectedProfile.IdentityCenter.Region
		if idcRegion == "" {
			idcRegion = selectedProfile.DefaultRegion
		}

		return NewIdentityCenterStrategy(
			selectedProfile.IdentityCenter.StartUrl,
			selectedProfile.IdentityCenter.DefaultRole,
			selectedProfile.DefaultRegion,
			idcRegion), nil
	case "iam":
		strategy, err := newCredsStrategyAws(selectedProfile.IAM.ProfileName, selectedProfile.DefaultRegion, selectedProfile.IAM.MfaSerial)
		if err != nil {
			return nil, fmt.Errorf("failed to create iam strategy: %w", err)
		}
		return strategy, nil
	default:
		return nil, fmt.Errorf("no strategy %q found", selectedProfile.AuthStrategy)
	}

}
