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
	RunE: func(cmd *cobra.Command, args []string) error {
		snapshotID := args[0]

		_, db, repo, err := getRepoContext()
		if err != nil {
			return err
		}
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
		if err := branchSvc.RestoreBranch(context.Background(), branch.ID, snapshotID); err != nil {
			return err
		}

		fmt.Println("Restore complete.")
		return nil
	},
}

func init() {
	restoreCmd.Flags().StringVar(&restoreBranch, "branch", "", "Target branch to restore (defaults to active branch)")
	rootCmd.AddCommand(restoreCmd)
}
