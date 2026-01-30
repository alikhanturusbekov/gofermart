package postgres

import (
	"context"
	"database/sql"
	"errors"
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
func (r *OrderRepository) Create(ctx context.Context, userID uuid.UUID, number string) (*entity.Order, error) {
	query := `
		INSERT INTO orders (user_id, number, status)
		VALUES ($1, $2, $3)
		RETURNING id, user_id, orders.number, status, accrual, uploaded_at
	`

	order := &entity.Order{}
	err := r.scanOrder(r.database.QueryRowContext(ctx, query, userID, number, entity.StatusNew), order)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, exception.ErrRecordExists
		}
		return nil, fmt.Errorf("failed to create order: %w", err)
	}

	return order, nil
}

// GetByNumber returns the order by number
func (r *OrderRepository) GetByNumber(ctx context.Context, number string) (*entity.Order, error) {
	query := `
		SELECT id, user_id, number, status, accrual, uploaded_at
		FROM orders
		WHERE orders.number = $1
	`

	order := &entity.Order{}
	err := r.scanOrder(r.database.QueryRowContext(ctx, query, number), order)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}

		return nil, fmt.Errorf("failed to get order by number: %w", err)
	}

	return order, nil
}

// UpdateOrderStatus updates order status
func (r *OrderRepository) UpdateOrderStatus(ctx context.Context, number string, status entity.OrderStatus) (*entity.Order, error) {
	query := `
		UPDATE orders
		SET status = $2
		WHERE number = $1
		RETURNING id, user_id, number, status, accrual, uploaded_at
	`

	order := &entity.Order{}
	err := r.scanOrder(
		r.database.QueryRowContext(ctx, query, number, status),
		order,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update order status: %w", err)
	}

	return order, nil
}

// MarkOrderProcessedTx updates order status and inserts accrual
func (r *OrderRepository) MarkOrderProcessedTx(ctx context.Context, tx *sql.Tx, number string, accrual *float64) (*entity.Order, error) {
	query := `
		UPDATE orders
		SET status = $2, accrual = $3
		WHERE number = $1
		RETURNING id, user_id, number, status, accrual, uploaded_at
	`

	order := &entity.Order{}
	err := r.scanOrder(
		tx.QueryRowContext(ctx, query, number, entity.StatusProcessed, accrual),
		order,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to mark order processed: %w", err)
	}

	return order, nil
}

// GetAllByUser gets all orders by user
func (r *OrderRepository) GetAllByUser(ctx context.Context, userID uuid.UUID) ([]*entity.Order, error) {
	query := `
		SELECT id, user_id, number, status, accrual, uploaded_at
		FROM orders
		WHERE user_id = $1
		ORDER BY uploaded_at DESC
	`

	rows, err := r.database.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get orders by user: %w", err)
	}
	defer rows.Close()

	var orders []*entity.Order
	for rows.Next() {
		order := &entity.Order{}
		if err := r.scanOrder(rows, order); err != nil {
			return nil, fmt.Errorf("failed to scan order: %w", err)
		}
		orders = append(orders, order)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return orders, nil
}

// scanOrder gets order entity from row
func (r *OrderRepository) scanOrder(s repository.Scanner, order *entity.Order) error {
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
