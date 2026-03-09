package app

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"pgv/internal/services"
)

var checkpointCmd = &cobra.Command{
	Use:   "checkpoint <message>",
	Short: "Create a new snapshot from the active branch",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		label := args[0]
		cfg, db, repo, lock, err := getRepoContext()
		if err != nil {
			return err
		}
		defer lock.Unlock()
		defer db.Close()

		if repo.ActiveBranchID == "" {
			return fmt.Errorf("no active branch")
		}

		snapSvc, err := services.NewSnapshotService(db, repo.SnapshotDriver)
		if err != nil {
			return err
		}

		fmt.Printf("Creating checkpoint '%s'...\n", label)
		snapID, err := snapSvc.CreateCheckpoint(context.Background(), cfg, repo.ID, repo.ActiveBranchID, label)
		if err != nil {
			return err
		}

		fmt.Printf("Checkpoint created successfully: %s\n", snapID)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(checkpointCmd)
}
