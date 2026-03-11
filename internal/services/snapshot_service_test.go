package services

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"pgv/internal/config"
	"pgv/internal/metadata"

	"github.com/google/uuid"
)

func TestSnapshotService_CreateCheckpoint(t *testing.T) {
	tempDir := t.TempDir()

	// 1. Initialize repo
	repoSvc := NewRepoService(tempDir)
	if err := repoSvc.Init("test-repo", ""); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// 2. Open DB
	dbPath := filepath.Join(tempDir, ".pgv", "meta", "state.db")
	db, err := metadata.Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open DB: %v", err)
	}
	defer db.Close()

	// 3. Create SnapshotService
	snapSvc, err := NewSnapshotService(db, "copydir")
	if err != nil {
		t.Fatalf("Failed to create SnapshotService: %v", err)
	}

	// Get repo and active branch
	var repo metadata.Repo
	if err := db.Get(&repo, "SELECT * FROM repos LIMIT 1"); err != nil {
		t.Fatalf("Failed to get repo: %v", err)
	}

	var branch metadata.Branch
	if err := db.Get(&branch, "SELECT * FROM branches WHERE id = ?", repo.ActiveBranchID); err != nil {
		t.Fatalf("Failed to get active branch: %v", err)
	}

	ctx := context.Background()
	cfg := config.DefaultConfig("test-repo")

	// 4. Test CreateCheckpoint
	label := "my-first-checkpoint"
	snapshotID, err := snapSvc.CreateCheckpoint(ctx, &cfg, repo.ID, branch.ID, label)
	if err != nil {
		t.Fatalf("CreateCheckpoint failed: %v", err)
	}

	if snapshotID == "" {
		t.Fatalf("Expected non-empty snapshot ID")
	}

	// Verify snapshot record
	var snap metadata.Snapshot
	if err := db.Get(&snap, "SELECT * FROM snapshots WHERE id = ?", snapshotID); err != nil {
		t.Fatalf("Failed to get created snapshot: %v", err)
	}

	if snap.Label != label {
		t.Errorf("Expected snapshot label %s, got %s", label, snap.Label)
	}
	if snap.RepoID != repo.ID {
		t.Errorf("Expected repo ID %s, got %s", repo.ID, snap.RepoID)
	}
	if *snap.SourceBranchID != branch.ID {
		t.Errorf("Expected source branch ID %s, got %s", branch.ID, *snap.SourceBranchID)
	}

	// Verify branch's head snapshot is updated
	var updatedBranch metadata.Branch
	if err := db.Get(&updatedBranch, "SELECT * FROM branches WHERE id = ?", branch.ID); err != nil {
		t.Fatalf("Failed to get updated branch: %v", err)
	}

	if updatedBranch.HeadSnapshotID != snapshotID {
		t.Errorf("Expected branch head snapshot to be %s, got %s", snapshotID, updatedBranch.HeadSnapshotID)
	}
}

func TestSnapshotService_DeleteSnapshot(t *testing.T) {
	tempDir := t.TempDir()

	repoSvc := NewRepoService(tempDir)
	if err := repoSvc.Init("test-repo", ""); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	dbPath := filepath.Join(tempDir, ".pgv", "meta", "state.db")
	db, err := metadata.Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open DB: %v", err)
	}
	defer db.Close()

	snapSvc, err := NewSnapshotService(db, "copydir")
	if err != nil {
		t.Fatalf("Failed to create SnapshotService: %v", err)
	}

	var repo metadata.Repo
	if err := db.Get(&repo, "SELECT * FROM repos LIMIT 1"); err != nil {
		t.Fatalf("Failed to get repo: %v", err)
	}

	now := time.Now().UTC()
	parentID := "snap_parent"
	childID := "snap_child"

	parent := metadata.Snapshot{
		ID:               parentID,
		RepoID:           repo.ID,
		ParentSnapshotID: nil,
		SourceBranchID:   nil,
		Label:            "parent",
		Kind:             "checkpoint",
		DataPath:         filepath.Join(tempDir, ".pgv", "storage", "snapshots", parentID),
		DriverType:       "copydir",
		RestorePointName: "rp_parent",
		LSN:              "0/0",
		SizeBytes:        0,
		CreatedAt:        now,
	}

	if _, err := db.NamedExec(`INSERT INTO snapshots (id, repo_id, parent_snapshot_id, source_branch_id, label, kind, data_path, driver_type, restore_point_name, lsn, size_bytes, created_at)
		VALUES (:id, :repo_id, :parent_snapshot_id, :source_branch_id, :label, :kind, :data_path, :driver_type, :restore_point_name, :lsn, :size_bytes, :created_at)`, parent); err != nil {
		t.Fatalf("Failed to insert parent snapshot: %v", err)
	}

	childParent := parentID
	child := metadata.Snapshot{
		ID:               childID,
		RepoID:           repo.ID,
		ParentSnapshotID: &childParent,
		SourceBranchID:   nil,
		Label:            "child",
		Kind:             "checkpoint",
		DataPath:         filepath.Join(tempDir, ".pgv", "storage", "snapshots", childID),
		DriverType:       "copydir",
		RestorePointName: "rp_child",
		LSN:              "0/0",
		SizeBytes:        0,
		CreatedAt:        now,
	}

	if _, err := db.NamedExec(`INSERT INTO snapshots (id, repo_id, parent_snapshot_id, source_branch_id, label, kind, data_path, driver_type, restore_point_name, lsn, size_bytes, created_at)
		VALUES (:id, :repo_id, :parent_snapshot_id, :source_branch_id, :label, :kind, :data_path, :driver_type, :restore_point_name, :lsn, :size_bytes, :created_at)`, child); err != nil {
		t.Fatalf("Failed to insert child snapshot: %v", err)
	}

	if err := snapSvc.DeleteSnapshot(context.Background(), repo.ID, parentID, false); err == nil {
		t.Fatalf("Expected delete without force to fail for parent snapshot")
	}

	if err := snapSvc.DeleteSnapshot(context.Background(), repo.ID, parentID, true); err != nil {
		t.Fatalf("Expected force delete to succeed: %v", err)
	}

	var count int
	if err := db.Get(&count, "SELECT count(*) FROM snapshots WHERE repo_id = ? AND id = ?", repo.ID, parentID); err != nil {
		t.Fatalf("Failed to verify parent deletion: %v", err)
	}
	if count != 0 {
		t.Fatalf("Expected parent snapshot to be deleted")
	}

	var childAfter metadata.Snapshot
	if err := db.Get(&childAfter, "SELECT * FROM snapshots WHERE repo_id = ? AND id = ?", repo.ID, childID); err != nil {
		t.Fatalf("Failed to load child snapshot after delete: %v", err)
	}
	if childAfter.ParentSnapshotID != nil {
		t.Fatalf("Expected child parent_snapshot_id to be reparented to NULL")
	}
}

func TestSnapshotService_DeleteSnapshot_RejectsBranchReferences(t *testing.T) {
	tempDir := t.TempDir()

	repoSvc := NewRepoService(tempDir)
	if err := repoSvc.Init("test-repo", ""); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	dbPath := filepath.Join(tempDir, ".pgv", "meta", "state.db")
	db, err := metadata.Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open DB: %v", err)
	}
	defer db.Close()

	snapSvc, err := NewSnapshotService(db, "copydir")
	if err != nil {
		t.Fatalf("Failed to create SnapshotService: %v", err)
	}

	var repo metadata.Repo
	if err := db.Get(&repo, "SELECT * FROM repos LIMIT 1"); err != nil {
		t.Fatalf("Failed to get repo: %v", err)
	}

	var branch metadata.Branch
	if err := db.Get(&branch, "SELECT * FROM branches WHERE id = ?", repo.ActiveBranchID); err != nil {
		t.Fatalf("Failed to get active branch: %v", err)
	}

	snapID := "snap_ref_" + uuid.New().String()[0:8]
	now := time.Now().UTC()
	snap := metadata.Snapshot{
		ID:               snapID,
		RepoID:           repo.ID,
		ParentSnapshotID: nil,
		SourceBranchID:   &branch.ID,
		Label:            "ref-snap",
		Kind:             "checkpoint",
		DataPath:         filepath.Join(tempDir, ".pgv", "storage", "snapshots", snapID),
		DriverType:       "copydir",
		RestorePointName: "rp_ref",
		LSN:              "0/0",
		SizeBytes:        0,
		CreatedAt:        now,
	}

	if _, err := db.NamedExec(`INSERT INTO snapshots (id, repo_id, parent_snapshot_id, source_branch_id, label, kind, data_path, driver_type, restore_point_name, lsn, size_bytes, created_at)
		VALUES (:id, :repo_id, :parent_snapshot_id, :source_branch_id, :label, :kind, :data_path, :driver_type, :restore_point_name, :lsn, :size_bytes, :created_at)`, snap); err != nil {
		t.Fatalf("Failed to insert snapshot: %v", err)
	}

	if _, err := db.Exec("UPDATE branches SET head_snapshot_id = ?, base_snapshot_id = ? WHERE id = ?", snapID, snapID, branch.ID); err != nil {
		t.Fatalf("Failed to wire branch snapshot refs: %v", err)
	}

	if err := snapSvc.DeleteSnapshot(context.Background(), repo.ID, snapID, true); err == nil {
		t.Fatalf("Expected delete to fail when snapshot is referenced by branch")
	}
}
