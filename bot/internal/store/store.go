package store

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"qq/anapa2006/internal/db"

	_ "github.com/mattn/go-sqlite3"
)

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

	if err := applyMigrations(conn); err != nil {
		conn.Close()
		return nil, err
	}

	return &Store{
		Queries: db.New(conn),
		Conn:    conn,
	}, nil
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
