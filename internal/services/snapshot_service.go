package services

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"pgv/internal/config"
	"pgv/internal/metadata"
	"pgv/internal/snapshot"
	"pgv/internal/snapshot/copydir"
	"pgv/internal/snapshot/cowfs"
)

type SnapshotService struct {
	db     *metadata.DB
	driver snapshot.Driver
}

func NewSnapshotService(db *metadata.DB, driver string) (*SnapshotService, error) {
	var d snapshot.Driver
	if driver == "cowfs" {
		d = cowfs.NewDriver()
	} else if driver == "copydir" {
		d = copydir.NewDriver()
	} else {
		d = copydir.NewDriver() // fallback
	}
	return &SnapshotService{db: db, driver: d}, nil
}

func (s *SnapshotService) CreateCheckpoint(ctx context.Context, cfg *config.Config, repoID, branchID, label string) (string, error) {
	var branch metadata.Branch
	if err := s.db.Get(&branch, "SELECT * FROM branches WHERE id = ?", branchID); err != nil {
		return "", fmt.Errorf("branch not found: %w", err)
	}

	var repo metadata.Repo
	if err := s.db.Get(&repo, "SELECT * FROM repos WHERE id = ?", repoID); err != nil {
		return "", fmt.Errorf("repo not found: %w", err)
	}

	wasRunning := branch.Status == "running"
	runtimeSvc, err := NewRuntimeService(s.db)
	if err != nil {
		return "", fmt.Errorf("could not initialize runtime service: %w", err)
	}

	if wasRunning {
		fmt.Printf("Stopping branch '%s' to create safe physical snapshot...\n", branch.Name)
		if err := runtimeSvc.StopBranch(ctx, branch.ID); err != nil {
			return "", fmt.Errorf("failed to stop branch for snapshot: %w", err)
		}
	}

	snapshotID := "snap_" + strings.ReplaceAll(uuid.New().String(), "-", "")[0:12]
	snapshotsDir := filepath.Join(repo.RootPath, ".pgv", "storage", "snapshots", snapshotID)

	req := snapshot.CreateSnapshotRequest{
		SourcePath: branch.DataPath,
		TargetPath: snapshotsDir,
	}

	res, err := s.driver.CreateSnapshot(ctx, req)
	if err != nil {
		// Try to restart branch if it was running before we return error
		if wasRunning {
			_ = runtimeSvc.StartBranch(ctx, branch.ID, cfg)
		}
		return "", fmt.Errorf("driver failed to create snapshot: %w", err)
	}

	if wasRunning {
		fmt.Printf("Restarting branch '%s'...\n", branch.Name)
		if err := runtimeSvc.StartBranch(ctx, branch.ID, cfg); err != nil {
			fmt.Printf("Warning: failed to restart branch after snapshot: %v\n", err)
		}
	}

	var parentID *string
	if branch.HeadSnapshotID != "" {
		parentID = &branch.HeadSnapshotID
	}

	now := time.Now().UTC()
	snap := metadata.Snapshot{
		ID:               snapshotID,
		RepoID:           repo.ID,
		ParentSnapshotID: parentID,
		SourceBranchID:   &branch.ID,
		Label:            label,
		Kind:             "checkpoint",
		DataPath:         snapshotsDir,
		DriverType:       s.driver.Name(),
		RestorePointName: "rp_" + snapshotID, // Mock
		LSN:              "0/0",              // Mock
		SizeBytes:        res.SizeBytes,
		CreatedAt:        now,
	}

	tx := s.db.MustBegin()
	_, err = tx.NamedExec(`INSERT INTO snapshots (id, repo_id, parent_snapshot_id, source_branch_id, label, kind, data_path, driver_type, restore_point_name, lsn, size_bytes, created_at)
		VALUES (:id, :repo_id, :parent_snapshot_id, :source_branch_id, :label, :kind, :data_path, :driver_type, :restore_point_name, :lsn, :size_bytes, :created_at)`, snap)
	if err != nil {
		tx.Rollback()
		return "", err
	}

	// Update branch's head snapshot
	_, err = tx.Exec(`UPDATE branches SET head_snapshot_id = ?, updated_at = ? WHERE id = ?`, snapshotID, now, branch.ID)
	if err != nil {
		tx.Rollback()
		return "", err
	}

	if err := tx.Commit(); err != nil {
		return "", err
	}

	return snapshotID, nil
}
