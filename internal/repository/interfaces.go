package repository

import (
	"context"
	"github.com/alikhanturusbekov/gofermart/internal/entity"
	"github.com/google/uuid"
)

// Scanner interface to get entities from database rows
type Scanner interface {
	Scan(dest ...any) error
}

// Repository main interface to work with repositories
type Repository interface {
	User() UserRepository
	Order() OrderRepository
}

// UserRepository interface to work with user repository
type UserRepository interface {
	Create(ctx context.Context, login, password string) (*entity.User, error)
	GetByLogin(ctx context.Context, login string) (*entity.User, error)
}

// OrderRepository interface to work with order repository
type OrderRepository interface {
	Create(ctx context.Context, userID uuid.UUID, number string) (*entity.Order, error)
	GetByNumber(ctx context.Context, number string) (*entity.Order, error)
	GetAllByUser(ctx context.Context, userID uuid.UUID) ([]*entity.Order, error)
	UpdateOrderStatus(ctx context.Context, number string, status entity.OrderStatus) (*entity.Order, error)
	MarkOrderProcessed(ctx context.Context, number string, accrual *float64) (*entity.Order, error)
}
