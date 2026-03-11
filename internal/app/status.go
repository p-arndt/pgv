package app

import (
	"fmt"

	"pgv/internal/metadata"

	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:     "status",
	Aliases: []string{"st"},
	Short:   "Display the status of the active branch and repository",
	Args:    cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		_, db, repo, lock, err := getRepoContext()
		if err != nil {
			return err
		}
		defer lock.Unlock()
		defer db.Close()

		fmt.Printf("Repository: %s\n", repo.Name)

		if repo.ActiveBranchID == "" {
			fmt.Println("Active Branch: None")
			return nil
		}

		var branch metadata.Branch
		if err := db.Get(&branch, "SELECT * FROM branches WHERE id = ?", repo.ActiveBranchID); err != nil {
			return fmt.Errorf("could not load active branch: %w", err)
		}

		fmt.Printf("Active Branch: %s\n", branch.Name)
		fmt.Printf("Status: %s\n", branch.Status)

		if branch.Status == "running" {
			fmt.Printf("Port: %d\n", branch.Port)
			// Could fetch instance info if needed
		}

		if branch.HeadSnapshotID != "" {
			var snap metadata.Snapshot
			if err := db.Get(&snap, "SELECT * FROM snapshots WHERE id = ?", branch.HeadSnapshotID); err == nil {
				fmt.Printf("Latest Snapshot: %s (%s)\n", snap.ID, snap.Label)
			}
		} else {
			fmt.Println("Latest Snapshot: None")
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
