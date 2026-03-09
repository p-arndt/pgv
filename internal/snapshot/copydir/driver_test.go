package copydir

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"pgv/internal/snapshot"
)

func TestDriver_Operations(t *testing.T) {
	driver := NewDriver()
	ctx := context.Background()

	// 1. Setup a dummy source directory with a file
	srcDir := t.TempDir()
	testFile := filepath.Join(srcDir, "test.txt")
	content := []byte("hello world")
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// 2. Test CreateSnapshot
	snapshotDir := filepath.Join(t.TempDir(), "snap")
	createReq := snapshot.CreateSnapshotRequest{
		SourcePath: srcDir,
		TargetPath: snapshotDir,
	}
	res, err := driver.CreateSnapshot(ctx, createReq)
	if err != nil {
		t.Fatalf("CreateSnapshot failed: %v", err)
	}
	if res.SizeBytes != int64(len(content)) {
		t.Errorf("Expected snapshot size %d, got %d", len(content), res.SizeBytes)
	}

	// Verify file was copied
	copiedFile := filepath.Join(snapshotDir, "test.txt")
	copiedData, err := os.ReadFile(copiedFile)
	if err != nil {
		t.Fatalf("Failed to read copied file: %v", err)
	}
	if string(copiedData) != string(content) {
		t.Errorf("Expected file content %q, got %q", string(content), string(copiedData))
	}

	// 3. Test StatObject
	stats, err := driver.StatObject(ctx, snapshotDir)
	if err != nil {
		t.Fatalf("StatObject failed: %v", err)
	}
	if stats.SizeBytes != int64(len(content)) {
		t.Errorf("Expected stat size %d, got %d", len(content), stats.SizeBytes)
	}

	// 4. Test CloneSnapshotToBranch
	branchDir := filepath.Join(t.TempDir(), "branch")
	cloneReq := snapshot.CloneSnapshotRequest{
		SourcePath: snapshotDir,
		TargetPath: branchDir,
	}
	_, err = driver.CloneSnapshotToBranch(ctx, cloneReq)
	if err != nil {
		t.Fatalf("CloneSnapshotToBranch failed: %v", err)
	}

	// Verify file was copied to branch
	branchFile := filepath.Join(branchDir, "test.txt")
	branchData, err := os.ReadFile(branchFile)
	if err != nil {
		t.Fatalf("Failed to read branch file: %v", err)
	}
	if string(branchData) != string(content) {
		t.Errorf("Expected branch file content %q, got %q", string(content), string(branchData))
	}

	// 5. Test DeleteBranchData
	delBranchReq := snapshot.DeleteBranchDataRequest{TargetPath: branchDir}
	if err := driver.DeleteBranchData(ctx, delBranchReq); err != nil {
		t.Fatalf("DeleteBranchData failed: %v", err)
	}
	if _, err := os.Stat(branchDir); !os.IsNotExist(err) {
		t.Errorf("Expected branch directory to be deleted")
	}

	// 6. Test DeleteSnapshot
	delSnapReq := snapshot.DeleteSnapshotRequest{TargetPath: snapshotDir}
	if err := driver.DeleteSnapshot(ctx, delSnapReq); err != nil {
		t.Fatalf("DeleteSnapshot failed: %v", err)
	}
	if _, err := os.Stat(snapshotDir); !os.IsNotExist(err) {
		t.Errorf("Expected snapshot directory to be deleted")
	}

	// 7. Test Name and Validate
	if driver.Name() != "copydir" {
		t.Errorf("Expected driver name 'copydir', got '%s'", driver.Name())
	}
	if err := driver.Validate(ctx, snapshot.ValidateDriverRequest{}); err != nil {
		t.Errorf("Validate failed: %v", err)
	}
}
