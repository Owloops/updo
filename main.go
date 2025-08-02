package main

import (
	"fmt"
	"os"

	"github.com/Owloops/updo/cmd/aws"
	"github.com/Owloops/updo/cmd/monitor"
	"github.com/Owloops/updo/cmd/root"
	"github.com/spf13/cobra"
)

var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	versionStr := fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, date)
	root.RootCmd.Version = versionStr

	root.RootCmd.SetVersionTemplate("updo version {{.Version}}\n")

	root.RootCmd.AddCommand(monitor.MonitorCmd)
	root.RootCmd.AddCommand(aws.AWSCmd)

	root.RootCmd.Run = func(cmd *cobra.Command, args []string) {
		if len(args) == 0 && cmd.CalledAs() == "updo" {
			monitor.MonitorCmd.Run(cmd, args)
			return
		}

		if err := cmd.Help(); err != nil {
			fmt.Fprintf(os.Stderr, "Error displaying help: %v\n", err)
			os.Exit(1)
		}
	}

	if err := root.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
