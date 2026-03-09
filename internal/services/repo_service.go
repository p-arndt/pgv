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
	"pgv/internal/snapshot/cowfs"
	"pgv/internal/util"
)

type RepoService struct {
	rootDir string
}

func NewRepoService(rootDir string) *RepoService {
	return &RepoService{rootDir: rootDir}
}

func (s *RepoService) PgvDir() string {
	return filepath.Join(s.rootDir, ".pgv")
}

func (s *RepoService) Init(repoName, fromDir string) error {
	pgvDir := s.PgvDir()
	if util.Exists(pgvDir) {
		return fmt.Errorf("repository already initialized at %s", pgvDir)
	}

	// 1. Create layout
	dirs := []string{
		pgvDir,
		filepath.Join(pgvDir, "meta"),
		filepath.Join(pgvDir, "storage", "branches"),
		filepath.Join(pgvDir, "storage", "snapshots"),
		filepath.Join(pgvDir, "wal", "archive"),
		filepath.Join(pgvDir, "run", "locks"),
		filepath.Join(pgvDir, "logs"),
	}
	for _, dir := range dirs {
		if err := util.EnsureDir(dir); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Smart Router: test CoW support
	var selectedDriver string
	cowDriver := cowfs.NewDriver()
	if err := cowDriver.Validate(context.Background(), snapshot.ValidateDriverRequest{TargetPath: s.rootDir}); err == nil {
		selectedDriver = "cowfs"
		fmt.Println("Copy-on-Write (CoW) filesystem detected. Zero-bloat snapshots enabled.")
	} else {
		selectedDriver = "copydir"
		fmt.Printf("CoW not available (%v). Falling back to full copy (copydir) snapshots.\n", err)
	}

	// 2. Create Default Config
	cfg := config.DefaultConfig(repoName)
	cfg.SnapshotDriver = selectedDriver // Update with the selected driver
	cfgPath := filepath.Join(pgvDir, "config.json")
	if err := config.SaveConfig(cfgPath, &cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	// 3. Create SQLite Metadata DB
	dbPath := filepath.Join(pgvDir, "meta", "state.db")
	db, err := metadata.Open(dbPath)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer db.Close()

	// 4. Create initial records
	repoID := uuid.New().String()
	now := time.Now().UTC()

	repo := metadata.Repo{
		ID:              repoID,
		Name:            repoName,
		RootPath:        s.rootDir,
		PostgresImage:   cfg.PostgresImage,
		PostgresVersion: "17", // Extract from image later
		SnapshotDriver:  cfg.SnapshotDriver,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	_, err = db.NamedExec(`INSERT INTO repos (id, name, root_path, postgres_image, postgres_version, snapshot_driver, created_at, updated_at)
		VALUES (:id, :name, :root_path, :postgres_image, :postgres_version, :snapshot_driver, :created_at, :updated_at)`, repo)
	if err != nil {
		return fmt.Errorf("failed to create repo record: %w", err)
	}

	mainDataPath := filepath.Join(pgvDir, "storage", "branches", cfg.DefaultBranch, "PGDATA")

	// If fromDir is provided, we copy the physical state
	if fromDir != "" {
		fmt.Printf("Importing physical data from %s...\n", fromDir)
		var importDriver snapshot.Driver
		if selectedDriver == "cowfs" {
			importDriver = cowfs.NewDriver()
		} else {
			importDriver = copydir.NewDriver()
		}

		req := snapshot.CloneSnapshotRequest{
			SourcePath: fromDir,
			TargetPath: mainDataPath,
		}
		if _, err := importDriver.CloneSnapshotToBranch(context.Background(), req); err != nil {
			return fmt.Errorf("failed to import data from %s: %w", fromDir, err)
		}
	}

	// Create 'main' branch record
	mainBranch := metadata.Branch{
		ID:             uuid.New().String(),
		RepoID:         repoID,
		Name:           cfg.DefaultBranch,
		BaseSnapshotID: "", // No base yet
		HeadSnapshotID: "", // No head yet
		DataPath:       mainDataPath,
		Status:         "stopped",
		Port:           cfg.BasePort,
		IsHead:         true,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	_, err = db.NamedExec(`INSERT INTO branches (id, repo_id, name, base_snapshot_id, head_snapshot_id, data_path, status, port, is_head, created_at, updated_at)
		VALUES (:id, :repo_id, :name, :base_snapshot_id, :head_snapshot_id, :data_path, :status, :port, :is_head, :created_at, :updated_at)`, mainBranch)
	if err != nil {
		return fmt.Errorf("failed to create main branch record: %w", err)
	}

	// Update repo active branch
	_, err = db.Exec(`UPDATE repos SET active_branch_id = ? WHERE id = ?`, mainBranch.ID, repoID)
	if err != nil {
		return fmt.Errorf("failed to update active branch: %w", err)
	}

	return nil
}
