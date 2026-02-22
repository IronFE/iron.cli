package ecr

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/IronFE/iron.cli/util/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type ecrCredsOptions struct {
	authProfile string
	accountName string
	role        string
	region      string
	format      string
}

func NewEcrCredsCommand() *cobra.Command {
	options := ecrCredsOptions{}
	cmd := &cobra.Command{
		Use:   "creds",
		Short: "Retrieves credentials for AWS ECR registry",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			formats := map[string]func(creds ecrCredentials) error{
				"text":  printText,
				"json":  printJson,
				"plain": printPassword,
			}

			printFunc, found := formats[options.format]
			if !found {
				return fmt.Errorf("unknown format option: %s", options.format)
			}

			creds, err := getCredentials(options.authProfile, options.role, options.accountName, options.region)
			if err != nil {
				return err
			}
			return printFunc(creds)
		},
	}

	cmd.Flags().StringVarP(&options.authProfile, "profile", "p", "", "The AWS credentials profile to use")
	cmd.Flags().StringVarP(&options.accountName, "account", "a", "", "sets the account name")
	cmd.Flags().StringVarP(&options.role, "role", "r", "AdministratorAccess", "sets the role to assume")
	cmd.Flags().StringVar(&options.region, "region", "", "sets the region to use")
	cmd.Flags().StringVarP(&options.format, "format", "f", "text", "Sets the format of the output. Allowed values are `plain` for pipeaple password, `json` and `text`")
	return cmd
}

func printJson(creds ecrCredentials) error {
	output, err := json.Marshal(creds)
	if err != nil {
		return fmt.Errorf("failed to marshal credentials: %w", err)
	}

	fmt.Println(string(output))
	return nil
}

func printText(creds ecrCredentials) error {
	fmt.Printf("Registry: \t%s\n", creds.RegistryUrl)
	fmt.Printf("Username: \t%s\n", creds.Username)
	fmt.Printf("Password: \t%s\n", creds.Password)
	return nil
}

func printPassword(creds ecrCredentials) error {
	fmt.Printf(creds.Password)
	return nil
}

type ecrCredentials struct {
	RegistryUrl string `json:"registry_url"`
	Username    string `json:"username"`
	Password    string `json:"password"`
}

func getCredentials(profile, role, accountName, region string) (ecrCredentials, error) {
	var err error
	var awsAbstraction aws.IAws
	awsAbstraction, err = aws.NewAws(profile)
	if err != nil {
		return ecrCredentials{}, err
	}

	access, err := awsAbstraction.AssumeRole(role, accountName)
	if err != nil {
		return ecrCredentials{}, fmt.Errorf("failed to assume role: %w", err)
	}

	if region == "" {
		region = awsAbstraction.Region()
	}

	config, err := aws.CreateConfig(access, region)
	if err != nil {
		return ecrCredentials{}, errors.Wrap(err, "failed to load default config")
	}

	client := ecr.NewFromConfig(config)
	input := &ecr.GetAuthorizationTokenInput{}

	output, err := client.GetAuthorizationToken(context.Background(), input)
	if err != nil {
		return ecrCredentials{}, fmt.Errorf("failed to get ecr authorization token: %w", err)
	}

	token := output.AuthorizationData[0].AuthorizationToken

	decoded, err := base64.StdEncoding.DecodeString(*token)
	if err != nil {
		return ecrCredentials{}, fmt.Errorf("failed to decode login token from AWS ECR: %w", err)
	}

	creds := strings.Split(string(decoded), ":")

	if len(creds) != 2 {
		return ecrCredentials{}, fmt.Errorf("failed to decode login token from AWS ECR: unexpected token format")
	}

	return ecrCredentials{
		Username:    creds[0],
		Password:    creds[1],
		RegistryUrl: fmt.Sprintf("%s.dkr.ecr.%s.amazonaws.com", access.AccountId, region),
	}, nil
}
