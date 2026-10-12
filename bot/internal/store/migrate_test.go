package store

import "testing"

func TestApplyMigrationsIdempotent(t *testing.T) {
	st, err := Open(":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer st.Close()

	tables := []string{
		"allowed_users", "sources", "posts", "post_media", "target_channels",
		"post_drafts", "draft_media", "schedule", "pending_replies", "schema_migrations",
	}
	for _, table := range tables {
		var name string
		if err := st.Conn.QueryRow(
			`SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?`, table,
		).Scan(&name); err != nil {
			t.Errorf("table %q missing: %v", table, err)
		}
	}

	var count int
	if err := st.Conn.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE version = 1`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Errorf("expected migration 1 to be recorded, got %d rows", count)
	}

	// Re-running is a no-op (idempotent).
	if err := applyMigrations(st.Conn); err != nil {
		t.Fatalf("re-apply migrations: %v", err)
	}

	// Simulate adopting a pre-migrations database: it has the schema but no
	// schema_migrations table. The runner should recreate the bookkeeping table
	// and adopt migration 1 without error.
	if _, err := st.Conn.Exec(`DROP TABLE schema_migrations`); err != nil {
		t.Fatal(err)
	}
	if err := applyMigrations(st.Conn); err != nil {
		t.Fatalf("adopt existing schema: %v", err)
	}
	if err := st.Conn.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE version = 1`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Errorf("expected migration 1 to be adopted, got %d rows", count)
	}
}

func TestParseVersion(t *testing.T) {
	cases := map[string]int{
		"0001_init.sql":  1,
		"0002_add_x.sql": 2,
		"42.sql":         42,
	}
	for name, want := range cases {
		got, err := parseVersion(name)
		if err != nil || got != want {
			t.Errorf("parseVersion(%q) = %d, %v; want %d", name, got, err, want)
		}
	}

	for _, name := range []string{"init.sql", "0_init.sql", "000x.sql"} {
		if _, err := parseVersion(name); err == nil {
			t.Errorf("parseVersion(%q) expected error, got nil", name)
		}
	}
}

func TestLoadMigrations(t *testing.T) {
	migs, err := loadMigrations()
	if err != nil {
		t.Fatalf("load migrations: %v", err)
	}
	if len(migs) == 0 {
		t.Fatal("expected at least one migration")
	}
	if migs[0].version != 1 {
		t.Errorf("first migration version = %d, want 1", migs[0].version)
	}
}
