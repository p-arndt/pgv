package app

import (
	"os"
	"path/filepath"
	"testing"

	"pgv/internal/services"
	"pgv/internal/util"
)

func TestGetRepoContext(t *testing.T) {
	// 1. Setup a dummy repo in a temp dir
	tempDir := t.TempDir()

	repoSvc := services.NewRepoService(tempDir)
	if err := repoSvc.Init("test-repo", ""); err != nil {
		t.Fatalf("Failed to init repo: %v", err)
	}

	// 2. Change working directory to temp dir
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get orig wd: %v", err)
	}
	defer os.Chdir(origWd) // Restore original wd after test

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to chdir: %v", err)
	}

	// 3. Test getRepoContext
	cfg, db, repo, lock, err := getRepoContext()
	if err != nil {
		t.Fatalf("getRepoContext failed: %v", err)
	}

	if cfg == nil {
		t.Errorf("Expected config to be loaded")
	} else if cfg.RepoName != "test-repo" {
		t.Errorf("Expected config repo name to be test-repo, got %s", cfg.RepoName)
	}

	if db == nil {
		t.Errorf("Expected db to be loaded")
	} else {
		db.Close()
	}

	if repo == nil {
		t.Errorf("Expected repo to be loaded")
	} else if repo.Name != "test-repo" {
		t.Errorf("Expected repo name to be test-repo, got %s", repo.Name)
	}

	if lock == nil {
		t.Errorf("Expected lock to be acquired")
	} else {
		lock.Unlock()
	}

	// 4. Test missing repo
	emptyDir := t.TempDir()
	if err := os.Chdir(emptyDir); err != nil {
		t.Fatalf("Failed to chdir to empty dir: %v", err)
	}

	_, _, _, _, err = getRepoContext()
	if err == nil {
		t.Fatalf("Expected error when calling getRepoContext in empty dir")
	}
}

func TestAppHelpers(t *testing.T) {
	// Basic tests to verify other helper functions/structs if they exist.
	// Currently getRepoContext is the main one.

	// Ensure that path construction and util usage handles edge cases well
	path := filepath.Join(t.TempDir(), "nonexistent", ".pgv")
	if util.Exists(path) {
		t.Errorf("Expected %s not to exist", path)
	}
}
