package commands

import (
	"context"

	"github.com/IronFE/iron.cli/terraform"
	"github.com/hashicorp/terraform-exec/tfexec"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func NewOutputCommand() *cobra.Command {
	var options *terraform.CliOptions
	cmd := &cobra.Command{
		Use:   "output",
		Short: "Prints the outputs of Terraform deployment",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			options.WorkDir = args[0]
			return output(terraform.NewTerraformExecution(options))
		},
	}
	options = ApplyTerraformOptions(cmd)

	return cmd
}

func output(execution terraform.ITerraformExecution) error {
	return execution.Execute(func(tf *tfexec.Terraform, execOptions terraform.ExecutionOptions) error {
		_, err := tf.Output(context.Background())
		if err != nil {
			return errors.Wrap(err, "failed to run terraform output")
		}

		return nil
	})
}
