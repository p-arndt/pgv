package util

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureDirAndExists(t *testing.T) {
	tempDir := t.TempDir()

	testPath := filepath.Join(tempDir, "test_ensure_dir")

	// Should not exist initially
	if Exists(testPath) {
		t.Fatalf("Expected %s to not exist", testPath)
	}

	// Create directory
	if err := EnsureDir(testPath); err != nil {
		t.Fatalf("EnsureDir failed: %v", err)
	}

	// Should exist now
	if !Exists(testPath) {
		t.Fatalf("Expected %s to exist after EnsureDir", testPath)
	}

	// Should be a directory
	info, err := os.Stat(testPath)
	if err != nil {
		t.Fatalf("Failed to stat path: %v", err)
	}
	if !info.IsDir() {
		t.Fatalf("Expected %s to be a directory", testPath)
	}

	// Calling EnsureDir again should not fail
	if err := EnsureDir(testPath); err != nil {
		t.Fatalf("EnsureDir failed on existing directory: %v", err)
	}
}
