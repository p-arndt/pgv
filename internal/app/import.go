package app

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
	"pgv/internal/metadata"
)

var importCmd = &cobra.Command{
	Use:   "import <source-postgres-url>",
	Short: "Import data from an existing Postgres URL via logical dump into the active branch",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		sourceURL := args[0]

		cfg, db, repo, err := getRepoContext()
		if err != nil {
			return err
		}
		defer db.Close()

		if repo.ActiveBranchID == "" {
			return fmt.Errorf("no active branch to import into")
		}

		var branch metadata.Branch
		if err := db.Get(&branch, "SELECT * FROM branches WHERE id = ?", repo.ActiveBranchID); err != nil {
			return err
		}

		if branch.Status != "running" {
			return fmt.Errorf("active branch '%s' is not running. Please run 'pgv start %s' first", branch.Name, branch.Name)
		}

		var instance metadata.Instance
		if err := db.Get(&instance, "SELECT * FROM instances WHERE branch_id = ? AND status = 'running' LIMIT 1", branch.ID); err != nil {
			return fmt.Errorf("failed to find running container for branch '%s'", branch.Name)
		}

		fmt.Printf("Importing from %s into branch '%s'...\n", sourceURL, branch.Name)

		// We use docker exec to pipe pg_dump to psql entirely within the container
		// This avoids needing any postgres client binaries on the host machine!
		cmdStr := fmt.Sprintf(`pg_dump --clean --if-exists --no-owner --no-privileges -d '%s' | psql -U %s -d %s`, sourceURL, cfg.PgUser, cfg.PgDatabase)

		dockerCmd := exec.Command("docker", "exec", "-i", instance.ContainerName, "sh", "-c", cmdStr)
		dockerCmd.Stdout = os.Stdout
		dockerCmd.Stderr = os.Stderr

		if err := dockerCmd.Run(); err != nil {
			return fmt.Errorf("import failed: %w", err)
		}

		fmt.Println("Import completed successfully!")
		fmt.Printf("Tip: Create a checkpoint now to save this state: pgv checkpoint \"initial import\"\n")

		return nil
	},
}

func init() {
	rootCmd.AddCommand(importCmd)
}
