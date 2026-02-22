package commands

import (
	"context"

	"github.com/IronFE/iron.cli/terraform"
	"github.com/hashicorp/terraform-exec/tfexec"
	"github.com/pkg/errors"

	"github.com/spf13/cobra"
)

func NewPlanCommand() *cobra.Command {

	var options *terraform.CliOptions
	cmd := &cobra.Command{
		Use:   "plan",
		Short: "Executes Terraforms plan functionality",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			options.WorkDir = args[0]
			return plan(terraform.NewTerraformExecution(options))
		},
	}

	options = ApplyTerraformOptions(cmd)

	return cmd
}

func plan(execution terraform.ITerraformExecution) error {
	return execution.Execute(func(tf *tfexec.Terraform, options terraform.ExecutionOptions) error {
		var tfOpts []tfexec.PlanOption
		for _, f := range options.VariableFiles {
			tfOpts = append(tfOpts, tfexec.VarFile(f))
		}

		ok, err := tf.Plan(context.Background(), tfOpts...)
		if !ok {
			return errors.Wrap(err, "failed to run terraform plan")
		}

		return nil
	})
}
