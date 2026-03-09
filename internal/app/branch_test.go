package app

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"pgv/internal/metadata"
	"pgv/internal/services"
)

func setupTestRepo(t *testing.T) (string, *metadata.DB, *metadata.Repo) {
	tempDir := t.TempDir()

	repoSvc := services.NewRepoService(tempDir)
	if err := repoSvc.Init("test-repo", ""); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	dbPath := filepath.Join(tempDir, ".pgv", "meta", "state.db")
	db, err := metadata.Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open DB: %v", err)
	}

	var repo metadata.Repo
	if err := db.Get(&repo, "SELECT * FROM repos LIMIT 1"); err != nil {
		t.Fatalf("Failed to get repo: %v", err)
	}

	return tempDir, db, &repo
}

func executeCommand(cmd *cobra.Command, args ...string) (string, error) {
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return buf.String(), err
}

func TestBranchCmd_FromActiveBranch(t *testing.T) {
	tempDir, db, repo := setupTestRepo(t)
	defer db.Close()

	origWd, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(origWd)

	// Ensure there is an active branch
	if repo.ActiveBranchID == "" {
		t.Fatalf("Expected repo to have an active branch after init")
	}

	// Create branch
	out, err := executeCommand(rootCmd, "branch", "feature-1")
	if err != nil {
		t.Fatalf("branch command failed: %v\nOutput: %s", err, out)
	}

	// Verify branch created
	var branch metadata.Branch
	if err := db.Get(&branch, "SELECT * FROM branches WHERE repo_id = ? AND name = ?", repo.ID, "feature-1"); err != nil {
		t.Fatalf("Branch 'feature-1' not found in db: %v", err)
	}

	// Should have branched from a new checkpoint of the active branch
	if branch.BaseSnapshotID == "" {
		t.Fatalf("Expected new branch to have a base snapshot ID")
	}
}

func TestBranchCmd_FromOtherBranch(t *testing.T) {
	tempDir, db, repo := setupTestRepo(t)
	defer db.Close()

	origWd, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(origWd)

	// The initial branch is typically 'main'
	var mainBranch metadata.Branch
	if err := db.Get(&mainBranch, "SELECT * FROM branches WHERE repo_id = ? AND name = ?", repo.ID, "main"); err != nil {
		t.Fatalf("Branch 'main' not found: %v", err)
	}

	// Create feature-1 from main
	out, err := executeCommand(rootCmd, "branch", "feature-1", "main")
	if err != nil {
		t.Fatalf("branch command failed: %v\nOutput: %s", err, out)
	}

	// Verify branch created
	var branch metadata.Branch
	if err := db.Get(&branch, "SELECT * FROM branches WHERE repo_id = ? AND name = ?", repo.ID, "feature-1"); err != nil {
		t.Fatalf("Branch 'feature-1' not found in db: %v", err)
	}
}

func TestBranchCmd_FromSnapshotID(t *testing.T) {
	tempDir, db, repo := setupTestRepo(t)
	defer db.Close()

	origWd, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(origWd)

	// Insert dummy snapshot
	snapID := uuid.New().String()
	now := time.Now().UTC()
	snap := metadata.Snapshot{
		ID:               snapID,
		RepoID:           repo.ID,
		ParentSnapshotID: nil,
		SourceBranchID:   nil,
		Label:            "test-snap",
		Kind:             "manual",
		DataPath:         filepath.Join(tempDir, "dummy-snap-path"),
		DriverType:       "copydir",
		RestorePointName: "rp",
		LSN:              "0/0",
		SizeBytes:        0,
		CreatedAt:        now,
	}

	_, err := db.NamedExec(`INSERT INTO snapshots (id, repo_id, label, kind, data_path, driver_type, restore_point_name, lsn, size_bytes, created_at)
		VALUES (:id, :repo_id, :label, :kind, :data_path, :driver_type, :restore_point_name, :lsn, :size_bytes, :created_at)`, snap)
	if err != nil {
		t.Fatalf("Failed to insert dummy snapshot: %v", err)
	}

	out, err := executeCommand(rootCmd, "branch", "feature-1", snapID)
	if err != nil {
		t.Fatalf("branch command failed: %v\nOutput: %s", err, out)
	}

	// Verify branch created
	var branch metadata.Branch
	if err := db.Get(&branch, "SELECT * FROM branches WHERE repo_id = ? AND name = ?", repo.ID, "feature-1"); err != nil {
		t.Fatalf("Branch 'feature-1' not found in db: %v", err)
	}

	if branch.BaseSnapshotID != snapID {
		t.Errorf("Expected branch to be created from %s, got %s", snapID, branch.BaseSnapshotID)
	}
}

func TestBranchCmd_FromSnapshotLabel(t *testing.T) {
	tempDir, db, repo := setupTestRepo(t)
	defer db.Close()

	origWd, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(origWd)

	// Insert dummy snapshot
	snapID := uuid.New().String()
	now := time.Now().UTC()
	snap := metadata.Snapshot{
		ID:               snapID,
		RepoID:           repo.ID,
		ParentSnapshotID: nil,
		SourceBranchID:   nil,
		Label:            "unique-label",
		Kind:             "manual",
		DataPath:         filepath.Join(tempDir, "dummy-snap-path"),
		DriverType:       "copydir",
		RestorePointName: "rp",
		LSN:              "0/0",
		SizeBytes:        0,
		CreatedAt:        now,
	}

	_, err := db.NamedExec(`INSERT INTO snapshots (id, repo_id, label, kind, data_path, driver_type, restore_point_name, lsn, size_bytes, created_at)
		VALUES (:id, :repo_id, :label, :kind, :data_path, :driver_type, :restore_point_name, :lsn, :size_bytes, :created_at)`, snap)
	if err != nil {
		t.Fatalf("Failed to insert dummy snapshot: %v", err)
	}

	out, err := executeCommand(rootCmd, "branch", "feature-1", "unique-label")
	if err != nil {
		t.Fatalf("branch command failed: %v\nOutput: %s", err, out)
	}

	// Verify branch created
	var branch metadata.Branch
	if err := db.Get(&branch, "SELECT * FROM branches WHERE repo_id = ? AND name = ?", repo.ID, "feature-1"); err != nil {
		t.Fatalf("Branch 'feature-1' not found in db: %v", err)
	}

	if branch.BaseSnapshotID != snapID {
		t.Errorf("Expected branch to be created from %s, got %s", snapID, branch.BaseSnapshotID)
	}
}
