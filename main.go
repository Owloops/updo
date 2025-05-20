package main

import (
	"fmt"
	"os"

	"github.com/Owloops/updo/cmd/monitor"
	"github.com/Owloops/updo/cmd/root"
	"github.com/spf13/cobra"
)

func main() {
	root.RootCmd.AddCommand(monitor.MonitorCmd)

	root.RootCmd.Run = func(cmd *cobra.Command, args []string) {
		if len(args) == 0 && cmd.CalledAs() == "updo" {
			monitor.MonitorCmd.Run(cmd, args)
			return
		}
		cmd.Help()
	}

	if err := root.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
