package ecr

import (
	"os"
	"os/exec"
	"strings"

	"github.com/apex/log"
	"github.com/spf13/cobra"
)

type ecrLoginOptions struct {
	authProfile string
	accountName string
	role        string
	region      string
	useDocker   bool
	usePodman   bool
}

func NewEcrLoginCommand() *cobra.Command {
	options := ecrLoginOptions{}
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Logs into an AWS ECR registry",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return ecrLogin(options)
		},
	}

	cmd.Flags().StringVarP(&options.authProfile, "profile", "p", "", "The AWS credentials profile to use")
	cmd.Flags().StringVarP(&options.accountName, "account", "a", "", "sets the account name")
	cmd.Flags().StringVarP(&options.role, "role", "r", "AdministratorAccess", "sets the role to assume")
	cmd.Flags().StringVar(&options.region, "region", "", "sets the region to use")
	cmd.Flags().BoolVarP(&options.usePodman, "podman", "", true, "Uses podman for logging in")
	cmd.Flags().BoolVarP(&options.useDocker, "docker", "", false, "Uses docker for logging in")

	return cmd
}

func ecrLogin(options ecrLoginOptions) error {
	creds, err := getCredentials(options.authProfile, options.role, options.accountName, options.region)
	if err != nil {
		return err
	}

	executableName := "podman"
	if options.useDocker {
		executableName = "docker"
	}

	log.Infof("logging into %s", creds.RegistryUrl)
	cmd := exec.Command(executableName, "login", "--username", creds.Username, "--password-stdin", creds.RegistryUrl)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = strings.NewReader(creds.Password)

	err = cmd.Run()

	return err
}
