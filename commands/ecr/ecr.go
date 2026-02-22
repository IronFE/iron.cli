package ecr

import "github.com/spf13/cobra"

func NewEcrCommand() *cobra.Command {
	var baseCommand = &cobra.Command{
		Use:   "ecr",
		Short: "Interacts with an AWS Elastic Container Registry",
	}

	baseCommand.AddCommand(NewEcrLoginCommand())
	baseCommand.AddCommand(NewEcrCredsCommand())
	return baseCommand
}
