package docker

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

func WaitForHealthy(ctx context.Context, port int, user, password, dbname string) error {
	connStr := fmt.Sprintf("postgres://%s:%s@127.0.0.1:%d/%s?sslmode=disable", user, password, port, dbname)

	// Try to connect for up to 30 seconds
	deadline := time.Now().Add(30 * time.Second)

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		conn, err := pgx.Connect(ctx, connStr)
		if err == nil {
			err = conn.Ping(ctx)
			conn.Close(ctx)
			if err == nil {
				return nil
			}
		}

		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("timeout waiting for postgres to become healthy on port %d", port)
}
