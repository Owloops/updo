package list

import (
	"fmt"
	"strings"

	"github.com/Owloops/updo/aws"
	"github.com/Owloops/updo/cmd/root"
	"github.com/Owloops/updo/utils"
	"github.com/spf13/cobra"
)

var ListCmd = &cobra.Command{
	Use:   "list",
	Short: "List deployed Lambda functions by region",
	Long: `List all deployed updo Lambda functions across AWS regions.

This command discovers which regions currently have updo Lambda functions
deployed, showing the function ARN and deployment status for each region.

Use this to see which regions you can specify for monitoring or destruction.`,
	Example: `  updo aws list
  updo aws list --profile my-profile`,
	RunE: func(cmd *cobra.Command, args []string) error {
		profile, _ := cmd.Flags().GetString("profile")

		if profile != "" {
			utils.Log.Plain(fmt.Sprintf("Using AWS profile: %s", profile))
		}

		spinnerStop := make(chan bool)
		go utils.Log.Spinner("Checking deployed regions...", spinnerStop)

		regions, err := aws.GetDeployedRegions(profile)

		spinnerStop <- true
		close(spinnerStop)

		if err != nil {
			return fmt.Errorf("failed to discover deployed regions: %w", err)
		}

		if len(regions) == 0 {
			utils.Log.Info("No updo Lambda functions found in any region")
			utils.Log.Plain("Use 'updo aws deploy' to deploy Lambda functions")
			return nil
		}

		utils.Log.Success(fmt.Sprintf("Found updo Lambda functions in %d regions:", len(regions)))
		for _, region := range regions {
			utils.Log.Plain(fmt.Sprintf("  • %s", utils.Log.Region(region)))
		}

		utils.Log.Plain(fmt.Sprintf("\nTo monitor these regions: updo monitor --aws-regions %s", joinRegions(regions)))
		utils.Log.Plain("To destroy from all: updo aws destroy --regions all")

		return nil
	},
}

func joinRegions(regions []string) string {
	return strings.Join(regions, ",")
}

func init() {
	ListCmd.Flags().String("profile", "", "AWS profile to use")
	root.HideMonitoringFlags(ListCmd)
}
