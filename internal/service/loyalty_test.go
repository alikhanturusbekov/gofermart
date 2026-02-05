package service_test

import (
	"context"
	"github.com/alikhanturusbekov/gofermart/internal/entity"
	"github.com/alikhanturusbekov/gofermart/internal/service"
	"github.com/alikhanturusbekov/gofermart/internal/service/mocks"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestLoyaltyService_UploadOrder(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	orderNumber := "ORD1"
	otherUser := uuid.New()

	mockOrderRepo := mocks.NewOrderRepository(t)
	mockRepo := mocks.NewRepository(t)
	mockWorker := &mocks.OrderProcessor{}

	mockRepo.On("Order").Return(mockOrderRepo)

	svc := service.NewLoyaltyService(mockRepo, mockWorker)

	mockOrderRepo.On("GetByNumber", ctx, orderNumber).Return(nil, nil).Once()
	mockOrderRepo.On("Create", ctx, userID, orderNumber).
		Return(&entity.Order{UserID: userID, Number: orderNumber}, nil).Once()
	mockWorker.On("Enqueue", mock.AnythingOfType("entity.OrderProcessTask")).Return().Once()

	err := svc.UploadOrder(ctx, userID, orderNumber)
	require.NoError(t, err)
	mockWorker.AssertCalled(t, "Enqueue", mock.MatchedBy(func(task entity.OrderProcessTask) bool {
		return task.Number == orderNumber
	}))

	// Upload same order again by same user -> ErrOrderExistsByUser
	mockOrderRepo.On("GetByNumber", ctx, orderNumber).
		Return(&entity.Order{UserID: userID, Number: orderNumber}, nil).Once()
	err = svc.UploadOrder(ctx, userID, orderNumber)
	require.ErrorIs(t, err, service.ErrOrderExistsByUser)

	// Upload same order by different user -> ErrOrderExistsByOther
	mockOrderRepo.On("GetByNumber", ctx, orderNumber).
		Return(&entity.Order{UserID: userID, Number: orderNumber}, nil).Once()
	err = svc.UploadOrder(ctx, otherUser, orderNumber)
	require.ErrorIs(t, err, service.ErrOrderExistsByOther)
}

func TestLoyaltyService_GetUserOrders(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	mockOrderRepo := mocks.NewOrderRepository(t)
	mockRepo := mocks.NewRepository(t)

	mockRepo.On("Order").Return(mockOrderRepo)

	mockOrders := []*entity.Order{
		{UserID: userID, Number: "ORD1"},
		{UserID: userID, Number: "ORD2"},
	}
	mockOrderRepo.On("GetAllByUser", ctx, userID).Return(mockOrders, nil)

	svc := service.NewLoyaltyService(mockRepo, nil)
	orders, err := svc.GetUserOrders(ctx, userID)
	require.NoError(t, err)
	require.Len(t, orders, 2)
}

func TestLoyaltyService_GetUserBalance(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	mockUserRepo := mocks.NewUserRepository(t)
	mockRepo := mocks.NewRepository(t)
	mockRepo.On("User").Return(mockUserRepo)

	mockBalance := &entity.UserBalance{Current: 100, Withdrawn: 0}
	mockUserRepo.On("GetUserBalance", ctx, userID).Return(mockBalance, nil)

	svc := service.NewLoyaltyService(mockRepo, nil)
	balance, err := svc.GetUserBalance(ctx, userID)
	require.NoError(t, err)
	require.Equal(t, 100.0, balance.Current)
}

func TestLoyaltyService_Withdraw(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	orderNumber := "ORD999"

	mockUserRepo := mocks.NewUserRepository(t)
	mockWithdrawalRepo := mocks.NewWithdrawalRepository(t)
	mockRepo := mocks.NewRepository(t)

	mockRepo.On("User").Return(mockUserRepo)
	mockRepo.On("Withdrawal").Return(mockWithdrawalRepo)
	mockRepo.On("BeginTx", ctx).Return(nil, nil)

	// Not enough balance
	mockUserRepo.On("SubtractUserBalanceTx", ctx, mock.Anything, userID, 100.0).Return(false, nil)
	svc := service.NewLoyaltyService(mockRepo, nil)
	err := svc.Withdraw(ctx, userID, orderNumber, 100)
	require.ErrorIs(t, err, service.ErrNotEnoughBalance)

	// Successful withdrawal
	//mockRepo.On("BeginTx", ctx).Return(nil, nil)
	mockUserRepo.On("SubtractUserBalanceTx", ctx, mock.Anything, userID, 30.0).Return(true, nil)
	mockWithdrawalRepo.On("CreateTx", ctx, mock.Anything, userID, orderNumber, 30.0).
		Return(&entity.Withdrawal{UserID: userID, OrderNumber: orderNumber, Sum: newFloat64(30)}, nil)

	err = svc.Withdraw(ctx, userID, orderNumber, 30)
	require.NoError(t, err)
}

func TestLoyaltyService_GetUserWithdrawals(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	mockWithdrawalRepo := mocks.NewWithdrawalRepository(t)
	mockRepo := mocks.NewRepository(t)
	mockRepo.On("Withdrawal").Return(mockWithdrawalRepo)

	sum1, sum2 := 10.0, 20.0
	mockWithdrawals := []*entity.Withdrawal{
		{UserID: userID, OrderNumber: "ORD1", Sum: &sum1},
		{UserID: userID, OrderNumber: "ORD2", Sum: &sum2},
	}
	mockWithdrawalRepo.On("GetAllByUser", ctx, userID).Return(mockWithdrawals, nil)

	svc := service.NewLoyaltyService(mockRepo, nil)
	withdrawals, err := svc.GetUserWithdrawals(ctx, userID)
	require.NoError(t, err)
	require.Len(t, withdrawals, 2)
}

// helper for pointers
func newFloat64(f float64) *float64 { return &f }
