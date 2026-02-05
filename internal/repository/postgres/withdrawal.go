package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/alikhanturusbekov/gofermart/internal/database"
	"github.com/alikhanturusbekov/gofermart/internal/entity"
	"github.com/alikhanturusbekov/gofermart/internal/repository"
	"github.com/google/uuid"
)

// WithdrawalRepository implementation with database
type WithdrawalRepository struct {
	database *sql.DB
}

// CreateTx creates withdrawal record within transaction
func (r *WithdrawalRepository) CreateTx(ctx context.Context, tx *sql.Tx, userID uuid.UUID, orderNumber string, sum float64) (*entity.Withdrawal, error) {
	query := `
		INSERT INTO withdrawals (user_id, order_number, sum)
		VALUES ($1, $2, $3)
		RETURNING id, user_id, order_number, sum, processed_at
	`

	withdrawal := &entity.Withdrawal{}
	err := r.scanWithdrawal(
		tx.QueryRowContext(ctx, query, userID, orderNumber, sum),
		withdrawal,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create withdrawal: %w", err)
	}

	return withdrawal, nil
}

// GetAllByUser gets user withdrawals
func (r *WithdrawalRepository) GetAllByUser(ctx context.Context, userID uuid.UUID) ([]*entity.Withdrawal, error) {
	query := `
		SELECT id, user_id, order_number, sum, processed_at
		FROM withdrawals
		WHERE user_id = $1
		ORDER BY processed_at DESC
	`

	var withdrawals []*entity.Withdrawal

	err := database.WithRetry(ctx, database.DefaultDBRetries, database.DefaultDBRetryDelay, func() error {
		rows, err := r.database.QueryContext(ctx, query, userID)
		if err != nil {
			return err
		}
		defer rows.Close()

		var tempWithdrawals []*entity.Withdrawal
		for rows.Next() {
			withdrawal := &entity.Withdrawal{}
			if scanErr := r.scanWithdrawal(rows, withdrawal); scanErr != nil {
				return scanErr
			}
			tempWithdrawals = append(tempWithdrawals, withdrawal)
		}

		if err := rows.Err(); err != nil {
			return err
		}

		withdrawals = tempWithdrawals
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get withdrawals by user: %w", err)
	}

	return withdrawals, nil
}

// scanWithdrawal gets withdrawal entity from row
func (r *WithdrawalRepository) scanWithdrawal(s repository.Scanner, withdrawal *entity.Withdrawal) error {
	return s.Scan(
		&withdrawal.ID,
		&withdrawal.UserID,
		&withdrawal.OrderNumber,
		&withdrawal.Sum,
		&withdrawal.ProcessedAt,
	)
}
