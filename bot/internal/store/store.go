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

	_ "github.com/mattn/go-sqlite3"
)

//go:embed schema.sql
var schemaSQL string

type Store struct {
	*db.Queries
	conn *sql.DB
}

func Open(path string) (*Store, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("store: resolve path %s: %w", path, err)
	}

	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		return nil, fmt.Errorf("store: create db directory: %w", err)
	}

	slog.LogAttrs(
		context.Background(),
		slog.LevelInfo,
		"opening database...",
		slog.String("path", absPath),
	)

	conn, err := sql.Open("sqlite3", absPath+"?_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("store: open %s: %w", absPath, err)
	}

	conn.SetMaxOpenConns(1)

	if err := conn.Ping(); err != nil {
		return nil, fmt.Errorf("store: ping %s: %w", absPath, err)
	}
	if _, err := conn.Exec(schemaSQL); err != nil {
		return nil, fmt.Errorf("store: apply schema: %w", err)
	}

	return &Store{
		Queries: db.New(conn),
		conn:    conn,
	}, err
}

func (s *Store) Close() error {
	return s.conn.Close()
}

func (s *Store) WithTx(ctx context.Context, fn func(*db.Queries) error) (err error) {
	tx, err := s.conn.BeginTx(ctx, nil)
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
