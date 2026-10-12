package store

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"sort"
	"strconv"
	"strings"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

type migration struct {
	version int
	name    string
	sql     string
}

const createMigrationsTable = `
CREATE TABLE IF NOT EXISTS schema_migrations (
  version INTEGER PRIMARY KEY,
  name TEXT NOT NULL,
  applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
`

// applyMigrations runs every migration that has not been applied yet, in
// version order, each inside its own transaction. A failed migration rolls
// back and aborts startup.
func applyMigrations(conn *sql.DB) error {
	if _, err := conn.Exec(createMigrationsTable); err != nil {
		return fmt.Errorf("store: create schema_migrations: %w", err)
	}

	migs, err := loadMigrations()
	if err != nil {
		return err
	}

	for _, m := range migs {
		applied, err := migrationApplied(conn, m.version)
		if err != nil {
			return err
		}
		if applied {
			continue
		}

		slog.LogAttrs(
			context.Background(),
			slog.LevelInfo,
			"applying migration",
			slog.Int("version", m.version),
			slog.String("name", m.name),
		)

		if err := runMigration(conn, m); err != nil {
			return fmt.Errorf("store: apply migration %04d_%s: %w", m.version, m.name, err)
		}
	}

	return nil
}

func loadMigrations() ([]migration, error) {
	entries, err := fs.ReadDir(migrationsFS, "migrations")
	if err != nil {
		return nil, fmt.Errorf("store: read migrations dir: %w", err)
	}

	var migs []migration
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}

		version, err := parseVersion(e.Name())
		if err != nil {
			return nil, err
		}

		content, err := migrationsFS.ReadFile("migrations/" + e.Name())
		if err != nil {
			return nil, fmt.Errorf("store: read migration %s: %w", e.Name(), err)
		}

		migs = append(migs, migration{
			version: version,
			name:    strings.TrimSuffix(e.Name(), ".sql"),
			sql:     string(content),
		})
	}

	sort.Slice(migs, func(i, j int) bool { return migs[i].version < migs[j].version })

	for i := 1; i < len(migs); i++ {
		if migs[i].version == migs[i-1].version {
			return nil, fmt.Errorf("store: duplicate migration version %d", migs[i].version)
		}
	}

	return migs, nil
}

// parseVersion reads the leading integer from a filename like "0001_init.sql".
func parseVersion(name string) (int, error) {
	base := strings.TrimSuffix(name, ".sql")
	num := base
	if idx := strings.Index(base, "_"); idx > 0 {
		num = base[:idx]
	}

	v, err := strconv.Atoi(num)
	if err != nil {
		return 0, fmt.Errorf("store: bad migration filename %q: version prefix is not an integer", name)
	}
	if v <= 0 {
		return 0, fmt.Errorf("store: bad migration filename %q: version must be positive", name)
	}
	return v, nil
}

func migrationApplied(conn *sql.DB, version int) (bool, error) {
	var exists bool
	err := conn.QueryRow(
		`SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = ?)`, version,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("store: check migration %d: %w", version, err)
	}
	return exists, nil
}

func runMigration(conn *sql.DB, m migration) error {
	tx, err := conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(m.sql); err != nil {
		return err
	}
	if _, err := tx.Exec(
		`INSERT INTO schema_migrations (version, name) VALUES (?, ?)`, m.version, m.name,
	); err != nil {
		return err
	}

	return tx.Commit()
}
