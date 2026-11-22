package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const probe = "s3cr3t-probe-value"

func clearEnv(t *testing.T) {
	t.Helper()
	t.Setenv("DATABASE_URL", "")
	t.Setenv("DATABASE_URL_FILE", "")
	os.Unsetenv("DATABASE_URL")
	os.Unsetenv("DATABASE_URL_FILE")
}

func TestSecretDirectPrecedence(t *testing.T) {
	clearEnv(t)
	t.Setenv("DATABASE_URL", probe)
	got, err := DatabaseURL()
	if err != nil || got != probe {
		t.Fatalf("direct = %q, %v", got, err)
	}
}

func TestSecretFilePrecedence(t *testing.T) {
	clearEnv(t)
	p := filepath.Join(t.TempDir(), "url")
	if err := os.WriteFile(p, []byte(probe+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DATABASE_URL_FILE", p)
	got, err := DatabaseURL()
	if err != nil || got != probe {
		t.Fatalf("file = %q, %v", got, err)
	}
}

func TestSecretConflictFailsClosed(t *testing.T) {
	clearEnv(t)
	t.Setenv("DATABASE_URL", probe)
	t.Setenv("DATABASE_URL_FILE", "/nonexistent")
	_, err := DatabaseURL()
	if err == nil {
		t.Fatal("conflicting variables accepted")
	}
	if strings.Contains(err.Error(), probe) {
		t.Fatalf("value leaked in error: %v", err)
	}
}

func TestSecretMissingFailsActionably(t *testing.T) {
	clearEnv(t)
	_, err := DatabaseURL()
	if err == nil {
		t.Fatal("missing credential accepted")
	}
	if !strings.Contains(err.Error(), "DATABASE_URL") {
		t.Fatalf("error names no variable: %v", err)
	}
}

func TestSecretUnreadableFile(t *testing.T) {
	clearEnv(t)
	t.Setenv("DATABASE_URL_FILE", filepath.Join(t.TempDir(), "absent"))
	_, err := DatabaseURL()
	if err == nil {
		t.Fatal("unreadable file accepted")
	}
	dir := t.TempDir()
	t.Setenv("DATABASE_URL_FILE", dir)
	if _, err := DatabaseURL(); err == nil {
		t.Fatal("directory as secret file accepted")
	}
}

func TestSecretWhitespaceAndEmpty(t *testing.T) {
	clearEnv(t)
	t.Setenv("DATABASE_URL", "  "+probe+" \n")
	got, err := DatabaseURL()
	if err != nil || got != probe {
		t.Fatalf("whitespace = %q, %v", got, err)
	}
	t.Setenv("DATABASE_URL", "  \n ")
	if _, err := DatabaseURL(); err == nil {
		t.Fatal("empty-after-trim accepted")
	}
}

func TestSecretRedaction(t *testing.T) {
	clearEnv(t)
	t.Setenv("DATABASE_URL", probe)
	t.Setenv("DATABASE_URL_FILE", "/nonexistent")
	_, err1 := DatabaseURL()
	clearEnv(t)
	if _, err := DatabaseURL(); strings.Contains(err.Error(), probe) {
		t.Fatalf("missing-value error leaks: %v", err)
	}
	if strings.Contains(err1.Error(), probe) {
		t.Fatalf("conflict error leaks: %v", err1)
	}
}

func TestDBOSSystemDatabaseURLPair(t *testing.T) {
	t.Setenv("DBOS_SYSTEM_DATABASE_URL", "")
	t.Setenv("DBOS_SYSTEM_DATABASE_URL_FILE", "")
	os.Unsetenv("DBOS_SYSTEM_DATABASE_URL")
	os.Unsetenv("DBOS_SYSTEM_DATABASE_URL_FILE")
	if _, err := DBOSSystemDatabaseURL(); err == nil {
		t.Fatal("missing DBOS credential accepted")
	}
	t.Setenv("DBOS_SYSTEM_DATABASE_URL", probe)
	if _, err := DBOSSystemDatabaseURL(); err != nil {
		t.Fatalf("direct DBOS credential rejected: %v", err)
	}
}
