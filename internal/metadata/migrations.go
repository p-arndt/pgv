package metadata

import (
	"database/sql"
)

var schema = `
CREATE TABLE IF NOT EXISTS repos (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	root_path TEXT NOT NULL,
	postgres_image TEXT NOT NULL,
	postgres_version TEXT NOT NULL,
	snapshot_driver TEXT NOT NULL,
	active_branch_id TEXT,
	created_at DATETIME NOT NULL,
	updated_at DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS branches (
	id TEXT PRIMARY KEY,
	repo_id TEXT NOT NULL REFERENCES repos(id),
	name TEXT NOT NULL,
	base_snapshot_id TEXT NOT NULL,
	head_snapshot_id TEXT NOT NULL,
	data_path TEXT NOT NULL,
	status TEXT NOT NULL,
	port INTEGER NOT NULL,
	is_head BOOLEAN NOT NULL DEFAULT 0,
	ttl_expires_at DATETIME,
	created_at DATETIME NOT NULL,
	updated_at DATETIME NOT NULL,
	UNIQUE(repo_id, name)
);

CREATE TABLE IF NOT EXISTS snapshots (
	id TEXT PRIMARY KEY,
	repo_id TEXT NOT NULL REFERENCES repos(id),
	parent_snapshot_id TEXT,
	source_branch_id TEXT,
	label TEXT NOT NULL,
	kind TEXT NOT NULL,
	data_path TEXT NOT NULL,
	driver_type TEXT NOT NULL,
	restore_point_name TEXT NOT NULL,
	lsn TEXT NOT NULL,
	size_bytes INTEGER NOT NULL,
	created_at DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS tags (
	id TEXT PRIMARY KEY,
	repo_id TEXT NOT NULL REFERENCES repos(id),
	snapshot_id TEXT NOT NULL REFERENCES snapshots(id),
	name TEXT NOT NULL,
	created_at DATETIME NOT NULL,
	UNIQUE(repo_id, name)
);

CREATE TABLE IF NOT EXISTS instances (
	id TEXT PRIMARY KEY,
	repo_id TEXT NOT NULL REFERENCES repos(id),
	branch_id TEXT NOT NULL REFERENCES branches(id),
	runtime_type TEXT NOT NULL,
	container_name TEXT NOT NULL,
	port INTEGER NOT NULL,
	status TEXT NOT NULL,
	pid INTEGER NOT NULL,
	started_at DATETIME NOT NULL,
	stopped_at DATETIME
);

CREATE TABLE IF NOT EXISTS operations (
	id TEXT PRIMARY KEY,
	repo_id TEXT NOT NULL REFERENCES repos(id),
	type TEXT NOT NULL,
	status TEXT NOT NULL,
	payload_json TEXT NOT NULL,
	error_text TEXT NOT NULL,
	started_at DATETIME NOT NULL,
	finished_at DATETIME
);
`

func RunMigrations(db *sql.DB) error {
	_, err := db.Exec(schema)
	return err
}
