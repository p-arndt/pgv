package metadata

import (
	"time"
)

type Repo struct {
	ID              string    `db:"id"`
	Name            string    `db:"name"`
	RootPath        string    `db:"root_path"`
	PostgresImage   string    `db:"postgres_image"`
	PostgresVersion string    `db:"postgres_version"`
	SnapshotDriver  string    `db:"snapshot_driver"`
	ActiveBranchID  string    `db:"active_branch_id"`
	CreatedAt       time.Time `db:"created_at"`
	UpdatedAt       time.Time `db:"updated_at"`
}

type Branch struct {
	ID             string     `db:"id"`
	RepoID         string     `db:"repo_id"`
	Name           string     `db:"name"`
	BaseSnapshotID string     `db:"base_snapshot_id"`
	HeadSnapshotID string     `db:"head_snapshot_id"`
	DataPath       string     `db:"data_path"`
	Status         string     `db:"status"`
	Port           int        `db:"port"`
	IsHead         bool       `db:"is_head"`
	TTLExpiresAt   *time.Time `db:"ttl_expires_at"`
	CreatedAt      time.Time  `db:"created_at"`
	UpdatedAt      time.Time  `db:"updated_at"`
}

type Snapshot struct {
	ID               string    `db:"id"`
	RepoID           string    `db:"repo_id"`
	ParentSnapshotID *string   `db:"parent_snapshot_id"`
	SourceBranchID   *string   `db:"source_branch_id"`
	Label            string    `db:"label"`
	Kind             string    `db:"kind"`
	DataPath         string    `db:"data_path"`
	DriverType       string    `db:"driver_type"`
	RestorePointName string    `db:"restore_point_name"`
	LSN              string    `db:"lsn"`
	SizeBytes        int64     `db:"size_bytes"`
	CreatedAt        time.Time `db:"created_at"`
}

type Tag struct {
	ID         string    `db:"id"`
	RepoID     string    `db:"repo_id"`
	SnapshotID string    `db:"snapshot_id"`
	Name       string    `db:"name"`
	CreatedAt  time.Time `db:"created_at"`
}

type Instance struct {
	ID            string     `db:"id"`
	RepoID        string     `db:"repo_id"`
	BranchID      string     `db:"branch_id"`
	RuntimeType   string     `db:"runtime_type"`
	ContainerName string     `db:"container_name"`
	Port          int        `db:"port"`
	Status        string     `db:"status"`
	PID           int        `db:"pid"`
	StartedAt     time.Time  `db:"started_at"`
	StoppedAt     *time.Time `db:"stopped_at"`
}

type Operation struct {
	ID          string     `db:"id"`
	RepoID      string     `db:"repo_id"`
	Type        string     `db:"type"`
	Status      string     `db:"status"`
	PayloadJSON string     `db:"payload_json"`
	ErrorText   string     `db:"error_text"`
	StartedAt   time.Time  `db:"started_at"`
	FinishedAt  *time.Time `db:"finished_at"`
}
