package app

import (
	"fmt"

	"github.com/spf13/cobra"
	"pgv/internal/metadata"
)

var urlCmd = &cobra.Command{
	Use:   "url [branch-name]",
	Short: "Display the connection URL for a branch (defaults to active branch)",
	Args:  cobra.MaximumNArgs(1),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			return getBranchesForCompletion()
		}
		return nil, cobra.ShellCompDirectiveNoFileComp
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, db, repo, lock, err := getRepoContext()
		if err != nil {
			return err
		}
		defer lock.Unlock()
		defer db.Close()

		branchName := ""
		if len(args) > 0 {
			branchName = args[0]
		} else {
			if repo.ActiveBranchID == "" {
				return fmt.Errorf("no branch specified and no active branch found")
			}
			var activeBranch metadata.Branch
			if err := db.Get(&activeBranch, "SELECT * FROM branches WHERE id = ?", repo.ActiveBranchID); err != nil {
				return fmt.Errorf("could not load active branch: %w", err)
			}
			branchName = activeBranch.Name
		}

		var branch metadata.Branch
		if err := db.Get(&branch, "SELECT * FROM branches WHERE repo_id = ? AND name = ?", repo.ID, branchName); err != nil {
			return fmt.Errorf("branch '%s' not found: %w", branchName, err)
		}

		url := fmt.Sprintf("postgres://%s:%s@127.0.0.1:%d/%s?sslmode=disable",
			cfg.PgUser, cfg.PgPassword, branch.Port, cfg.PgDatabase)

		fmt.Println(url)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(urlCmd)
}
