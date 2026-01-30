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

func (ls *LoyaltyService) UploadOrder(ctx context.Context, userID uuid.UUID, number string) error {
	existingOrder, err := ls.repository.Order().GetByNumber(ctx, number)
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

	_, err = ls.repository.Order().Create(ctx, userID, number)
	if err != nil {
		return err
	}

	ls.orderProcessWorker.Enqueue(entity.OrderProcessTask{Number: number})

	return nil
}
