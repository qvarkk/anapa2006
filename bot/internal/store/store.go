package store

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"qq/anapa2006/internal/db"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

//go:embed schema.sql
var schemaSQL string

type Store struct {
	*db.Queries
	Conn *sql.DB
}

func Open(path string) (*Store, error) {
	// ":memory:" is a special SQLite name; resolving it through filepath.Abs
	// would turn it into a regular file named ":memory:" in the working
	// directory, so keep it verbatim.
	dsn := path
	if path != ":memory:" {
		absPath, err := filepath.Abs(path)
		if err != nil {
			return nil, fmt.Errorf("store: resolve path %s: %w", path, err)
		}

		if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
			return nil, fmt.Errorf("store: create db directory: %w", err)
		}
		dsn = absPath
	}

	slog.LogAttrs(
		context.Background(),
		slog.LevelInfo,
		"opening database...",
		slog.String("path", dsn),
	)

	conn, err := sql.Open("sqlite3", dsn+"?_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("store: open %s: %w", dsn, err)
	}

	conn.SetMaxOpenConns(1)

	if err := conn.Ping(); err != nil {
		return nil, fmt.Errorf("store: ping %s: %w", dsn, err)
	}
	if _, err := conn.Exec(schemaSQL); err != nil {
		return nil, fmt.Errorf("store: apply schema: %w", err)
	}

	if err := verifySchema(conn); err != nil {
		conn.Close()
		return nil, err
	}

	return &Store{
		Queries: db.New(conn),
		Conn:    conn,
	}, err
}

// verifySchema rejects databases created before a schema change. The schema is
// applied with CREATE TABLE IF NOT EXISTS, so an existing database silently keeps
// its old shape; without this check the mismatch surfaces later as a confusing
// constraint or foreign key error.
func verifySchema(conn *sql.DB) error {
	checks := []struct {
		table   string
		marker  string
		present bool // whether the marker must appear in the stored DDL
	}{
		{"pending_replies", "awaiting_channel_handle", true},
		{"pending_replies", "awaiting_media_add", true},
		{"post_drafts", "user_id", false},
		{"post_drafts", "target_channel_id", true},
		{"schedule", "target_chat_id", false},
		{"schedule", "sent_message_id", true},
		{"target_channels", "chat_id", true},
	}

	for _, c := range checks {
		var ddl string
		err := conn.QueryRow(
			`SELECT sql FROM sqlite_master WHERE type = 'table' AND name = ?`, c.table,
		).Scan(&ddl)
		if err != nil {
			return fmt.Errorf("store: read schema of table %q: %w", c.table, err)
		}

		if strings.Contains(ddl, c.marker) != c.present {
			return fmt.Errorf(
				"store: outdated database schema: table %q does not match the current schema (%q must be %s); "+
					"delete the database file (or the docker volume) and start again",
				c.table, c.marker, presentLabel(c.present),
			)
		}
	}

	return nil
}

func presentLabel(present bool) string {
	if present {
		return "present"
	}
	return "absent"
}

func (s *Store) Close() error {
	return s.Conn.Close()
}

func (s *Store) WithTx(ctx context.Context, fn func(*db.Queries) error) (err error) {
	tx, err := s.Conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("store: begin tx: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
		if err != nil {
			_ = tx.Rollback()
			return
		}
		err = tx.Commit()
	}()

	err = fn(s.Queries.WithTx(tx))
	return err
}
