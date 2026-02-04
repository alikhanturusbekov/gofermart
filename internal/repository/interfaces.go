package repository

import (
	"context"
	"database/sql"
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
	Withdrawal() WithdrawalRepository

	BeginTx(ctx context.Context) (*sql.Tx, error)
}

// UserRepository interface to work with users
type UserRepository interface {
	CreateUserWithBalance(ctx context.Context, login, password string) (*entity.User, error)
	GetByLogin(ctx context.Context, login string) (*entity.User, error)
	GetUserBalance(ctx context.Context, userID uuid.UUID) (*entity.UserBalance, error)
	AddUserBalanceTx(ctx context.Context, tx *sql.Tx, userID uuid.UUID, accrual *float64) error
	SubtractUserBalanceTx(ctx context.Context, tx *sql.Tx, userID uuid.UUID, withdrawalAmount float64) (bool, error)
}

// OrderRepository interface to work with orders
type OrderRepository interface {
	Create(ctx context.Context, userID uuid.UUID, number string) (*entity.Order, error)
	GetByNumber(ctx context.Context, number string) (*entity.Order, error)
	GetAllByUser(ctx context.Context, userID uuid.UUID) ([]*entity.Order, error)
	UpdateOrderStatus(ctx context.Context, number string, status entity.OrderStatus) (*entity.Order, error)
	MarkOrderProcessedTx(ctx context.Context, tx *sql.Tx, number string, accrual *float64) (*entity.Order, error)
}

// WithdrawalRepository interface to work with withdrawals
type WithdrawalRepository interface {
	CreateTx(ctx context.Context, tx *sql.Tx, userID uuid.UUID, orderNumber string, sum float64) (*entity.Withdrawal, error)
	GetAllByUser(ctx context.Context, userID uuid.UUID) ([]*entity.Withdrawal, error)
}
