package commands

import (
	"github.com/IronFE/iron.cli/terraform"
	"github.com/spf13/cobra"
)

func ApplyTerraformOptions(command *cobra.Command) *terraform.CliOptions {
	optionset := terraform.CliOptions{}

	command.Flags().StringVarP(&optionset.AuthProfile, "profile", "p", "", "The AWS credentials profile to use")
	command.Flags().StringVarP(&optionset.TargetAccount, "account", "a", "", "Alias of the AWS Account to run the Terraform command on")
	command.Flags().StringVarP(&optionset.RoleToAssume, "role", "r", "", "The AWS role to assume")
	command.Flags().BoolVar(&optionset.NoRoleAssume, "no-assume", false, "Prevents any role assume and work directly with the user")
	command.MarkFlagRequired("account")
	command.Flags().BoolVar(&optionset.KeepTempDir, "keep-temp", false, "Keep the temp dir created during terraform operation")
	command.Flags().StringVarP(&optionset.DebugLevel, "debug", "d", "", "Sets the terraform log level. Valid values are: TRACE, DEBUG, INFO, WARN, ERROR. There is a bug, so keep the temp folder and look into it (https://github.com/hashicorp/terraform-exec/issues/436). See https://developer.hashicorp.com/terraform/internals/debugging")
	command.Flags().BoolVarP(&optionset.Mfa, "mfa", "m", false, "Asks for an MFA Token")
	command.Flags().StringVarP(&optionset.Variant, "variant", "v", "", "Put in variant of variables to set. An appropriate .tfvars file in the `variants` folder must be present.")
	command.Flags().StringVarP(&optionset.DeploymentName, "name", "n", "", "Sets the name of the deployment. If nothing is given, the name of folder the terraform files are in is used.")

	return &optionset
}
