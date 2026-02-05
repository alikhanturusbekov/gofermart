package setup

import (
	"database/sql"
	"errors"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
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

// applyMigrations runs migrations on the database
func applyMigrations(db *sql.DB, dir string) error {
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return err
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://"+dir,
		"postgres",
		driver,
	)
	if err != nil {
		return err
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}

	return nil
}
