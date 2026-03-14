// Package db provides the PostgreSQL connection pool and migration runner.
package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Connect creates a pgxpool connection pool and verifies connectivity.
// Retries up to maxAttempts times (useful during Docker Compose startup).
func Connect(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("db: parse config: %w", err)
	}

	cfg.MaxConns = 20
	cfg.MinConns = 2
	cfg.HealthCheckPeriod = 30 * time.Second
	cfg.MaxConnIdleTime = 5 * time.Minute

	const maxAttempts = 10
	var pool *pgxpool.Pool
	for i := range maxAttempts {
		pool, err = pgxpool.NewWithConfig(ctx, cfg)
		if err == nil {
			if pingErr := pool.Ping(ctx); pingErr == nil {
				return pool, nil
			} else {
				pool.Close()
				err = pingErr
			}
		}
		if i < maxAttempts-1 {
			fmt.Printf("db: connection attempt %d/%d failed: %v — retrying in 2s\n", i+1, maxAttempts, err)
			time.Sleep(2 * time.Second)
		}
	}
	return nil, fmt.Errorf("db: could not connect after %d attempts: %w", maxAttempts, err)
}
