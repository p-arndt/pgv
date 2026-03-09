package metadata

import (
	"fmt"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	*sqlx.DB
}

func Open(dsn string) (*DB, error) {
	db, err := sqlx.Connect("sqlite3", dsn+"?_fk=1&_journal=wal")
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := RunMigrations(db.DB); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return &DB{DB: db}, nil
}
