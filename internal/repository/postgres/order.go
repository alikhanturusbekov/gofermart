package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/alikhanturusbekov/gofermart/internal/entity"
	"github.com/alikhanturusbekov/gofermart/internal/exception"
	"github.com/alikhanturusbekov/gofermart/internal/repository"
	"github.com/google/uuid"
)

type OrderRepository struct {
	database *sql.DB
}

// Create saves a new order
func (or *OrderRepository) Create(ctx context.Context, userID uuid.UUID, number string) (*entity.Order, error) {
	query := `
		INSERT INTO orders (user_id, number, status)
		VALUES ($1, $2, $3)
		RETURNING id, user_id, orders.number, status, accrual, uploaded_at
	`

	order := &entity.Order{}
	err := or.scanOrder(or.database.QueryRowContext(ctx, query, userID, number, entity.StatusNew), order)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, exception.ErrRecordExists
		}
		return nil, fmt.Errorf("failed to create order: %w", err)
	}

	return order, nil
}

// GetByNumber returns the order by number
func (or *OrderRepository) GetByNumber(ctx context.Context, number string) (*entity.Order, error) {
	query := `
		SELECT id, user_id, number, status, accrual, uploaded_at
		FROM orders
		WHERE orders.number = $1
	`

	order := &entity.Order{}
	err := or.scanOrder(or.database.QueryRowContext(ctx, query, number), order)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by login: %w", err)
	}

	return order, nil
}

// scanOrder gets order entity from row
func (or *OrderRepository) scanOrder(s repository.Scanner, order *entity.Order) error {
	var accrual sql.NullFloat64 // helper for nullable float64

	if err := s.Scan(
		&order.ID,
		&order.UserID,
		&order.Number,
		&order.Status,
		&accrual,
		&order.UploadedAt,
	); err != nil {
		return err
	}

	if accrual.Valid {
		order.Accrual = &accrual.Float64
	} else {
		order.Accrual = nil
	}

	return nil
}
