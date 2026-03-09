package app

import (
	"fmt"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the current PGV version",
	Run: func(cmd *cobra.Command, args []string) {
		// write to the command's output to allow tests to capture it via
		// rootCmd.SetOut in addition to normal stdout behavior.
		fmt.Fprintln(cmd.OutOrStdout(), Version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
