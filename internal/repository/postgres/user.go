package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/alikhanturusbekov/gofermart/internal/entity"
	"github.com/alikhanturusbekov/gofermart/internal/repository"
	"github.com/google/uuid"
)

var (
	ErrUserAlreadyExists = fmt.Errorf("user already exists")
)

// UserRepository implementation with database
type UserRepository struct {
	database *sql.DB
}

// CreateUserWithBalance create user and balance
func (r *UserRepository) CreateUserWithBalance(ctx context.Context, login, password string) (*entity.User, error) {
	tx, err := r.database.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Create user
	queryUser := `
        INSERT INTO users (login, password)
        VALUES ($1, $2)
        RETURNING id, login, password, created_at
    `
	user := &entity.User{}
	err = r.scanUser(tx.QueryRowContext(ctx, queryUser, login, password), user)
	if err != nil {
		tx.Rollback()
		if isUniqueViolation(err) {
			return nil, ErrUserAlreadyExists
		}
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Create user balance
	queryBalance := `
        INSERT INTO user_balances (user_id)
        VALUES ($1)
    `
	_, err = tx.ExecContext(ctx, queryBalance, user.ID)
	if err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to create user balance: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
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

// GetUserBalance gets user balance
func (r *UserRepository) GetUserBalance(ctx context.Context, userID uuid.UUID) (*entity.UserBalance, error) {
	query := `
		SELECT user_id, user_balances.current, withdrawn
		FROM user_balances
		WHERE user_id = $1
	`

	userBalance := &entity.UserBalance{}
	err := r.scanUserBalance(r.database.QueryRowContext(ctx, query, userID), userBalance)
	if err != nil {
		return nil, fmt.Errorf("failed to get user balance: %w", err)
	}

	return userBalance, nil
}

// AddUserBalanceTx adds current balance within transaction
func (r *UserRepository) AddUserBalanceTx(ctx context.Context, tx *sql.Tx, userID uuid.UUID, accrual *float64) error {
	query := `
		UPDATE user_balances 
		SET current = current + $1, updated_at = now()
		WHERE user_id = $2
		RETURNING user_id, user_balances.current, withdrawn`
	_, err := tx.ExecContext(ctx, query, accrual, userID)
	return err
}

// SubtractUserBalanceTx subtract points from the user balance
func (r *UserRepository) SubtractUserBalanceTx(ctx context.Context, tx *sql.Tx, userID uuid.UUID, withdrawalAmount float64) error {
	query := `
		UPDATE user_balances 
		SET current = current - $1, withdrawn = withdrawn + $1, updated_at = now()
		WHERE user_id = $2
		RETURNING user_id, user_balances.current, withdrawn`
	_, err := tx.ExecContext(ctx, query, withdrawalAmount, userID)
	return err
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

// scanUserBalance gets userBalance entity from row
func (r *UserRepository) scanUserBalance(s repository.Scanner, userBalance *entity.UserBalance) error {
	return s.Scan(
		&userBalance.UserID,
		&userBalance.Current,
		&userBalance.Withdrawn,
	)
}
