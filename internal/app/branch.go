package app

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"pgv/internal/metadata"
	"pgv/internal/services"
)

var branchCmd = &cobra.Command{
	Use:   "branch <source-snapshot> <branch-name>",
	Short: "Create a writable branch from a snapshot",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		source := args[0]
		branchName := args[1]

		_, db, repo, err := getRepoContext()
		if err != nil {
			return err
		}
		defer db.Close()

		branchSvc, err := services.NewBranchService(db, repo.SnapshotDriver)
		if err != nil {
			return err
		}

		// Resolve source snapshot
		var snap metadata.Snapshot
		if err := db.Get(&snap, "SELECT * FROM snapshots WHERE repo_id = ? AND id = ?", repo.ID, source); err != nil {
			// Fallback: try tag or label resolution later, for MVP require exact ID
			return fmt.Errorf("could not find snapshot '%s': %w", source, err)
		}

		fmt.Printf("Creating branch '%s' from snapshot '%s'...\n", branchName, source)
		branchID, err := branchSvc.CreateBranch(context.Background(), repo.ID, snap.ID, branchName)
		if err != nil {
			return err
		}

		fmt.Printf("Branch created successfully: %s\n", branchID)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(branchCmd)
}
