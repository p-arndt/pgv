package services

import (
	"path/filepath"
	"testing"

	"pgv/internal/config"
	"pgv/internal/metadata"
	"pgv/internal/util"
)

func TestRepoService_Init(t *testing.T) {
	tempDir := t.TempDir()

	// Create repo service
	svc := NewRepoService(tempDir)

	repoName := "test-repo"
	err := svc.Init(repoName, "")
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// 1. Verify layout was created
	pgvDir := filepath.Join(tempDir, ".pgv")
	if !util.Exists(pgvDir) {
		t.Fatalf("Expected .pgv dir to be created")
	}

	// 2. Verify config was saved
	configPath := filepath.Join(pgvDir, "config.json")
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load saved config: %v", err)
	}
	if cfg.RepoName != repoName {
		t.Errorf("Expected config repo name %s, got %s", repoName, cfg.RepoName)
	}

	// 3. Verify Metadata DB was created and records exist
	dbPath := filepath.Join(pgvDir, "meta", "state.db")
	if !util.Exists(dbPath) {
		t.Fatalf("Expected state.db to be created")
	}

	db, err := metadata.Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open metadata db: %v", err)
	}
	defer db.Close()

	// Check repo record
	var repos []metadata.Repo
	if err := db.Select(&repos, "SELECT * FROM repos"); err != nil {
		t.Fatalf("Failed to query repos: %v", err)
	}
	if len(repos) != 1 {
		t.Fatalf("Expected 1 repo record, got %d", len(repos))
	}
	if repos[0].Name != repoName {
		t.Errorf("Expected repo name %s, got %s", repoName, repos[0].Name)
	}

	// Check main branch record
	var branches []metadata.Branch
	if err := db.Select(&branches, "SELECT * FROM branches"); err != nil {
		t.Fatalf("Failed to query branches: %v", err)
	}
	if len(branches) != 1 {
		t.Fatalf("Expected 1 branch record, got %d", len(branches))
	}
	if branches[0].Name != cfg.DefaultBranch {
		t.Errorf("Expected branch name %s, got %s", cfg.DefaultBranch, branches[0].Name)
	}
	if !branches[0].IsHead {
		t.Errorf("Expected main branch to be head")
	}

	// Verify error on double Init
	err = svc.Init(repoName, "")
	if err == nil {
		t.Fatalf("Expected error when initializing already initialized repo")
	}
}
