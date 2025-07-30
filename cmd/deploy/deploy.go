package deploy

import (
	"fmt"
	"strings"

	"github.com/Owloops/updo/aws"
	"github.com/Owloops/updo/cmd/root"
	"github.com/Owloops/updo/utils"
	"github.com/spf13/cobra"
)

var DeployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploy Lambda functions for multi-region monitoring",
	Long: `Deploy Lambda functions to AWS regions for multi-region website monitoring.

By default, deploys to these regions:
- us-east-1 (US East - N. Virginia)
- us-west-2 (US West - Oregon)  
- eu-central-1 (Europe - Frankfurt)
- ap-southeast-1 (Asia Pacific - Singapore)

Requires AWS credentials to be configured (AWS CLI, environment variables, or IAM roles).`,
	Example: `  updo deploy
  updo deploy --regions us-east-1,eu-west-1
  updo deploy -r us-east-1`,
	RunE: func(cmd *cobra.Command, args []string) error {
		regions, _ := cmd.Flags().GetStringSlice("regions")
		profile, _ := cmd.Flags().GetString("profile")
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		sequential, _ := cmd.Flags().GetBool("sequential")

		if dryRun {
			utils.Log.Info(fmt.Sprintf("Dry run: Would deploy Lambda functions to regions: %s", strings.Join(regions, ", ")))
			if profile != "" {
				utils.Log.Plain(fmt.Sprintf("AWS Profile: %s", profile))
			}
			utils.Log.Plain("Function: updo-executor-{region}")
			utils.Log.Plain("IAM Role: updo-lambda-execution-role (created if not exists)")
			utils.Log.Plain("Runtime: provided.al2023 (ARM64)")
			utils.Log.Plain("Memory: 128 MB, Timeout: 30s")
			utils.Log.Plain("\nUse without --dry-run to execute deployment")
			return nil
		}

		utils.Log.Info(fmt.Sprintf("Deploying Lambda functions to regions: %s", strings.Join(regions, ", ")))
		if profile != "" {
			utils.Log.Plain(fmt.Sprintf("Using AWS profile: %s", profile))
		}
		if sequential {
			utils.Log.Plain("Deploying sequentially")
		}

		results := aws.DeployToRegions(regions, aws.DeploymentOptions{
			Profile:    profile,
			Sequential: sequential,
		})

		successful := 0
		failed := 0

		for _, result := range results {
			regionStr := utils.Log.Region(result.Region)
			if result.Success {
				utils.Log.Success(fmt.Sprintf("%s %s", regionStr, result.FunctionArn))
				successful++
			} else {
				utils.Log.Error(fmt.Sprintf("%s %s", regionStr, result.Error))
				failed++
			}
		}

		utils.Log.Plain(fmt.Sprintf("\nDeployment completed: %d successful, %d failed", successful, failed))

		if failed > 0 {
			return fmt.Errorf("deployment failed in %d regions", failed)
		}

		return nil
	},
}

func init() {
	defaultRegions := []string{"us-east-1", "us-west-2", "eu-central-1", "ap-southeast-1"}
	DeployCmd.Flags().StringSlice("regions", defaultRegions, "AWS regions to deploy to")
	DeployCmd.Flags().String("profile", "", "AWS profile to use")
	DeployCmd.Flags().Bool("dry-run", false, "Show what would be deployed without executing")
	DeployCmd.Flags().Bool("sequential", false, "Deploy regions sequentially instead of parallel")

	root.HideMonitoringFlags(DeployCmd)
}
