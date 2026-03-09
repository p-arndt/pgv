package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Config struct {
	RepoName          string          `json:"repoName"`
	Runtime           string          `json:"runtime"`
	PostgresImage     string          `json:"postgresImage"`
	SnapshotDriver    string          `json:"snapshotDriver"`
	DefaultBranch     string          `json:"defaultBranch"`
	BasePort          int             `json:"basePort"`
	PgUser            string          `json:"pgUser"`
	PgPassword        string          `json:"pgPassword"`
	PgDatabase        string          `json:"pgDatabase"`
	WalArchiveEnabled bool            `json:"walArchiveEnabled"`
	WalArchivePath    string          `json:"walArchivePath"`
	Retention         RetentionConfig `json:"retention"`
}

type RetentionConfig struct {
	AutoSnapshots int    `json:"autoSnapshots"`
	TempBranchTTL string `json:"tempBranchTTL"`
}

func DefaultConfig(repoName string) Config {
	return Config{
		RepoName:          repoName,
		Runtime:           "docker",
		PostgresImage:     "postgres:17",
		SnapshotDriver:    "copydir", // start with simple driver
		DefaultBranch:     "main",
		BasePort:          5540,
		PgUser:            "postgres",
		PgPassword:        "postgres",
		PgDatabase:        "app",
		WalArchiveEnabled: false,
		WalArchivePath:    ".pgv/wal/archive",
		Retention: RetentionConfig{
			AutoSnapshots: 20,
			TempBranchTTL: "24h",
		},
	}
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func SaveConfig(path string, cfg *Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
