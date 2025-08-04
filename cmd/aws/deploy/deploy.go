package deploy

import (
	"fmt"
	"strings"

	"github.com/Owloops/updo/aws"
	"github.com/Owloops/updo/cmd/root"
	"github.com/Owloops/updo/utils"
	"github.com/spf13/cobra"
)

var _defaultRegions = []string{
	"us-east-1", "us-west-1", "us-west-2", "eu-west-1", "eu-central-1",
	"ap-southeast-1", "ap-southeast-2", "ap-northeast-1", "ap-northeast-2",
	"ap-south-1", "sa-east-1", "ca-central-1", "eu-west-2",
}

var DeployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploy Lambda functions for multi-region monitoring",
	Long: `Deploy Lambda functions to AWS regions for multi-region website monitoring.

By default, deploys to 13 regional edge caches:
- us-east-1 (US East - N. Virginia)
- us-west-1 (US West - N. California)
- us-west-2 (US West - Oregon)
- eu-west-1 (Europe - Ireland)
- eu-central-1 (Europe - Frankfurt)
- eu-west-2 (Europe - London)
- ap-southeast-1 (Asia Pacific - Singapore)
- ap-southeast-2 (Asia Pacific - Sydney)
- ap-northeast-1 (Asia Pacific - Tokyo)
- ap-northeast-2 (Asia Pacific - Seoul)
- ap-south-1 (Asia Pacific - Mumbai)
- sa-east-1 (South America - São Paulo)
- ca-central-1 (Canada - Central)

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

		if profile != "" {
			utils.Log.Plain(fmt.Sprintf("Using AWS profile: %s", profile))
		}

		results := aws.DeployToRegions(regions, aws.DeploymentOptions{
			Profile:    profile,
			Sequential: sequential,
		})

		successful, failed, failedRegions := processResults(results)

		if failed > 0 {
			utils.Log.Plain(fmt.Sprintf("\nDeployment completed: %d successful, %d failed", successful, failed))
			utils.Log.Plain(fmt.Sprintf("Failed regions: %s", strings.Join(failedRegions, ", ")))
		} else {
			utils.Log.Success(fmt.Sprintf("Deployment completed successfully in all %d regions", successful))
		}

		if failed > 0 {
			return fmt.Errorf("deployment failed in %d regions", failed)
		}

		return nil
	},
}

func processResults(results []aws.DeploymentResult) (successful, failed int, failedRegions []string) {
	for _, result := range results {
		if result.Success {
			successful++
		} else {
			failed++
			failedRegions = append(failedRegions, result.Region)
			utils.Log.Error(fmt.Sprintf("%s %s", utils.Log.Region(result.Region), result.Error))
		}
	}
	return
}

func init() {
	DeployCmd.Flags().StringSlice("regions", _defaultRegions, "AWS regions to deploy to")
	DeployCmd.Flags().String("profile", "", "AWS profile to use")
	DeployCmd.Flags().Bool("dry-run", false, "Show what would be deployed without executing")
	DeployCmd.Flags().Bool("sequential", false, "Deploy regions sequentially instead of parallel")

	root.HideMonitoringFlags(DeployCmd)
}
