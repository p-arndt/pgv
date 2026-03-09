package app

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"pgv/internal/services"
)

var initCmd = &cobra.Command{
	Use:   "init [repo-name]",
	Short: "Initialize a new PGV repository",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}

		repoName := filepath.Base(cwd)
		if len(args) > 0 {
			repoName = args[0]
		}

		fmt.Printf("Initializing repository '%s' in %s\n", repoName, cwd)
		repoService := services.NewRepoService(cwd)
		if err := repoService.Init(repoName); err != nil {
			return err
		}

		fmt.Println("Initialization complete.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
