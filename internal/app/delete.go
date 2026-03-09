package app

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"pgv/internal/services"
)

var forceDelete bool

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete resources (branches, snapshots)",
}

var deleteBranchCmd = &cobra.Command{
	Use:   "branch <branch-name>",
	Short: "Delete a branch and its data",
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

		fmt.Printf("Deleting branch '%s'...\n", branchName)
		if err := branchSvc.DeleteBranch(context.Background(), repo.ID, branchName, forceDelete); err != nil {
			return err
		}

		fmt.Printf("Successfully deleted branch '%s'.\n", branchName)
		return nil
	},
}

func init() {
	deleteBranchCmd.Flags().BoolVarP(&forceDelete, "force", "f", false, "Force delete an active or running branch")
	deleteCmd.AddCommand(deleteBranchCmd)
	rootCmd.AddCommand(deleteCmd)
}
