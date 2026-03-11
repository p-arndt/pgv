package app

import (
	"context"
	"fmt"

	"pgv/internal/metadata"
	"pgv/internal/services"

	"github.com/spf13/cobra"
)

var startParallel bool

var startCmd = &cobra.Command{
	Use:   "start <branch-name>",
	Short: "Start a branch as a Postgres container/process",
	Args:  cobra.ExactArgs(1),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			return getBranchesForCompletion()
		}
		return nil, cobra.ShellCompDirectiveNoFileComp
	},
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
		if err := runtimeSvc.StartBranchWithOptions(context.Background(), branch.ID, cfg, services.StartBranchOptions{Parallel: startParallel}); err != nil {
			return err
		}

		if err := db.Get(&branch, "SELECT * FROM branches WHERE id = ?", branch.ID); err != nil {
			return fmt.Errorf("branch started but failed to load updated port: %w", err)
		}
		fmt.Printf("Branch '%s' started on port %d.\n", branchName, branch.Port)
		return nil
	},
}

func init() {
	startCmd.Flags().BoolVar(&startParallel, "parallel", false, "Start branch in parallel without stopping currently running branches")
	rootCmd.AddCommand(startCmd)
}
