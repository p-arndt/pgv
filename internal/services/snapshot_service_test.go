package services

import (
	"context"
	"path/filepath"
	"testing"

	"pgv/internal/config"
	"pgv/internal/metadata"
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
