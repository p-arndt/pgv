package app

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"pgv/internal/metadata"
	"pgv/internal/services"
)

var startCmd = &cobra.Command{
	Use:   "start <branch-name>",
	Short: "Start a branch as a Postgres container/process",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		branchName := args[0]

		cfg, db, repo, lock, err := getRepoContext()
		if err != nil {
			return err
		}
		defer lock.Unlock()
		defer db.Close()

		var branch metadata.Branch
		if err := db.Get(&branch, "SELECT * FROM branches WHERE repo_id = ? AND name = ?", repo.ID, branchName); err != nil {
			return fmt.Errorf("branch '%s' not found: %w", branchName, err)
		}

		runtimeSvc, err := services.NewRuntimeService(db)
		if err != nil {
			return err
		}

		fmt.Printf("Starting branch '%s'...\n", branchName)
		if err := runtimeSvc.StartBranch(context.Background(), branch.ID, cfg); err != nil {
			return err
		}

		fmt.Printf("Branch '%s' started on port %d.\n", branchName, branch.Port)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
}
