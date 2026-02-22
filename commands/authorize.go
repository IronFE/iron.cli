package commands

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/IronFE/iron.cli/util/aws"
	"github.com/spf13/cobra"
)

type authorizeOptions struct {
	accountName  string
	args         []string
	authProfile  string
	credsOnly    bool
	interactive  bool
	role         string
	noRoleAssume bool
}

func NewAuthorizeCommand() *cobra.Command {
	options := authorizeOptions{}
	cmd := &cobra.Command{
		Use:   "authorize",
		Short: "Assume role in an account",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			options.args = args
			return authorize(options)
		},
	}

	cmd.Flags().StringVarP(&options.authProfile, "profile", "p", "", "The AWS credentials profile to use")
	cmd.Flags().StringVarP(&options.accountName, "account", "a", "", "sets the account name")
	cmd.Flags().StringVarP(&options.role, "role", "r", "", "sets the role to assume")
	cmd.Flags().BoolVarP(&options.noRoleAssume, "no-assume", "n", false, "Prevents any role assume and work directly with the user")
	cmd.Flags().BoolVarP(&options.credsOnly, "creds", "c", false, "Prints the credentials instead of executing a command")
	cmd.Flags().BoolVarP(&options.interactive, "interactive", "i", false, "Starts an interactive process")

	return cmd
}

func authorize(options authorizeOptions) error {
	var err error
	var awsAbstraction aws.IAws
	awsAbstraction, err = aws.NewAws(options.authProfile)
	if err != nil {
		return err
	}

	var credentials *aws.AwsAccountAccess

	if options.noRoleAssume {
		credentials, err = awsAbstraction.SessionToken(30 * time.Minute)
		if err != nil {
			return fmt.Errorf("failed to get session token for current user: %w", err)
		}
	} else {
		credentials, err = awsAbstraction.AssumeRole(options.role, options.accountName)
		if err != nil {
			return fmt.Errorf("failed to assume role %q in %q: %w", options.role, options.accountName, err)
		}
	}

	if options.credsOnly {
		fmt.Printf("AWS_ACCESS_KEY_ID=%s\nAWS_SECRET_ACCESS_KEY=%s\nAWS_SESSION_TOKEN=%s\n", credentials.AccessKeyId, credentials.SecretKey, credentials.SessionToken)
		return nil
	}

	cmd := exec.Command(options.args[0], options.args[1:]...)

	envs := os.Environ()

	allEnvs := make([]string, len(envs)+3)
	copy(allEnvs, envs)

	allEnvs[len(envs)] = fmt.Sprintf("%s=%s", "AWS_ACCESS_KEY_ID", credentials.AccessKeyId)
	allEnvs[len(envs)+1] = fmt.Sprintf("%s=%s", "AWS_SECRET_ACCESS_KEY", credentials.SecretKey)
	allEnvs[len(envs)+2] = fmt.Sprintf("%s=%s", "AWS_SESSION_TOKEN", credentials.SessionToken)

	cmd.Env = allEnvs
	if options.interactive {
		cmd.Stdin = os.Stdin
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()

	return err
}
