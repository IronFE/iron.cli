package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/IronFE/iron.cli/terraform"
	"github.com/IronFE/iron.cli/util"
	"github.com/apex/log"
	"github.com/hashicorp/terraform-exec/tfexec"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type destroyOptions struct {
	Confirm bool
}

func NewDestroyCommand() *cobra.Command {
	var terraformOptions *terraform.CliOptions
	var options = &destroyOptions{}
	cmd := &cobra.Command{
		Use:   "destroy",
		Short: "Destroys resources via terraform",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			terraformOptions.WorkDir = args[0]
			return destroy(terraform.NewTerraformExecution(terraformOptions), options)
		},
	}
	terraformOptions = ApplyTerraformOptions(cmd)
	cmd.Flags().BoolVarP(&options.Confirm, "confirm", "c", false, "Stops terraform after planning")
	return cmd
}

func destroy(execution terraform.ITerraformExecution, options *destroyOptions) error {
	return execution.Execute(func(tf *tfexec.Terraform, execOptions terraform.ExecutionOptions) error {

		var applyFunc func() error
		if options.Confirm {
			planOpts := []tfexec.PlanOption{
				tfexec.Destroy(true),
				tfexec.Out("plan"),
			}
			for _, f := range execOptions.VariableFiles {
				planOpts = append(planOpts, tfexec.VarFile(f))
			}

			changes, err := tf.Plan(context.Background(), planOpts...)
			if err != nil {
				return fmt.Errorf("failed to run terraform plan: %w", err)
			}

			if !changes {
				log.Info("No changes in plan")
				return nil
			}

			response, err := util.AskUser("Apply plan? (y, n)")
			if err != nil {
				return fmt.Errorf("user did not respond: %w", err)
			}
			normalizedResponse := strings.ToLower(response)

			if normalizedResponse != "y" && normalizedResponse != "yes" {
				return errors.Errorf("user aborted deployment")
			}

			applyFunc = func() error {
				applyOpts := []tfexec.ApplyOption{tfexec.DirOrPlan("plan")}
				for _, f := range execOptions.VariableFiles {
					applyOpts = append(applyOpts, tfexec.VarFile(f))
				}
				return tf.Apply(context.Background(), applyOpts...)
			}
		} else {
			var destroyOpts []tfexec.DestroyOption
			for _, f := range execOptions.VariableFiles {
				destroyOpts = append(destroyOpts, tfexec.VarFile(f))
			}
			applyFunc = func() error {
				return tf.Destroy(context.Background(), destroyOpts...)
			}
		}

		err := applyFunc()

		if err != nil {
			return errors.Wrap(err, "failed to run terraform apply")
		}
		return nil

	})
}
