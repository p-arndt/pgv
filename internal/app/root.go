package app

import (
	"github.com/spf13/cobra"
)

// Version is the current CLI version. It is set at build time via ldflags
// (see the GitHub Actions workflow) and defaults to "dev" when running in
// a normal development environment.
var Version = "dev"

var rootCmd = &cobra.Command{
	Use:   "pgv",
	Short: "PGV — Local Postgres Version Control",
	Long:  `PGV is a local developer tool that provides fast rollback, branching, and restore for PostgreSQL with data included.`,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Add commands here later
}
