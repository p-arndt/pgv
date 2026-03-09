package services

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"pgv/internal/config"
	"pgv/internal/metadata"
	"pgv/internal/util"
)

func TestBranchService_CreateCheckoutDelete(t *testing.T) {
	tempDir := t.TempDir()

	// 1. Initialize repo
	repoSvc := NewRepoService(tempDir)
	if err := repoSvc.Init("test-repo", ""); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// 2. Open DB and create BranchService
	dbPath := filepath.Join(tempDir, ".pgv", "meta", "state.db")
	db, err := metadata.Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open DB: %v", err)
	}
	defer db.Close()

	branchSvc, err := NewBranchService(db, "copydir")
	if err != nil {
		t.Fatalf("Failed to create BranchService: %v", err)
	}

	// Get repo
	var repo metadata.Repo
	if err := db.Get(&repo, "SELECT * FROM repos LIMIT 1"); err != nil {
		t.Fatalf("Failed to get repo: %v", err)
	}

	// Create a dummy snapshot record to branch from
	snapID := uuid.New().String()
	now := time.Now().UTC()
	snap := metadata.Snapshot{
		ID:               snapID,
		RepoID:           repo.ID,
		ParentSnapshotID: nil,
		SourceBranchID:   nil,
		Label:            "test-snap",
		Kind:             "manual",
		DataPath:         filepath.Join(tempDir, "dummy-snap-path"), // Need dummy dir
		DriverType:       "copydir",
		RestorePointName: "rp",
		LSN:              "0/0",
		SizeBytes:        0,
		CreatedAt:        now,
	}

	_, err = db.NamedExec(`INSERT INTO snapshots (id, repo_id, label, kind, data_path, driver_type, restore_point_name, lsn, size_bytes, created_at)
		VALUES (:id, :repo_id, :label, :kind, :data_path, :driver_type, :restore_point_name, :lsn, :size_bytes, :created_at)`, snap)
	if err != nil {
		t.Fatalf("Failed to insert dummy snapshot: %v", err)
	}

	ctx := context.Background()

	// 3. Test CreateBranch
	newBranchName := "feature-x"
	branchID, err := branchSvc.CreateBranch(ctx, repo.ID, snapID, newBranchName)
	if err != nil {
		t.Fatalf("CreateBranch failed: %v", err)
	}
	if branchID == "" {
		t.Fatalf("Expected non-empty branch ID")
	}

	var branch metadata.Branch
	if err := db.Get(&branch, "SELECT * FROM branches WHERE id = ?", branchID); err != nil {
		t.Fatalf("Failed to get created branch: %v", err)
	}
	if branch.Name != newBranchName {
		t.Errorf("Expected branch name %s, got %s", newBranchName, branch.Name)
	}
	if branch.IsHead {
		t.Errorf("Expected newly created branch to NOT be head")
	}

	// 4. Test Checkout
	if err := branchSvc.Checkout(ctx, repo.ID, newBranchName); err != nil {
		t.Fatalf("Checkout failed: %v", err)
	}

	// Verify head changed
	var heads []metadata.Branch
	if err := db.Select(&heads, "SELECT * FROM branches WHERE is_head = 1 AND repo_id = ?", repo.ID); err != nil {
		t.Fatalf("Failed to get head branches: %v", err)
	}
	if len(heads) != 1 {
		t.Fatalf("Expected exactly 1 head branch, got %d", len(heads))
	}
	if heads[0].Name != newBranchName {
		t.Errorf("Expected new head to be %s, got %s", newBranchName, heads[0].Name)
	}

	// Verify repo active branch changed
	var updatedRepo metadata.Repo
	if err := db.Get(&updatedRepo, "SELECT * FROM repos WHERE id = ?", repo.ID); err != nil {
		t.Fatalf("Failed to get updated repo: %v", err)
	}
	if updatedRepo.ActiveBranchID != branch.ID {
		t.Errorf("Expected repo active branch to be %s, got %s", branch.ID, updatedRepo.ActiveBranchID)
	}

	// 5. Test DeleteBranch
	// Should fail without force because it's active
	err = branchSvc.DeleteBranch(ctx, repo.ID, newBranchName, false)
	if err == nil {
		t.Fatalf("Expected error when deleting active branch without force")
	}

	// Force delete
	err = branchSvc.DeleteBranch(ctx, repo.ID, newBranchName, true)
	if err != nil {
		t.Fatalf("DeleteBranch (force) failed: %v", err)
	}

	// Verify deletion
	var deletedBranch metadata.Branch
	err = db.Get(&deletedBranch, "SELECT * FROM branches WHERE id = ?", branch.ID)
	if err == nil {
		t.Fatalf("Expected branch to be deleted")
	}
}

func TestBranchService_RestoreBranch(t *testing.T) {
	tempDir := t.TempDir()

	// 1. Initialize repo
	repoSvc := NewRepoService(tempDir)
	if err := repoSvc.Init("test-repo", ""); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// 2. Open DB and create BranchService
	dbPath := filepath.Join(tempDir, ".pgv", "meta", "state.db")
	db, err := metadata.Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open DB: %v", err)
	}
	defer db.Close()

	branchSvc, err := NewBranchService(db, "copydir")
	if err != nil {
		t.Fatalf("Failed to create BranchService: %v", err)
	}

	// Get repo and branch
	var repo metadata.Repo
	if err := db.Get(&repo, "SELECT * FROM repos LIMIT 1"); err != nil {
		t.Fatalf("Failed to get repo: %v", err)
	}
	var branch metadata.Branch
	if err := db.Get(&branch, "SELECT * FROM branches WHERE id = ?", repo.ActiveBranchID); err != nil {
		t.Fatalf("Failed to get active branch: %v", err)
	}

	// Create a dummy snapshot record to restore from
	snapID := uuid.New().String()
	now := time.Now().UTC()

	// Create actual snapshot data dir
	snapDir := filepath.Join(tempDir, "snap-data")
	if err := util.EnsureDir(snapDir); err != nil {
		t.Fatalf("failed to create snap dir: %v", err)
	}

	snap := metadata.Snapshot{
		ID:               snapID,
		RepoID:           repo.ID,
		ParentSnapshotID: nil,
		SourceBranchID:   &branch.ID,
		Label:            "test-restore-snap",
		Kind:             "manual",
		DataPath:         snapDir,
		DriverType:       "copydir",
		RestorePointName: "rp",
		LSN:              "0/0",
		SizeBytes:        0,
		CreatedAt:        now,
	}

	_, err = db.NamedExec(`INSERT INTO snapshots (id, repo_id, parent_snapshot_id, source_branch_id, label, kind, data_path, driver_type, restore_point_name, lsn, size_bytes, created_at)
		VALUES (:id, :repo_id, :parent_snapshot_id, :source_branch_id, :label, :kind, :data_path, :driver_type, :restore_point_name, :lsn, :size_bytes, :created_at)`, snap)
	if err != nil {
		t.Fatalf("Failed to insert dummy snapshot: %v", err)
	}

	ctx := context.Background()
	cfg := config.DefaultConfig("test-repo")

	// 3. Test RestoreBranch
	err = branchSvc.RestoreBranch(ctx, &cfg, branch.ID, snapID)
	if err != nil {
		t.Fatalf("RestoreBranch failed: %v", err)
	}

	// Verify branch's head snapshot changed
	var updatedBranch metadata.Branch
	if err := db.Get(&updatedBranch, "SELECT * FROM branches WHERE id = ?", branch.ID); err != nil {
		t.Fatalf("Failed to get updated branch: %v", err)
	}
	if updatedBranch.HeadSnapshotID != snapID {
		t.Errorf("Expected branch head snapshot to be %s, got %s", snapID, updatedBranch.HeadSnapshotID)
	}
}
