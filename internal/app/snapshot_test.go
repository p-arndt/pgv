package app

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"pgv/internal/metadata"
)

func TestSnapshotCmd_DeleteMode(t *testing.T) {
	tempDir, db, repo := setupTestRepo(t)
	defer db.Close()

	origWd, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(origWd)

	now := time.Now().UTC()
	snapID := "snap_for_delete"
	snap := metadata.Snapshot{
		ID:               snapID,
		RepoID:           repo.ID,
		ParentSnapshotID: nil,
		SourceBranchID:   nil,
		Label:            "snapshot-delete",
		Kind:             "checkpoint",
		DataPath:         filepath.Join(tempDir, ".pgv", "storage", "snapshots", snapID),
		DriverType:       "copydir",
		RestorePointName: "rp_delete",
		LSN:              "0/0",
		SizeBytes:        0,
		CreatedAt:        now,
	}

	if _, err := db.NamedExec(`INSERT INTO snapshots (id, repo_id, parent_snapshot_id, source_branch_id, label, kind, data_path, driver_type, restore_point_name, lsn, size_bytes, created_at)
		VALUES (:id, :repo_id, :parent_snapshot_id, :source_branch_id, :label, :kind, :data_path, :driver_type, :restore_point_name, :lsn, :size_bytes, :created_at)`, snap); err != nil {
		t.Fatalf("Failed to insert snapshot: %v", err)
	}

	out, err := executeCommand(rootCmd, "snapshot", "-d", snapID)
	if err != nil {
		t.Fatalf("snapshot delete command failed: %v\nOutput: %s", err, out)
	}

	var count int
	if err := db.Get(&count, "SELECT count(*) FROM snapshots WHERE repo_id = ? AND id = ?", repo.ID, snapID); err != nil {
		t.Fatalf("failed to verify snapshot deletion: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected snapshot to be deleted")
	}
}

func TestSnapshotCmd_ForceRequiresDelete(t *testing.T) {
	tempDir, db, _ := setupTestRepo(t)
	defer db.Close()
	origWd, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(origWd)

	if _, err := executeCommand(rootCmd, "snapshot", "--force", "dummy"); err == nil {
		t.Fatalf("expected --force without -d to fail")
	}
}
