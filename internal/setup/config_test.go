package setup

import (
	"flag"
	"os"
	"path/filepath"
	"testing"
)

func resetFlags() {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
}

func writeTempConfig(t *testing.T, content string) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	return path
}

func TestLoadConfig_FromYAML(t *testing.T) {
	resetFlags()

	origPath := configFilePath
	t.Cleanup(func() { configFilePath = origPath })

	configFilePath = writeTempConfig(t,
		`run_address: ":8080"
database_uri: "postgres://db"
accrual_system_address: "http://accrual"
auth_secret_key: "yaml-secret"`)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.RunAddress != ":8080" {
		t.Fatalf("RunAddress = %q", cfg.RunAddress)
	}
	if cfg.DatabaseURI != "postgres://db" {
		t.Fatalf("DatabaseURI = %q", cfg.DatabaseURI)
	}
	if cfg.AccrualSystemAddress != "http://accrual" {
		t.Fatalf("AccrualSystemAddress = %q", cfg.AccrualSystemAddress)
	}
	if cfg.AuthSecretKey != "yaml-secret" {
		t.Fatalf("AuthSecretKey = %q", cfg.AuthSecretKey)
	}
}

func TestLoadConfig_EnvOverridesYAML(t *testing.T) {
	resetFlags()

	origPath := configFilePath
	t.Cleanup(func() { configFilePath = origPath })

	configFilePath = writeTempConfig(t,
		`run_address: ":8080"
database_uri: "postgres://db"
accrual_system_address: "http://accrual"
auth_secret_key: "yaml-secret"`)

	t.Setenv("RUN_ADDRESS", ":9090")
	t.Setenv("DATABASE_URI", "postgres://env-db")
	t.Setenv("SECRET_KEY", "env-secret")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.RunAddress != ":9090" {
		t.Fatalf("RunAddress = %q", cfg.RunAddress)
	}
	if cfg.DatabaseURI != "postgres://env-db" {
		t.Fatalf("DatabaseURI = %q", cfg.DatabaseURI)
	}
	if cfg.AuthSecretKey != "env-secret" {
		t.Fatalf("AuthSecretKey = %q", cfg.AuthSecretKey)
	}
}

func TestLoadConfig_FlagsOverrideAll(t *testing.T) {
	resetFlags()

	origPath := configFilePath
	t.Cleanup(func() { configFilePath = origPath })

	configFilePath = writeTempConfig(t,
		`run_address: ":8080"
database_uri: "postgres://db"
accrual_system_address: "http://accrual"
auth_secret_key: "yaml-secret"`)

	os.Args = []string{
		"cmd",
		"-a=:7070",
		"-d=postgres://flag-db",
		"-r=http://flag-accrual",
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.RunAddress != ":7070" {
		t.Fatalf("RunAddress = %q", cfg.RunAddress)
	}
	if cfg.DatabaseURI != "postgres://flag-db" {
		t.Fatalf("DatabaseURI = %q", cfg.DatabaseURI)
	}
	if cfg.AccrualSystemAddress != "http://flag-accrual" {
		t.Fatalf("AccrualSystemAddress = %q", cfg.AccrualSystemAddress)
	}
}

func TestLoadConfig_DefaultAuthSecretKey(t *testing.T) {
	resetFlags()

	origPath := configFilePath
	t.Cleanup(func() { configFilePath = origPath })

	configFilePath = writeTempConfig(t,
		`run_address: ":8080"
database_uri: "postgres://db"
accrual_system_address: "http://accrual"`)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.AuthSecretKey != defaultAuthSecretKey {
		t.Fatalf("AuthSecretKey = %q", cfg.AuthSecretKey)
	}
}

func TestLoadConfig_ValidationError(t *testing.T) {
	resetFlags()

	origPath := configFilePath
	t.Cleanup(func() { configFilePath = origPath })

	t.Setenv("RUN_ADDRESS", "")
	t.Setenv("DATABASE_URI", "")
	t.Setenv("ACCRUAL_SYSTEM_ADDRESS", "")
	t.Setenv("SECRET_KEY", "")

	configFilePath = writeTempConfig(t, ``)

	_, err := LoadConfig()
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
}
