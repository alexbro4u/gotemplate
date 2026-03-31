package testutil

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func DSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		env("TEST_POSTGRES_USER", "gotemplate_test"),
		env("TEST_POSTGRES_PASSWORD", "gotemplate_test"),
		env("TEST_POSTGRES_HOST", "localhost"),
		env("TEST_POSTGRES_PORT", "5432"),
		env("TEST_POSTGRES_DB", "gotemplate_test"),
	)
}

func NewTestDB(t *testing.T) *sqlx.DB {
	t.Helper()
	db, err := sqlx.Connect("postgres", DSN())
	if err != nil {
		t.Fatalf("connect to test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func MigrationsDir() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "..", "..", "migrations")
}

func RunMigrations(t *testing.T, db *sqlx.DB) {
	t.Helper()
	dir := MigrationsDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read migrations dir: %v", err)
	}

	var upFiles []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".up.sql") {
			upFiles = append(upFiles, e.Name())
		}
	}
	sort.Strings(upFiles)

	for _, name := range upFiles {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("read migration %s: %v", name, err)
		}
		if _, err := db.Exec(string(data)); err != nil {
			t.Fatalf("exec migration %s: %v", name, err)
		}
	}
}

func TruncateAll(t *testing.T, db *sqlx.DB) {
	t.Helper()
	tables := []string{"user_groups", "request_cache", "users", "groups"}
	for _, table := range tables {
		if _, err := db.Exec(fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table)); err != nil {
			t.Fatalf("truncate %s: %v", table, err)
		}
	}
}

func SeedGroups(t *testing.T, db *sqlx.DB) {
	t.Helper()
	_, err := db.Exec(`INSERT INTO groups (name) VALUES ('users'), ('admin') ON CONFLICT (name) DO NOTHING`)
	if err != nil {
		t.Fatalf("seed groups: %v", err)
	}
}
