package services

import (
	"path/filepath"
	"testing"
	"time"

	"pgv/internal/metadata"

	"github.com/google/uuid"
)

func TestRuntimeService_nextAvailablePort(t *testing.T) {
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

	var repo metadata.Repo
	if err := db.Get(&repo, "SELECT * FROM repos LIMIT 1"); err != nil {
		t.Fatalf("Failed to get repo: %v", err)
	}

	var mainBranch metadata.Branch
	if err := db.Get(&mainBranch, "SELECT * FROM branches WHERE id = ?", repo.ActiveBranchID); err != nil {
		t.Fatalf("Failed to get main branch: %v", err)
	}

	now := time.Now().UTC()

	secondBranch := metadata.Branch{
		ID:             uuid.New().String(),
		RepoID:         repo.ID,
		Name:           "feature-2",
		BaseSnapshotID: "",
		HeadSnapshotID: "",
		DataPath:       filepath.Join(tempDir, ".pgv", "storage", "branches", "feature-2", "PGDATA"),
		Status:         "running",
		Port:           5541,
		IsHead:         false,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if _, err := db.NamedExec(`INSERT INTO branches (id, repo_id, name, base_snapshot_id, head_snapshot_id, data_path, status, port, is_head, created_at, updated_at)
		VALUES (:id, :repo_id, :name, :base_snapshot_id, :head_snapshot_id, :data_path, :status, :port, :is_head, :created_at, :updated_at)`, secondBranch); err != nil {
		t.Fatalf("Failed to insert second branch: %v", err)
	}

	firstInstance := metadata.Instance{
		ID:            uuid.New().String(),
		RepoID:        repo.ID,
		BranchID:      mainBranch.ID,
		RuntimeType:   "docker",
		ContainerName: "pgv-test-repo-main",
		Port:          5540,
		Status:        "running",
		PID:           0,
		StartedAt:     now,
	}
	if _, err := db.NamedExec(`INSERT INTO instances (id, repo_id, branch_id, runtime_type, container_name, port, status, pid, started_at)
		VALUES (:id, :repo_id, :branch_id, :runtime_type, :container_name, :port, :status, :pid, :started_at)`, firstInstance); err != nil {
		t.Fatalf("Failed to insert first running instance: %v", err)
	}

	secondInstance := metadata.Instance{
		ID:            uuid.New().String(),
		RepoID:        repo.ID,
		BranchID:      secondBranch.ID,
		RuntimeType:   "docker",
		ContainerName: "pgv-test-repo-feature-2",
		Port:          5541,
		Status:        "running",
		PID:           0,
		StartedAt:     now,
	}
	if _, err := db.NamedExec(`INSERT INTO instances (id, repo_id, branch_id, runtime_type, container_name, port, status, pid, started_at)
		VALUES (:id, :repo_id, :branch_id, :runtime_type, :container_name, :port, :status, :pid, :started_at)`, secondInstance); err != nil {
		t.Fatalf("Failed to insert second running instance: %v", err)
	}

	runtimeSvc := &RuntimeService{db: db}

	port, err := runtimeSvc.nextAvailablePort(repo.ID, 5540)
	if err != nil {
		t.Fatalf("nextAvailablePort failed: %v", err)
	}
	if port != 5542 {
		t.Fatalf("Expected next available port 5542, got %d", port)
	}
}

func TestRuntimeService_nextAvailablePort_DefaultBasePortFallback(t *testing.T) {
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

	var repo metadata.Repo
	if err := db.Get(&repo, "SELECT * FROM repos LIMIT 1"); err != nil {
		t.Fatalf("Failed to get repo: %v", err)
	}

	runtimeSvc := &RuntimeService{db: db}
	port, err := runtimeSvc.nextAvailablePort(repo.ID, 0)
	if err != nil {
		t.Fatalf("nextAvailablePort failed: %v", err)
	}
	if port != 5540 {
		t.Fatalf("Expected default fallback port 5540, got %d", port)
	}
}
