package app

import (
	"github.com/spf13/cobra"
)

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
