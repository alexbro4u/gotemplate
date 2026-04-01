package testutil

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // register postgres driver
)

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func DSN() string {
	host := env("TEST_POSTGRES_HOST", "localhost")
	port := env("TEST_POSTGRES_PORT", "5433")
	user := env("TEST_POSTGRES_USER", "gotemplate_test")
	password := env("TEST_POSTGRES_PASSWORD", "gotemplate_test")
	db := env("TEST_POSTGRES_DB", "gotemplate_test")
	return fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable",
		user,
		password,
		net.JoinHostPort(host, port),
		db,
	)
}

func NewTestDB(t *testing.T) *sqlx.DB {
	t.Helper()
	db, err := sqlx.Connect("postgres", DSN())
	if err != nil {
		t.Fatalf("connect to test db: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := db.Close(); closeErr != nil {
			t.Logf("close test db: %v", closeErr)
		}
	})
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
		data, readErr := os.ReadFile(filepath.Join(dir, name))
		if readErr != nil {
			t.Fatalf("read migration %s: %v", name, readErr)
		}
		if _, execErr := db.Exec(string(data)); execErr != nil {
			t.Fatalf("exec migration %s: %v", name, execErr)
		}
	}
}

func TruncateAll(t *testing.T, db *sqlx.DB) {
	t.Helper()
	tables := []string{
		"password_reset_tokens",
		"token_blacklist",
		"user_groups",
		"request_cache",
		"users",
		"groups",
	}
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
