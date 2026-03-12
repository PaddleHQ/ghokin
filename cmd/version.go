package cmd

import (
	"github.com/spf13/cobra"
)

var appVersion = ""

// versionCmd represents the version command.
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "App version",
	Run:   setupCmdFunc(version),
}

func version(msgHandler messageHandler, _ *cobra.Command, _ []string) {
	msgHandler.success(appVersion)
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
