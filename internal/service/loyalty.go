package service

import (
	"context"
	"github.com/alikhanturusbekov/gofermart/internal/exception"
	"github.com/alikhanturusbekov/gofermart/internal/repository"
	"github.com/google/uuid"
)

type LoyaltyService struct {
	repository repository.Repository
}

// NewLoyaltyService creates service to work with orders and withdrawals
func NewLoyaltyService(repository repository.Repository) *LoyaltyService {
	return &LoyaltyService{repository: repository}
}

func (ls *LoyaltyService) UploadOrder(ctx context.Context, userID uuid.UUID, number string) error {
	existingOrder, err := ls.repository.Order().GetByNumber(ctx, number)

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

	return nil
}
