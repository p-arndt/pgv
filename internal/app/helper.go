package app

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"pgv/internal/config"
	"pgv/internal/locks"
	"pgv/internal/metadata"
)

func getRepoContext() (*config.Config, *metadata.DB, *metadata.Repo, *locks.RepoLock, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, nil, nil, nil, err
	}

	pgvDir := filepath.Join(cwd, ".pgv")
	cfgPath := filepath.Join(pgvDir, "config.json")
	cfg, err := config.LoadConfig(cfgPath)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("could not load config, is this a pgv repo? %w", err)
	}

	lock, err := locks.AcquireRepoLock(cwd)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	dbPath := filepath.Join(pgvDir, "meta", "state.db")
	db, err := metadata.Open(dbPath)
	if err != nil {
		lock.Unlock()
		return nil, nil, nil, nil, fmt.Errorf("could not open metadata db: %w", err)
	}

	var repo metadata.Repo
	if err := db.Get(&repo, "SELECT * FROM repos LIMIT 1"); err != nil {
		db.Close()
		lock.Unlock()
		return nil, nil, nil, nil, fmt.Errorf("could not find repo record: %w", err)
	}

	return cfg, db, &repo, lock, nil
}

// getReadOnlyDBContext opens the DB without taking the repo lock, useful for autocompletion.
func getReadOnlyDBContext() (*metadata.DB, *metadata.Repo, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, nil, err
	}

	pgvDir := filepath.Join(cwd, ".pgv")
	dbPath := filepath.Join(pgvDir, "meta", "state.db")

	// If it doesn't exist, we're probably not in a repo
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return nil, nil, fmt.Errorf("not a pgv repo")
	}

	db, err := metadata.Open(dbPath)
	if err != nil {
		return nil, nil, err
	}

	var repo metadata.Repo
	if err := db.Get(&repo, "SELECT * FROM repos LIMIT 1"); err != nil {
		db.Close()
		return nil, nil, err
	}

	return db, &repo, nil
}

func getSnapshotsForCompletion() ([]string, cobra.ShellCompDirective) {
	db, repo, err := getReadOnlyDBContext()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	defer db.Close()

	var snapshots []metadata.Snapshot
	if err := db.Select(&snapshots, "SELECT id, label FROM snapshots WHERE repo_id = ? ORDER BY created_at DESC", repo.ID); err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	var comps []string
	for _, s := range snapshots {
		if s.Label != "" {
			comps = append(comps, fmt.Sprintf("%s\t%s", s.ID, s.Label))
		} else {
			comps = append(comps, s.ID)
		}
	}
	return comps, cobra.ShellCompDirectiveNoFileComp
}

func getBranchesForCompletion() ([]string, cobra.ShellCompDirective) {
	db, repo, err := getReadOnlyDBContext()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	defer db.Close()

	var branches []metadata.Branch
	if err := db.Select(&branches, "SELECT id, name FROM branches WHERE repo_id = ? ORDER BY created_at DESC", repo.ID); err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	var comps []string
	for _, b := range branches {
		comps = append(comps, fmt.Sprintf("%s\t%s", b.Name, b.ID))
	}
	return comps, cobra.ShellCompDirectiveNoFileComp
}
