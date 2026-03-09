package app

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"pgv/internal/metadata"
	"pgv/internal/services"
)

var rollbackCmd = &cobra.Command{
	Use:   "rollback",
	Short: "Quickly roll back the active branch to its most recent snapshot",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, db, repo, lock, err := getRepoContext()
		if err != nil {
			return err
		}
		defer lock.Unlock()
		defer db.Close()

		if repo.ActiveBranchID == "" {
			return fmt.Errorf("no active branch to rollback")
		}

		var branch metadata.Branch
		if err := db.Get(&branch, "SELECT * FROM branches WHERE id = ?", repo.ActiveBranchID); err != nil {
			return fmt.Errorf("could not load active branch: %w", err)
		}

		if branch.HeadSnapshotID == "" {
			return fmt.Errorf("branch '%s' has no snapshots to roll back to", branch.Name)
		}

		branchSvc, err := services.NewBranchService(db, repo.SnapshotDriver)
		if err != nil {
			return err
		}

		fmt.Printf("Rolling back branch '%s' to its latest snapshot (%s)...\n", branch.Name, branch.HeadSnapshotID)
		if err := branchSvc.RestoreBranch(context.Background(), cfg, branch.ID, branch.HeadSnapshotID); err != nil {
			return err
		}

		fmt.Println("Rollback complete.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(rollbackCmd)
}
