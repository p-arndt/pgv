package app

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"pgv/internal/metadata"
	"pgv/internal/services"
)

var stopCmd = &cobra.Command{
	Use:   "stop <branch-name>",
	Short: "Stop a running branch",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		branchName := args[0]

		_, db, repo, err := getRepoContext()
		if err != nil {
			return err
		}
		defer db.Close()

		var branch metadata.Branch
		if err := db.Get(&branch, "SELECT * FROM branches WHERE repo_id = ? AND name = ?", repo.ID, branchName); err != nil {
			return fmt.Errorf("branch '%s' not found: %w", branchName, err)
		}

		runtimeSvc, err := services.NewRuntimeService(db)
		if err != nil {
			return err
		}

		fmt.Printf("Stopping branch '%s'...\n", branchName)
		if err := runtimeSvc.StopBranch(context.Background(), branch.ID); err != nil {
			return err
		}

		fmt.Println("Branch stopped successfully.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(stopCmd)
}
