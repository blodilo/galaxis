package db

import (
	"errors"
	"fmt"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// Migrate runs all pending up-migrations from the given migrations directory.
// migrationsPath should be a file:// URL, e.g. "file://migrations".
//
// golang-migrate's pgx/v5 driver requires the scheme "pgx5://".
// We accept the standard postgres:// / postgresql:// and rewrite it automatically.
func Migrate(databaseURL, migrationsPath string) error {
	migrateURL := toPgx5URL(databaseURL)

	m, err := migrate.New(migrationsPath, migrateURL)
	if err != nil {
		return fmt.Errorf("db: migrate init: %w", err)
	}
	defer func() { _, _ = m.Close() }()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("db: migrate up: %w", err)
	}
	return nil
}

// toPgx5URL rewrites postgres:// or postgresql:// to pgx5:// as required
// by github.com/golang-migrate/migrate/v4/database/pgx/v5.
func toPgx5URL(u string) string {
	for _, prefix := range []string{"postgresql://", "postgres://"} {
		if strings.HasPrefix(u, prefix) {
			return "pgx5://" + u[len(prefix):]
		}
	}
	return u
}
