package postgres

import (
	"context"
	"database/sql"
	"strings"

	"github.com/alikhanturusbekov/gofermart/internal/repository"
)

// Repository implementation with database
type Repository struct {
	database *sql.DB
}

// NewRepository returns postgres repository
func NewRepository(database *sql.DB) *Repository {
	return &Repository{database: database}
}

// User returns user repository
func (r *Repository) User() repository.UserRepository {
	return &UserRepository{database: r.database}
}

// Order returns order repository
func (r *Repository) Order() repository.OrderRepository {
	return &OrderRepository{database: r.database}
}

// BeginTx begins transaction
func (r *Repository) BeginTx(ctx context.Context) (*sql.Tx, error) {
	return r.database.BeginTx(ctx, nil)
}

// isUniqueViolation checks if the error indicates violation of unique constraints
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}

	return strings.Contains(err.Error(), "SQLSTATE 23505")
}
