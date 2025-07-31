package aws

import (
	"github.com/spf13/cobra"

	"github.com/Owloops/updo/cmd/aws/deploy"
	"github.com/Owloops/updo/cmd/aws/destroy"
	"github.com/Owloops/updo/cmd/aws/list"
	"github.com/Owloops/updo/cmd/root"
)

var AWSCmd = &cobra.Command{
	Use:   "aws",
	Short: "AWS Lambda operations for multi-region monitoring",
	Long: `Manage AWS Lambda functions for multi-region website monitoring.

This command group provides operations to deploy, destroy, and list
Lambda functions across multiple AWS regions for distributed monitoring.`,
	Example: `  updo aws deploy
  updo aws deploy --regions us-east-1,eu-west-1
  updo aws destroy --regions all
  updo aws list`,
}

func init() {
	AWSCmd.AddCommand(deploy.DeployCmd)
	AWSCmd.AddCommand(destroy.DestroyCmd)
	AWSCmd.AddCommand(list.ListCmd)

	root.HideMonitoringFlags(AWSCmd)
}
