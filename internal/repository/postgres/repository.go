package postgres

import (
	"database/sql"

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
