package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/alikhanturusbekov/gofermart/internal/entity"
	"github.com/alikhanturusbekov/gofermart/internal/exception"
	"github.com/alikhanturusbekov/gofermart/internal/repository"
)

// UserRepository implementation with database
type UserRepository struct {
	database *sql.DB
}

// Create saves user data to database
func (r *UserRepository) Create(ctx context.Context, login, password string) (*entity.User, error) {
	query := `
		INSERT INTO users (login, password)
		VALUES ($1, $2)
		RETURNING id, login, password, created_at
	`

	user := &entity.User{}
	err := r.scanUser(r.database.QueryRowContext(ctx, query, login, password), user)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, exception.ErrRecordExists
		}
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

// GetByLogin gets one user by login
func (r *UserRepository) GetByLogin(ctx context.Context, login string) (*entity.User, error) {
	query := `
		SELECT id, login, password, created_at
		FROM users
		WHERE login = $1
	`

	user := &entity.User{}
	err := r.scanUser(r.database.QueryRowContext(ctx, query, login), user)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by login: %w", err)
	}

	return user, nil
}

// scanUser gets user entity from row
func (r *UserRepository) scanUser(s repository.Scanner, user *entity.User) error {
	return s.Scan(
		&user.ID,
		&user.Login,
		&user.Password,
		&user.CreatedAt,
	)
}
