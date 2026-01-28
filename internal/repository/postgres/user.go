package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/alikhanturusbekov/gofermart/internal/entity"
	"github.com/alikhanturusbekov/gofermart/internal/repository"
	"strings"
)

// UserRepository implementation with database
type UserRepository struct {
	database *sql.DB
}

// scanner interface to get entities from database rows
type scanner interface {
	Scan(dest ...any) error
}

// Create saves user data to database
func (ur *UserRepository) Create(ctx context.Context, login, password string) (*entity.User, error) {
	query := `
		INSERT INTO users (login, password)
		VALUES ($1, $2)
		RETURNING id, login, password, created_at
	`

	user := &entity.User{}
	err := ur.scanUser(ur.database.QueryRowContext(ctx, query, login, password), user)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, repository.ErrLoginExists
		}
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

// GetByLogin gets one user by login
func (ur *UserRepository) GetByLogin(ctx context.Context, login string) (*entity.User, error) {
	query := `
		SELECT id, login, password, created_at
		FROM users
		WHERE login = $1
	`

	user := &entity.User{}
	err := ur.scanUser(ur.database.QueryRowContext(ctx, query, login), user)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by login: %w", err)
	}

	return user, nil
}

// scanUser gets user entity from row
func (ur *UserRepository) scanUser(s scanner, user *entity.User) error {
	return s.Scan(
		&user.ID,
		&user.Login,
		&user.Password,
		&user.CreatedAt,
	)
}

// isUniqueViolation checks if the error indicates violation of unique constraints
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}

	return strings.Contains(err.Error(), "SQLSTATE 23505")
}
