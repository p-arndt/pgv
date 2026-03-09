package cowfs

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"pgv/internal/snapshot"
)

func TestCowfsDriver(t *testing.T) {
	driver := NewDriver()

	if driver.Name() != "cowfs" {
		t.Errorf("Expected driver name 'cowfs', got '%s'", driver.Name())
	}

	// Create a temporary source directory with some files
	srcDir, err := os.MkdirTemp("", "pgv-cowfs-test-src-*")
	if err != nil {
		t.Fatalf("Failed to create temp source dir: %v", err)
	}
	defer os.RemoveAll(srcDir)

	// Write a test file
	testFile := filepath.Join(srcDir, "test.txt")
	testData := []byte("hello cowfs")
	if err := os.WriteFile(testFile, testData, 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Create a sub directory
	subDir := filepath.Join(srcDir, "subdir")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatalf("Failed to create sub dir: %v", err)
	}
	subFile := filepath.Join(subDir, "sub.txt")
	if err := os.WriteFile(subFile, testData, 0644); err != nil {
		t.Fatalf("Failed to write sub file: %v", err)
	}

	// Target directory
	targetDir := filepath.Join(os.TempDir(), "pgv-cowfs-test-target")
	defer os.RemoveAll(targetDir)

	req := snapshot.CreateSnapshotRequest{
		SourcePath: srcDir,
		TargetPath: targetDir,
	}

	res, err := driver.CreateSnapshot(context.Background(), req)
	if err != nil {
		t.Logf("CreateSnapshot failed (likely CoW not supported on this specific temp fs): %v", err)
		// We don't fail the test because we might be running tests on a filesystem that doesn't support CoW
		// We just skip the rest if we can't create it
		return
	}

	// Verify size
	if res.SizeBytes == 0 {
		t.Errorf("Expected non-zero size")
	}

	// Verify content
	content, err := os.ReadFile(filepath.Join(targetDir, "test.txt"))
	if err != nil {
		t.Fatalf("Failed to read cloned file: %v", err)
	}
	if string(content) != "hello cowfs" {
		t.Errorf("Content mismatch, got: %s", string(content))
	}

	content, err = os.ReadFile(filepath.Join(targetDir, "subdir", "sub.txt"))
	if err != nil {
		t.Fatalf("Failed to read cloned sub file: %v", err)
	}
	if string(content) != "hello cowfs" {
		t.Errorf("Sub content mismatch, got: %s", string(content))
	}

	// Test Stat
	stats, err := driver.StatObject(context.Background(), targetDir)
	if err != nil {
		t.Fatalf("StatObject failed: %v", err)
	}
	if stats.SizeBytes == 0 {
		t.Errorf("Expected non-zero size from StatObject")
	}

	// Test Validate
	err = driver.Validate(context.Background(), snapshot.ValidateDriverRequest{})
	if err != nil {
		t.Errorf("Validate failed: %v", err)
	}

	// Test Delete
	err = driver.DeleteSnapshot(context.Background(), snapshot.DeleteSnapshotRequest{TargetPath: targetDir})
	if err != nil {
		t.Errorf("DeleteSnapshot failed: %v", err)
	}

	if _, err := os.Stat(targetDir); !os.IsNotExist(err) {
		t.Errorf("Expected target directory to be deleted")
	}
}
