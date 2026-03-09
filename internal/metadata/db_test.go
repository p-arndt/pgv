package metadata

import (
	"path/filepath"
	"testing"
)

func TestOpenAndMigrate(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	// 1. Open should create the DB and run migrations
	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open db: %v", err)
	}
	defer db.Close()

	// 2. Verify tables exist by querying sqlite_master
	tables := []string{
		"repos",
		"branches",
		"snapshots",
		"tags",
		"instances",
		"operations",
	}

	for _, table := range tables {
		var count int
		err := db.Get(&count, "SELECT count(*) FROM sqlite_master WHERE type='table' AND name=?", table)
		if err != nil {
			t.Errorf("Failed to query for table %s: %v", table, err)
		}
		if count != 1 {
			t.Errorf("Table %s was not created", table)
		}
	}
}
