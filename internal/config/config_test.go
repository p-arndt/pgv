package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	repoName := "testrepo"
	cfg := DefaultConfig(repoName)

	if cfg.RepoName != repoName {
		t.Errorf("Expected repo name %s, got %s", repoName, cfg.RepoName)
	}

	if cfg.Runtime != "docker" {
		t.Errorf("Expected runtime docker, got %s", cfg.Runtime)
	}

	if cfg.PostgresImage != "postgres:17" {
		t.Errorf("Expected postgres image postgres:17, got %s", cfg.PostgresImage)
	}

	if cfg.SnapshotDriver != "copydir" {
		t.Errorf("Expected snapshot driver copydir, got %s", cfg.SnapshotDriver)
	}

	if cfg.DefaultBranch != "main" {
		t.Errorf("Expected default branch main, got %s", cfg.DefaultBranch)
	}

	if cfg.BasePort != 5540 {
		t.Errorf("Expected base port 5540, got %d", cfg.BasePort)
	}
}

func TestLoadSaveConfig(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")

	cfg := DefaultConfig("testrepo")
	cfg.PgUser = "customuser"

	// Test SaveConfig
	err := SaveConfig(configPath, &cfg)
	if err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatalf("Config file was not created")
	}

	// Test LoadConfig
	loadedCfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// Compare some values
	if loadedCfg.RepoName != cfg.RepoName {
		t.Errorf("Expected repo name %s, got %s", cfg.RepoName, loadedCfg.RepoName)
	}

	if loadedCfg.PgUser != "customuser" {
		t.Errorf("Expected pg user customuser, got %s", loadedCfg.PgUser)
	}
}

func TestLoadConfig_FileNotFound(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "nonexistent.json")

	_, err := LoadConfig(configPath)
	if err == nil {
		t.Fatalf("Expected error when loading non-existent config file, got nil")
	}
}
