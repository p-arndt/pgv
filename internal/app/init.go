package app

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"pgv/internal/services"
)

var initFromDir string

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
		if err := repoService.Init(repoName, initFromDir); err != nil {
			return err
		}

		fmt.Println("Initialization complete.")
		if initFromDir != "" {
			fmt.Println("Data successfully imported from", initFromDir)
		}
		return nil
	},
}

func init() {
	initCmd.Flags().StringVar(&initFromDir, "from-dir", "", "Import physical state from existing PGDATA directory")
	rootCmd.AddCommand(initCmd)
}
