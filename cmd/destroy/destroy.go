package destroy

import (
	"fmt"
	"strings"

	"github.com/Owloops/updo/aws"
	"github.com/Owloops/updo/cmd/root"
	"github.com/Owloops/updo/utils"
	"github.com/spf13/cobra"
)

var DestroyCmd = &cobra.Command{
	Use:   "destroy",
	Short: "Destroy Lambda functions from regions",
	Long: `Destroy Lambda functions from AWS regions.

By default, destroys from 13 regional edge caches:
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

This will remove the Lambda functions but keep the IAM role for future deployments.`,
	Example: `  updo destroy
  updo destroy --regions us-east-1,eu-west-1
  updo destroy -r us-east-1`,
	RunE: func(cmd *cobra.Command, args []string) error {
		regions, _ := cmd.Flags().GetStringSlice("regions")
		profile, _ := cmd.Flags().GetString("profile")
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		sequential, _ := cmd.Flags().GetBool("sequential")

		if dryRun {
			utils.Log.Info(fmt.Sprintf("Dry run: Would destroy Lambda functions from regions: %s", strings.Join(regions, ", ")))
			if profile != "" {
				utils.Log.Plain(fmt.Sprintf("AWS Profile: %s", profile))
			}
			utils.Log.Plain("Function: updo-executor-{region}")

			if aws.IsDestroyingAllDefaultRegions(regions) {
				utils.Log.Plain("IAM Role: updo-lambda-execution-role (will be deleted)")
			} else {
				utils.Log.Plain("IAM Role: updo-lambda-execution-role (will remain - not destroying from all regions)")
			}

			utils.Log.Plain("\nUse without --dry-run to execute destroy")
			return nil
		}

		if profile != "" {
			utils.Log.Plain(fmt.Sprintf("Using AWS profile: %s", profile))
		}

		results := aws.DestroyFromRegions(regions, aws.DeploymentOptions{
			Profile:    profile,
			Sequential: sequential,
		})

		successful := 0
		failed := 0
		var failedRegions []string

		for _, result := range results {
			if result.Success {
				successful++
			} else {
				failed++
				failedRegions = append(failedRegions, result.Region)
				utils.Log.Error(fmt.Sprintf("%s %s", utils.Log.Region(result.Region), result.Error))
			}
		}

		if failed > 0 {
			utils.Log.Plain(fmt.Sprintf("\nDestroy completed: %d successful, %d failed", successful, failed))
			utils.Log.Plain(fmt.Sprintf("Failed regions: %s", strings.Join(failedRegions, ", ")))
		} else {
			utils.Log.Success(fmt.Sprintf("Destroy completed successfully in all %d regions", successful))
		}

		if failed > 0 {
			return fmt.Errorf("destroy failed in %d regions", failed)
		}

		return nil
	},
}

func init() {
	defaultRegions := []string{
		"us-east-1", "us-west-1", "us-west-2", "eu-west-1", "eu-central-1",
		"ap-southeast-1", "ap-southeast-2", "ap-northeast-1", "ap-northeast-2",
		"ap-south-1", "sa-east-1", "ca-central-1", "eu-west-2",
	}
	DestroyCmd.Flags().StringSlice("regions", defaultRegions, "AWS regions to destroy from")
	DestroyCmd.Flags().String("profile", "", "AWS profile to use")
	DestroyCmd.Flags().Bool("dry-run", false, "Show what would be destroyed without executing")
	DestroyCmd.Flags().Bool("sequential", false, "Destroy from regions sequentially instead of parallel")

	root.HideMonitoringFlags(DestroyCmd)
}
