package app

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"pgv/internal/metadata"
)

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List branches and snapshots",
	RunE: func(cmd *cobra.Command, args []string) error {
		_, db, repo, lock, err := getRepoContext()
		if err != nil {
			return err
		}
		defer lock.Unlock()
		defer db.Close()

		var branches []metadata.Branch
		if err := db.Select(&branches, "SELECT * FROM branches WHERE repo_id = ?", repo.ID); err != nil {
			return err
		}

		fmt.Println("Branches:")
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		for _, b := range branches {
			active := " "
			if b.ID == repo.ActiveBranchID {
				active = "*"
			}
			fmt.Fprintf(w, "%s %s\tport %d\t%s\n", active, b.Name, b.Port, b.Status)
		}
		w.Flush()

		fmt.Println("\nSnapshots:")
		var snapshots []metadata.Snapshot
		if err := db.Select(&snapshots, "SELECT * FROM snapshots WHERE repo_id = ? ORDER BY created_at DESC", repo.ID); err != nil {
			return err
		}

		w2 := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		for _, s := range snapshots {
			fmt.Fprintf(w2, "  %s\t%s\t%s\n", s.ID, s.Label, s.CreatedAt.Format("2006-01-02 15:04:05"))
		}
		w2.Flush()

		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
