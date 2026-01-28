package setup

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/jackc/pgx/v5/stdlib"
)

const (
	driverName    = "pgx"
	migrationsDir = "migrations"
)

// PrepareDatabase sets up database and migrations
func PrepareDatabase(config *Config) (*sql.DB, error) {
	database, err := sql.Open(driverName, config.DatabaseURI)
	if err != nil {
		return nil, err
	}

	err = applyMigrations(database, migrationsDir)
	if err != nil {
		return nil, err
	}

	return database, nil
}

// applyMigrations reads the migrations file and applies changes to database
func applyMigrations(db *sql.DB, dir string) error {
	_, err := db.Exec(`
        CREATE TABLE IF NOT EXISTS schema_migrations (
            version VARCHAR(255) PRIMARY KEY
        );
    `)
	if err != nil {
		return fmt.Errorf("failed to create schema_migrations table: %w", err)
	}

	files, err := filepath.Glob(filepath.Join(dir, "*.up.sql"))
	if err != nil {
		return err
	}

	for _, file := range files {
		version := filepath.Base(file)

		var exists string
		err := db.QueryRow("SELECT version FROM schema_migrations WHERE version=$1", version).Scan(&exists)
		if err == nil {
			continue
		} else if err != sql.ErrNoRows {
			return fmt.Errorf("failed to check migration %s: %w", version, err)
		}

		sqlBytes, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", file, err)
		}

		_, err = db.Exec(string(sqlBytes))
		if err != nil {
			return fmt.Errorf("failed to apply migration %s: %w", file, err)
		}

		_, err = db.Exec("INSERT INTO schema_migrations(version) VALUES ($1)", version)
		if err != nil {
			return fmt.Errorf("failed to record applied migration %s: %w", version, err)
		}
	}

	return nil
}
