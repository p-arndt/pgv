package app

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"pgv/internal/services"
)

var checkoutCmd = &cobra.Command{
	Use:   "checkout <branch-name>",
	Short: "Switch the active branch",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		branchName := args[0]

		_, db, repo, lock, err := getRepoContext()
		if err != nil {
			return err
		}
		defer lock.Unlock()
		defer db.Close()

		branchSvc, err := services.NewBranchService(db, repo.SnapshotDriver)
		if err != nil {
			return err
		}

		fmt.Printf("Switching to branch '%s'...\n", branchName)
		if err := branchSvc.Checkout(context.Background(), repo.ID, branchName); err != nil {
			return err
		}

		fmt.Printf("Successfully switched to branch '%s'.\n", branchName)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(checkoutCmd)
}
