package service

import (
	"context"
	"github.com/alikhanturusbekov/gofermart/internal/entity"
	"github.com/alikhanturusbekov/gofermart/internal/exception"
	"github.com/alikhanturusbekov/gofermart/internal/repository"
	"github.com/alikhanturusbekov/gofermart/internal/worker"
	"github.com/google/uuid"
)

type LoyaltyService struct {
	repository         repository.Repository
	orderProcessWorker *worker.OrderProcessWorker
}

// NewLoyaltyService creates service to work with orders and withdrawals
func NewLoyaltyService(repository repository.Repository, orderProcessWorker *worker.OrderProcessWorker) *LoyaltyService {
	return &LoyaltyService{repository: repository, orderProcessWorker: orderProcessWorker}
}

func (s *LoyaltyService) UploadOrder(ctx context.Context, userID uuid.UUID, number string) error {
	existingOrder, err := s.repository.Order().GetByNumber(ctx, number)
	if err != nil {
		return err
	}

	// If order already exists check the owner
	if existingOrder != nil {
		if existingOrder.UserID == userID {
			return exception.ErrOrderExistsByUser
		} else {
			return exception.ErrOrderExistsByOther
		}
	}

	_, err = s.repository.Order().Create(ctx, userID, number)
	if err != nil {
		return err
	}

	s.orderProcessWorker.Enqueue(entity.OrderProcessTask{Number: number})

	return nil
}

// GetUserOrders gets all orders owned by user
func (s *LoyaltyService) GetUserOrders(ctx context.Context, userID uuid.UUID) ([]*entity.Order, error) {
	orders, err := s.repository.Order().GetAllByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	return orders, nil
}

// GetUserBalance gets user balance
func (s *LoyaltyService) GetUserBalance(ctx context.Context, userID uuid.UUID) (*entity.UserBalance, error) {
	userBalance, err := s.repository.User().GetUserBalance(ctx, userID)
	if err != nil {
		return nil, err
	}
	return userBalance, nil
}

// Withdraw withdraws points from user balance
func (s *LoyaltyService) Withdraw(ctx context.Context, userID uuid.UUID, orderNumber string, withdrawalAmount float64) error {
	// Gets user balance
	userBalance, err := s.repository.User().GetUserBalance(ctx, userID)
	if err != nil {
		return err
	}

	// Compares withdrawal amount and balance
	if userBalance.Current < withdrawalAmount {
		return err
	}

	// Begins transaction
	tx, err := s.repository.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Subtract withdrawal from balance
	err = s.repository.User().SubtractUserBalanceTx(ctx, tx, userID, withdrawalAmount)
	if err != nil {
		tx.Rollback()
		return err
	}

	// Create withdrawal record
	_, err = s.repository.Withdrawal().CreateTx(ctx, tx, userID, orderNumber, withdrawalAmount)
	if err != nil {
		tx.Rollback()
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

// GetUserWithdrawals gets user withdrawals
func (s *LoyaltyService) GetUserWithdrawals(ctx context.Context, userID uuid.UUID) ([]*entity.Withdrawal, error) {
	withdrawals, err := s.repository.Withdrawal().GetAllByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	return withdrawals, nil
}
