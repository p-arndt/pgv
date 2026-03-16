package services

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"pgv/internal/config"
	"pgv/internal/metadata"
	"pgv/internal/runtime/docker"

	derrdefs "github.com/docker/docker/errdefs"
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

func TestRuntimeService_StartBranchWithOptions_ReconcilesStaleRunningState(t *testing.T) {
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
	if _, err := db.Exec(`UPDATE branches SET status = ?, updated_at = ? WHERE id = ?`, "running", now, mainBranch.ID); err != nil {
		t.Fatalf("Failed to mark branch running: %v", err)
	}

	instance := metadata.Instance{
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
		VALUES (:id, :repo_id, :branch_id, :runtime_type, :container_name, :port, :status, :pid, :started_at)`, instance); err != nil {
		t.Fatalf("Failed to insert instance: %v", err)
	}

	fake := &fakeRuntimeManager{
		statusByContainer: map[string]string{"pgv-test-repo-main": "not-found"},
		startErr:          errors.New("boom"),
	}
	runtimeSvc := &RuntimeService{db: db, manager: fake}

	cfg := config.DefaultConfig(repo.Name)
	err = runtimeSvc.StartBranchWithOptions(context.Background(), mainBranch.ID, &cfg, StartBranchOptions{})
	if err == nil {
		t.Fatal("expected start error")
	}

	if fake.startCalls != 1 {
		t.Fatalf("expected exactly one start attempt, got %d", fake.startCalls)
	}

	var updatedBranch metadata.Branch
	if err := db.Get(&updatedBranch, "SELECT * FROM branches WHERE id = ?", mainBranch.ID); err != nil {
		t.Fatalf("Failed to reload branch: %v", err)
	}
	if updatedBranch.Status != "stopped" {
		t.Fatalf("expected stale running branch to be reconciled to stopped, got %s", updatedBranch.Status)
	}

	var runningInstanceCount int
	if err := db.Get(&runningInstanceCount, "SELECT count(*) FROM instances WHERE branch_id = ? AND status = 'running'", mainBranch.ID); err != nil {
		t.Fatalf("Failed to count running instances: %v", err)
	}
	if runningInstanceCount != 0 {
		t.Fatalf("expected no running instances after reconciliation, got %d", runningInstanceCount)
	}
}

func TestRuntimeService_StopBranch_IgnoresMissingContainerAndReconciles(t *testing.T) {
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
	if _, err := db.Exec(`UPDATE branches SET status = ?, updated_at = ? WHERE id = ?`, "running", now, mainBranch.ID); err != nil {
		t.Fatalf("Failed to mark branch running: %v", err)
	}

	instance := metadata.Instance{
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
		VALUES (:id, :repo_id, :branch_id, :runtime_type, :container_name, :port, :status, :pid, :started_at)`, instance); err != nil {
		t.Fatalf("Failed to insert instance: %v", err)
	}

	notFoundErr := derrdefs.NotFound(errors.New("container not found"))
	fake := &fakeRuntimeManager{
		stopErr:   notFoundErr,
		removeErr: notFoundErr,
	}
	runtimeSvc := &RuntimeService{db: db, manager: fake}

	if err := runtimeSvc.StopBranch(context.Background(), mainBranch.ID); err != nil {
		t.Fatalf("StopBranch failed: %v", err)
	}

	if fake.stopCalls != 1 {
		t.Fatalf("expected one stop call, got %d", fake.stopCalls)
	}
	if fake.removeCalls != 1 {
		t.Fatalf("expected one remove call, got %d", fake.removeCalls)
	}

	var updatedBranch metadata.Branch
	if err := db.Get(&updatedBranch, "SELECT * FROM branches WHERE id = ?", mainBranch.ID); err != nil {
		t.Fatalf("Failed to reload branch: %v", err)
	}
	if updatedBranch.Status != "stopped" {
		t.Fatalf("expected branch to be stopped, got %s", updatedBranch.Status)
	}

	var runningInstanceCount int
	if err := db.Get(&runningInstanceCount, "SELECT count(*) FROM instances WHERE branch_id = ? AND status = 'running'", mainBranch.ID); err != nil {
		t.Fatalf("Failed to count running instances: %v", err)
	}
	if runningInstanceCount != 0 {
		t.Fatalf("expected no running instances after stop, got %d", runningInstanceCount)
	}
}

type fakeRuntimeManager struct {
	statusByContainer map[string]string
	startErr          error
	stopErr           error
	removeErr         error
	startCalls        int
	stopCalls         int
	removeCalls       int
}

func (f *fakeRuntimeManager) Start(_ context.Context, _ docker.StartRequest) (string, error) {
	f.startCalls++
	if f.startErr != nil {
		return "", f.startErr
	}
	return "container-id", nil
}

func (f *fakeRuntimeManager) Stop(_ context.Context, _ string) error {
	f.stopCalls++
	return f.stopErr
}

func (f *fakeRuntimeManager) Remove(_ context.Context, _ string) error {
	f.removeCalls++
	return f.removeErr
}

func (f *fakeRuntimeManager) Status(_ context.Context, containerID string) (string, error) {
	if status, ok := f.statusByContainer[containerID]; ok {
		return status, nil
	}
	return "not-found", nil
}
