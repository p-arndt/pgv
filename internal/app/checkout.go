package app

import (
	"context"
	"fmt"

	"pgv/internal/metadata"
	"pgv/internal/services"

	"github.com/spf13/cobra"
)

var checkoutParallel bool

var checkoutCmd = &cobra.Command{
	Use:     "checkout <branch-name>",
	Aliases: []string{"switch", "co"},
	Short:   "Switch the active branch and run it",
	Args:    cobra.ExactArgs(1),
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

		var targetBranch metadata.Branch
		if err := db.Get(&targetBranch, "SELECT * FROM branches WHERE repo_id = ? AND name = ?", repo.ID, branchName); err != nil {
			return fmt.Errorf("branch '%s' not found: %w", branchName, err)
		}

		branchSvc, err := services.NewBranchService(db, repo.SnapshotDriver)
		if err != nil {
			return err
		}

		runtimeSvc, err := services.NewRuntimeService(db)
		if err != nil {
			return err
		}

		if !checkoutParallel && repo.ActiveBranchID != "" && repo.ActiveBranchID != targetBranch.ID {
			var activeBranch metadata.Branch
			if err := db.Get(&activeBranch, "SELECT * FROM branches WHERE id = ?", repo.ActiveBranchID); err == nil && activeBranch.Status == "running" {
				fmt.Printf("Stopping active branch '%s'...\n", activeBranch.Name)
				if err := runtimeSvc.StopBranch(context.Background(), activeBranch.ID); err != nil {
					return fmt.Errorf("failed to stop active branch '%s': %w", activeBranch.Name, err)
				}
			}
		}

		fmt.Printf("Switching to branch '%s'...\n", branchName)
		if err := branchSvc.Checkout(context.Background(), repo.ID, branchName); err != nil {
			return err
		}

		if checkoutParallel {
			fmt.Printf("Starting checked out branch '%s' in parallel mode...\n", branchName)
		} else {
			fmt.Printf("Starting checked out branch '%s' on port %d...\n", branchName, cfg.BasePort)
		}
		if err := runtimeSvc.StartBranchWithOptions(context.Background(), targetBranch.ID, cfg, services.StartBranchOptions{Parallel: checkoutParallel}); err != nil {
			return fmt.Errorf("switched branch but failed to start '%s': %w", branchName, err)
		}

		if err := db.Get(&targetBranch, "SELECT * FROM branches WHERE id = ?", targetBranch.ID); err == nil {
			fmt.Printf("Branch '%s' is running on port %d.\n", branchName, targetBranch.Port)
		}

		fmt.Printf("Successfully switched to branch '%s'.\n", branchName)
		return nil
	},
}

func init() {
	checkoutCmd.Flags().BoolVar(&checkoutParallel, "parallel", false, "Keep currently running branches and start checked out branch on another available port")
	rootCmd.AddCommand(checkoutCmd)
}
