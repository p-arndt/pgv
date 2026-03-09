package services

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"pgv/internal/config"
	"pgv/internal/metadata"
	"pgv/internal/snapshot"
	"pgv/internal/snapshot/copydir"
)

type BranchService struct {
	db     *metadata.DB
	driver snapshot.Driver
}

func NewBranchService(db *metadata.DB, driver string) (*BranchService, error) {
	var d snapshot.Driver
	if driver == "copydir" {
		d = copydir.NewDriver()
	} else {
		d = copydir.NewDriver()
	}
	return &BranchService{db: db, driver: d}, nil
}

func (s *BranchService) CreateBranch(ctx context.Context, repoID, sourceSnapshotID, newBranchName string) (string, error) {
	var repo metadata.Repo
	if err := s.db.Get(&repo, "SELECT * FROM repos WHERE id = ?", repoID); err != nil {
		return "", fmt.Errorf("repo not found: %w", err)
	}

	var snap metadata.Snapshot
	if err := s.db.Get(&snap, "SELECT * FROM snapshots WHERE id = ?", sourceSnapshotID); err != nil {
		return "", fmt.Errorf("snapshot not found: %w", err)
	}

	// 1. Check if branch already exists
	var count int
	s.db.Get(&count, "SELECT count(*) FROM branches WHERE repo_id = ? AND name = ?", repoID, newBranchName)
	if count > 0 {
		return "", fmt.Errorf("branch %s already exists", newBranchName)
	}

	// 2. Clone snapshot to branch data path
	branchDataPath := filepath.Join(repo.RootPath, ".pgv", "storage", "branches", newBranchName, "PGDATA")
	req := snapshot.CloneSnapshotRequest{
		SourcePath: snap.DataPath,
		TargetPath: branchDataPath,
	}
	if _, err := s.driver.CloneSnapshotToBranch(ctx, req); err != nil {
		return "", fmt.Errorf("driver failed to clone branch: %w", err)
	}

	// 3. Allocate port (naive MVP approach: max port + 1)
	var maxPort int
	s.db.Get(&maxPort, "SELECT MAX(port) FROM branches WHERE repo_id = ?", repoID)
	if maxPort == 0 {
		maxPort = 5540
	} else {
		maxPort++
	}

	now := time.Now().UTC()
	branch := metadata.Branch{
		ID:             uuid.New().String(),
		RepoID:         repoID,
		Name:           newBranchName,
		BaseSnapshotID: snap.ID,
		HeadSnapshotID: snap.ID,
		DataPath:       branchDataPath,
		Status:         "stopped",
		Port:           maxPort,
		IsHead:         false,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	tx := s.db.MustBegin()
	_, err := tx.NamedExec(`INSERT INTO branches (id, repo_id, name, base_snapshot_id, head_snapshot_id, data_path, status, port, is_head, created_at, updated_at)
		VALUES (:id, :repo_id, :name, :base_snapshot_id, :head_snapshot_id, :data_path, :status, :port, :is_head, :created_at, :updated_at)`, branch)
	if err != nil {
		tx.Rollback()
		return "", err
	}

	if err := tx.Commit(); err != nil {
		return "", err
	}

	return branch.ID, nil
}

func (s *BranchService) RestoreBranch(ctx context.Context, cfg *config.Config, branchID, snapshotID string) error {
	var branch metadata.Branch
	if err := s.db.Get(&branch, "SELECT * FROM branches WHERE id = ?", branchID); err != nil {
		return fmt.Errorf("branch not found: %w", err)
	}

	var snap metadata.Snapshot
	if err := s.db.Get(&snap, "SELECT * FROM snapshots WHERE id = ?", snapshotID); err != nil {
		return fmt.Errorf("snapshot not found: %w", err)
	}

	wasRunning := branch.Status == "running"
	runtimeSvc, err := NewRuntimeService(s.db)
	if err != nil {
		return fmt.Errorf("could not initialize runtime service: %w", err)
	}

	if wasRunning {
		fmt.Printf("Stopping branch '%s' to perform safe restore...\n", branch.Name)
		if err := runtimeSvc.StopBranch(ctx, branch.ID); err != nil {
			return fmt.Errorf("failed to stop branch for restore: %w", err)
		}
	}

	// Delete existing branch data
	if err := s.driver.DeleteBranchData(ctx, snapshot.DeleteBranchDataRequest{TargetPath: branch.DataPath}); err != nil {
		return fmt.Errorf("failed to clean existing branch data: %w", err)
	}

	// Clone snapshot to branch data path
	req := snapshot.CloneSnapshotRequest{
		SourcePath: snap.DataPath,
		TargetPath: branch.DataPath,
	}
	if _, err := s.driver.CloneSnapshotToBranch(ctx, req); err != nil {
		// Attempt to start if it was running, although data might be broken now
		if wasRunning {
			fmt.Printf("Warning: failed to restore branch. Restarting in potentially broken state...\n")
			_ = runtimeSvc.StartBranch(ctx, branch.ID, cfg)
		}
		return fmt.Errorf("driver failed to restore branch: %w", err)
	}

	if wasRunning {
		fmt.Printf("Restarting branch '%s'...\n", branch.Name)
		if err := runtimeSvc.StartBranch(ctx, branch.ID, cfg); err != nil {
			fmt.Printf("Warning: failed to restart branch after restore: %v\n", err)
		}
	}

	// Update branch metadata
	now := time.Now().UTC()
	tx := s.db.MustBegin()

	_, err = tx.Exec(`UPDATE branches SET head_snapshot_id = ?, updated_at = ? WHERE id = ?`, snap.ID, now, branch.ID)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (s *BranchService) Checkout(ctx context.Context, repoID, branchName string) error {
	var branch metadata.Branch
	if err := s.db.Get(&branch, "SELECT * FROM branches WHERE repo_id = ? AND name = ?", repoID, branchName); err != nil {
		return fmt.Errorf("branch %s not found: %w", branchName, err)
	}

	tx := s.db.MustBegin()

	// Unset old head
	_, err := tx.Exec(`UPDATE branches SET is_head = 0 WHERE repo_id = ?`, repoID)
	if err != nil {
		tx.Rollback()
		return err
	}

	// Set new head
	_, err = tx.Exec(`UPDATE branches SET is_head = 1 WHERE id = ?`, branch.ID)
	if err != nil {
		tx.Rollback()
		return err
	}

	// Update repo active branch
	_, err = tx.Exec(`UPDATE repos SET active_branch_id = ? WHERE id = ?`, branch.ID, repoID)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (s *BranchService) DeleteBranch(ctx context.Context, repoID, branchName string, force bool) error {
	var branch metadata.Branch
	if err := s.db.Get(&branch, "SELECT * FROM branches WHERE repo_id = ? AND name = ?", repoID, branchName); err != nil {
		return fmt.Errorf("branch %s not found: %w", branchName, err)
	}

	if branch.IsHead && !force {
		return fmt.Errorf("cannot delete active branch without --force flag")
	}

	if branch.Status == "running" && !force {
		return fmt.Errorf("cannot delete running branch, stop it first or use --force flag")
	}

	if branch.Status == "running" {
		runtimeSvc, err := NewRuntimeService(s.db)
		if err == nil {
			_ = runtimeSvc.StopBranch(ctx, branch.ID)
		}
	}

	// Delete existing branch data
	if err := s.driver.DeleteBranchData(ctx, snapshot.DeleteBranchDataRequest{TargetPath: branch.DataPath}); err != nil {
		return fmt.Errorf("failed to clean existing branch data: %w", err)
	}

	tx := s.db.MustBegin()
	// Clean up instances record
	_, _ = tx.Exec(`DELETE FROM instances WHERE branch_id = ?`, branch.ID)

	_, err := tx.Exec(`DELETE FROM branches WHERE id = ?`, branch.ID)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete branch metadata: %w", err)
	}

	if branch.IsHead {
		// If we force deleted HEAD, set active branch to null
		_, _ = tx.Exec(`UPDATE repos SET active_branch_id = NULL WHERE id = ?`, repoID)
	}

	return tx.Commit()
}
