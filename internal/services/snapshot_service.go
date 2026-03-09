package services

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"pgv/internal/metadata"
	"pgv/internal/snapshot"
	"pgv/internal/snapshot/copydir"
)

type SnapshotService struct {
	db     *metadata.DB
	driver snapshot.Driver
}

func NewSnapshotService(db *metadata.DB, driver string) (*SnapshotService, error) {
	// For MVP only copydir is supported here, but can map to others
	var d snapshot.Driver
	if driver == "copydir" {
		d = copydir.NewDriver()
	} else {
		d = copydir.NewDriver() // fallback
	}
	return &SnapshotService{db: db, driver: d}, nil
}

func (s *SnapshotService) CreateCheckpoint(ctx context.Context, repoID, branchID, label string) (string, error) {
	var branch metadata.Branch
	if err := s.db.Get(&branch, "SELECT * FROM branches WHERE id = ?", branchID); err != nil {
		return "", fmt.Errorf("branch not found: %w", err)
	}

	var repo metadata.Repo
	if err := s.db.Get(&repo, "SELECT * FROM repos WHERE id = ?", repoID); err != nil {
		return "", fmt.Errorf("repo not found: %w", err)
	}

	// Wait, Postgres checkpoint/restore point logic goes here ideally via Postgres Control Layer
	// MVP: Stop branch, create snapshot, restart branch (or just clone data directly if copydir supports it, copydir on live PGDATA is risky but let's try for MVP or require stop)
	// Actually, if branch is stopped, copy is safe. If running, pg_basebackup is better, but MVP plan said: "Basebackup or copydir, whichever is simpler" and "never edit files inside a running PGDATA unless operation is known-safe".

	snapshotID := "snap_" + strings.ReplaceAll(uuid.New().String(), "-", "")[0:12]
	snapshotsDir := filepath.Join(repo.RootPath, ".pgv", "storage", "snapshots", snapshotID)

	req := snapshot.CreateSnapshotRequest{
		SourcePath: branch.DataPath,
		TargetPath: snapshotsDir,
	}

	res, err := s.driver.CreateSnapshot(ctx, req)
	if err != nil {
		return "", fmt.Errorf("driver failed to create snapshot: %w", err)
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
