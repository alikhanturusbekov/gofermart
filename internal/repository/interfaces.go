package repository

import (
	"context"
	"errors"
	"github.com/alikhanturusbekov/gofermart/internal/entity"
)

var (
	ErrLoginExists = errors.New("login already exists")
)

// Repository main interface to work with repositories
type Repository interface {
	User() UserRepository
}

// UserRepository interface to work with user repository
type UserRepository interface {
	Create(ctx context.Context, login, password string) (*entity.User, error)
	GetByLogin(ctx context.Context, login string) (*entity.User, error)
}
