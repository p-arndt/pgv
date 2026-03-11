package services

import (
	"context"
	"fmt"
	"time"

	"pgv/internal/config"
	"pgv/internal/metadata"
	"pgv/internal/runtime/docker"

	"github.com/google/uuid"
)

type RuntimeService struct {
	db      *metadata.DB
	manager *docker.Manager
}

type StartBranchOptions struct {
	Parallel bool
}

func NewRuntimeService(db *metadata.DB) (*RuntimeService, error) {
	manager, err := docker.NewManager()
	if err != nil {
		return nil, err
	}
	return &RuntimeService{db: db, manager: manager}, nil
}

func (s *RuntimeService) StartBranch(ctx context.Context, branchID string, cfg *config.Config) error {
	return s.StartBranchWithOptions(ctx, branchID, cfg, StartBranchOptions{})
}

func (s *RuntimeService) StartBranchWithOptions(ctx context.Context, branchID string, cfg *config.Config, opts StartBranchOptions) error {
	var branch metadata.Branch
	if err := s.db.Get(&branch, "SELECT * FROM branches WHERE id = ?", branchID); err != nil {
		return fmt.Errorf("branch not found: %w", err)
	}

	if branch.Status == "running" {
		return fmt.Errorf("branch %s is already running", branch.Name)
	}

	hostPort := cfg.BasePort
	if hostPort <= 0 {
		hostPort = branch.Port
	}

	if opts.Parallel {
		selectedPort, err := s.nextAvailablePort(branch.RepoID, hostPort)
		if err != nil {
			return err
		}
		hostPort = selectedPort
	} else {
		// Default mode: ensure only one branch is running per repository on the base port.
		var runningBranches []metadata.Branch
		if err := s.db.Select(&runningBranches, "SELECT * FROM branches WHERE repo_id = ? AND status = 'running' AND id != ?", branch.RepoID, branch.ID); err != nil {
			return fmt.Errorf("failed to query running branches: %w", err)
		}
		for _, running := range runningBranches {
			if err := s.StopBranch(ctx, running.ID); err != nil {
				return fmt.Errorf("failed to stop running branch %s: %w", running.Name, err)
			}
		}
	}

	containerName := fmt.Sprintf("pgv-%s-%s", cfg.RepoName, branch.Name)

	req := docker.StartRequest{
		ContainerName: containerName,
		Image:         cfg.PostgresImage,
		PGDataPath:    branch.DataPath,
		HostPort:      hostPort,
		User:          cfg.PgUser,
		Password:      cfg.PgPassword,
		Database:      cfg.PgDatabase,
	}

	containerID, err := s.manager.Start(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}

	err = docker.WaitForHealthy(ctx, hostPort, cfg.PgUser, cfg.PgPassword, cfg.PgDatabase)
	if err != nil {
		// Try to stop if unhealthy
		_ = s.manager.Stop(ctx, containerID)
		_ = s.manager.Remove(ctx, containerID)
		return fmt.Errorf("container started but postgres is not healthy: %w", err)
	}

	now := time.Now().UTC()
	instance := metadata.Instance{
		ID:            uuid.New().String(),
		RepoID:        branch.RepoID,
		BranchID:      branch.ID,
		RuntimeType:   "docker",
		ContainerName: containerName,
		Port:          hostPort,
		Status:        "running",
		PID:           0, // PID not strictly needed for docker
		StartedAt:     now,
	}

	tx := s.db.MustBegin()
	_, err = tx.NamedExec(`INSERT INTO instances (id, repo_id, branch_id, runtime_type, container_name, port, status, pid, started_at) 
		VALUES (:id, :repo_id, :branch_id, :runtime_type, :container_name, :port, :status, :pid, :started_at)`, instance)
	if err != nil {
		tx.Rollback()
		return err
	}

	_, err = tx.Exec(`UPDATE branches SET status = ?, port = ?, updated_at = ? WHERE id = ?`, "running", hostPort, now, branch.ID)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (s *RuntimeService) nextAvailablePort(repoID string, basePort int) (int, error) {
	if basePort <= 0 {
		basePort = 5540
	}

	var runningPorts []int
	if err := s.db.Select(&runningPorts, "SELECT port FROM instances WHERE repo_id = ? AND status = 'running'", repoID); err != nil {
		return 0, fmt.Errorf("failed to list running ports: %w", err)
	}

	used := make(map[int]bool, len(runningPorts))
	for _, p := range runningPorts {
		used[p] = true
	}

	for p := basePort; p <= basePort+1000; p++ {
		if !used[p] {
			return p, nil
		}
	}

	return 0, fmt.Errorf("no free port available in range %d-%d", basePort, basePort+1000)
}

func (s *RuntimeService) StopBranch(ctx context.Context, branchID string) error {
	var branch metadata.Branch
	if err := s.db.Get(&branch, "SELECT * FROM branches WHERE id = ?", branchID); err != nil {
		return fmt.Errorf("branch not found: %w", err)
	}

	if branch.Status != "running" {
		return nil // already stopped
	}

	var instance metadata.Instance
	if err := s.db.Get(&instance, "SELECT * FROM instances WHERE branch_id = ? AND status = 'running' LIMIT 1", branchID); err != nil {
		// inconsistency? force stop
	}

	containerName := instance.ContainerName
	if containerName == "" {
		// Fallback
		var repo metadata.Repo
		_ = s.db.Get(&repo, "SELECT * FROM repos WHERE id = ?", branch.RepoID)
		containerName = fmt.Sprintf("pgv-%s-%s", repo.Name, branch.Name)
	}

	if err := s.manager.Stop(ctx, containerName); err != nil {
		// Log error but proceed to remove
	}

	if err := s.manager.Remove(ctx, containerName); err != nil {
		return fmt.Errorf("failed to remove container: %w", err)
	}

	now := time.Now().UTC()
	tx := s.db.MustBegin()

	_, err := tx.Exec(`UPDATE instances SET status = ?, stopped_at = ? WHERE branch_id = ? AND status = 'running'`, "stopped", now, branch.ID)
	if err != nil {
		tx.Rollback()
		return err
	}

	_, err = tx.Exec(`UPDATE branches SET status = ?, updated_at = ? WHERE id = ?`, "stopped", now, branch.ID)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}
