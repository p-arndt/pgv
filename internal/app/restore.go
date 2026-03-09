package app

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"pgv/internal/metadata"
	"pgv/internal/services"
)

var restoreBranch string

var restoreCmd = &cobra.Command{
	Use:   "restore <snapshot-id>",
	Short: "Replace branch writable state with chosen snapshot state",
	Args:  cobra.ExactArgs(1),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			return getSnapshotsForCompletion()
		}
		return nil, cobra.ShellCompDirectiveNoFileComp
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		snapshotID := args[0]

		cfg, db, repo, lock, err := getRepoContext()
		if err != nil {
			return err
		}
		defer lock.Unlock()
		defer db.Close()

		targetBranch := restoreBranch
		if targetBranch == "" {
			if repo.ActiveBranchID == "" {
				return fmt.Errorf("no branch specified and no active branch")
			}
			var branch metadata.Branch
			if err := db.Get(&branch, "SELECT * FROM branches WHERE id = ?", repo.ActiveBranchID); err != nil {
				return err
			}
			targetBranch = branch.Name
		}

		var branch metadata.Branch
		if err := db.Get(&branch, "SELECT * FROM branches WHERE repo_id = ? AND name = ?", repo.ID, targetBranch); err != nil {
			return fmt.Errorf("branch '%s' not found", targetBranch)
		}

		branchSvc, err := services.NewBranchService(db, repo.SnapshotDriver)
		if err != nil {
			return err
		}

		fmt.Printf("Restoring branch '%s' to snapshot '%s'...\n", targetBranch, snapshotID)
		if err := branchSvc.RestoreBranch(context.Background(), cfg, branch.ID, snapshotID); err != nil {
			return err
		}

		fmt.Println("Restore complete.")
		return nil
	},
}

func init() {
	restoreCmd.Flags().StringVar(&restoreBranch, "branch", "", "Target branch to restore (defaults to active branch)")
	_ = restoreCmd.RegisterFlagCompletionFunc("branch", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return getBranchesForCompletion()
	})
	rootCmd.AddCommand(restoreCmd)
}
