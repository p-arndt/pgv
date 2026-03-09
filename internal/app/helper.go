package app

import (
	"fmt"
	"os"
	"path/filepath"

	"pgv/internal/config"
	"pgv/internal/metadata"
)

func getRepoContext() (*config.Config, *metadata.DB, *metadata.Repo, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, nil, nil, err
	}

	pgvDir := filepath.Join(cwd, ".pgv")
	cfgPath := filepath.Join(pgvDir, "config.json")
	cfg, err := config.LoadConfig(cfgPath)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("could not load config, is this a pgv repo? %w", err)
	}

	dbPath := filepath.Join(pgvDir, "meta", "state.db")
	db, err := metadata.Open(dbPath)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("could not open metadata db: %w", err)
	}

	var repo metadata.Repo
	if err := db.Get(&repo, "SELECT * FROM repos LIMIT 1"); err != nil {
		return nil, nil, nil, fmt.Errorf("could not find repo record: %w", err)
	}

	return cfg, db, &repo, nil
}
