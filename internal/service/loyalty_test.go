package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"database/sql"

	"github.com/alikhanturusbekov/gofermart/internal/entity"
	"github.com/alikhanturusbekov/gofermart/internal/exception"
	"github.com/alikhanturusbekov/gofermart/internal/repository"
	"github.com/google/uuid"
)

// Mocking repositories and worker

type MockUserRepo struct {
	balances map[uuid.UUID]*entity.UserBalance
}

func (m *MockUserRepo) CreateUserWithBalance(ctx context.Context, login, password string) (*entity.User, error) {
	return &entity.User{Login: login}, nil
}
func (m *MockUserRepo) GetByLogin(ctx context.Context, login string) (*entity.User, error) {
	return &entity.User{Login: login}, nil
}
func (m *MockUserRepo) GetUserBalance(ctx context.Context, userID uuid.UUID) (*entity.UserBalance, error) {
	b, ok := m.balances[userID]
	if !ok {
		b = &entity.UserBalance{Current: 0, Withdrawn: 0}
		m.balances[userID] = b
	}
	return b, nil
}
func (m *MockUserRepo) AddUserBalanceTx(ctx context.Context, tx *sql.Tx, userID uuid.UUID, accrual *float64) error {
	b, _ := m.balances[userID]
	if accrual != nil {
		b.Current += *accrual
	}
	return nil
}
func (m *MockUserRepo) SubtractUserBalanceTx(ctx context.Context, tx *sql.Tx, userID uuid.UUID, withdrawalAmount float64) error {
	b, ok := m.balances[userID]
	if !ok || b.Current < withdrawalAmount {
		return exception.ErrNotEnoughBalance
	}
	b.Current -= withdrawalAmount
	b.Withdrawn += withdrawalAmount
	return nil
}

type MockOrderRepo struct {
	orders     map[string]*entity.Order
	userOrders map[uuid.UUID][]*entity.Order
}

func (m *MockOrderRepo) Create(ctx context.Context, userID uuid.UUID, number string) (*entity.Order, error) {
	o := &entity.Order{UserID: userID, Number: number}
	m.orders[number] = o
	m.userOrders[userID] = append(m.userOrders[userID], o)
	return o, nil
}
func (m *MockOrderRepo) GetByNumber(ctx context.Context, number string) (*entity.Order, error) {
	return m.orders[number], nil
}
func (m *MockOrderRepo) GetAllByUser(ctx context.Context, userID uuid.UUID) ([]*entity.Order, error) {
	return m.userOrders[userID], nil
}
func (m *MockOrderRepo) UpdateOrderStatus(ctx context.Context, number string, status entity.OrderStatus) (*entity.Order, error) {
	return m.orders[number], nil
}
func (m *MockOrderRepo) MarkOrderProcessedTx(ctx context.Context, tx *sql.Tx, number string, accrual *float64) (*entity.Order, error) {
	return m.orders[number], nil
}

type MockWithdrawalRepo struct {
	withdrawals map[uuid.UUID][]*entity.Withdrawal
}

func (m *MockWithdrawalRepo) CreateTx(ctx context.Context, tx *sql.Tx, userID uuid.UUID, orderNumber string, sum float64) (*entity.Withdrawal, error) {
	w := &entity.Withdrawal{
		UserID:      userID,
		OrderNumber: orderNumber,
		Sum:         &sum,
		ProcessedAt: time.Now(),
	}
	m.withdrawals[userID] = append(m.withdrawals[userID], w)
	return w, nil
}
func (m *MockWithdrawalRepo) GetAllByUser(ctx context.Context, userID uuid.UUID) ([]*entity.Withdrawal, error) {
	return m.withdrawals[userID], nil
}

type MockRepository struct {
	userRepo       *MockUserRepo
	orderRepo      *MockOrderRepo
	withdrawalRepo *MockWithdrawalRepo
}

func (m *MockRepository) User() repository.UserRepository              { return m.userRepo }
func (m *MockRepository) Order() repository.OrderRepository            { return m.orderRepo }
func (m *MockRepository) Withdrawal() repository.WithdrawalRepository  { return m.withdrawalRepo }
func (m *MockRepository) BeginTx(ctx context.Context) (*sql.Tx, error) { return nil, nil }

type MockWorker struct {
	enqueued []entity.OrderProcessTask
}

func (w *MockWorker) Enqueue(task entity.OrderProcessTask) { w.enqueued = append(w.enqueued, task) }

// Setups the service for tests

func setupService() (*LoyaltyService, *MockRepository, *MockWorker) {
	userRepo := &MockUserRepo{balances: map[uuid.UUID]*entity.UserBalance{}}
	orderRepo := &MockOrderRepo{orders: map[string]*entity.Order{}, userOrders: map[uuid.UUID][]*entity.Order{}}
	withdrawRepo := &MockWithdrawalRepo{withdrawals: map[uuid.UUID][]*entity.Withdrawal{}}
	repo := &MockRepository{userRepo, orderRepo, withdrawRepo}
	worker := &MockWorker{}
	service := NewLoyaltyService(repo, worker)
	return service, repo, worker
}

// Tests start here

func TestLoyaltyService_UploadOrder(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	service, _, worker := setupService()

	err := service.UploadOrder(ctx, userID, "ORD1")
	if err != nil {
		t.Fatal(err)
	}
	if len(worker.enqueued) != 1 || worker.enqueued[0].Number != "ORD1" {
		t.Fatal("order not enqueued correctly")
	}

	// Upload same order again by same user -> ErrOrderExistsByUser
	err = service.UploadOrder(ctx, userID, "ORD1")
	if !errors.Is(err, exception.ErrOrderExistsByUser) {
		t.Fatal("expected ErrOrderExistsByUser")
	}

	// Upload same order by different user -> ErrOrderExistsByOther
	err = service.UploadOrder(ctx, uuid.New(), "ORD1")
	if !errors.Is(err, exception.ErrOrderExistsByOther) {
		t.Fatal("expected ErrOrderExistsByOther")
	}
}

func TestLoyaltyService_GetUserOrders(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	service, repo, _ := setupService()

	// Add orders manually
	repo.orderRepo.userOrders[userID] = []*entity.Order{
		{UserID: userID, Number: "ORD1"},
		{UserID: userID, Number: "ORD2"},
	}

	orders, err := service.GetUserOrders(ctx, userID)
	if err != nil {
		t.Fatal(err)
	}
	if len(orders) != 2 {
		t.Fatal("expected 2 orders")
	}
}

func TestLoyaltyService_GetUserBalance(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	service, repo, _ := setupService()

	repo.userRepo.balances[userID] = &entity.UserBalance{Current: 100, Withdrawn: 0}
	balance, err := service.GetUserBalance(ctx, userID)
	if err != nil {
		t.Fatal(err)
	}
	if balance.Current != 100 {
		t.Fatal("balance incorrect")
	}
}

func TestLoyaltyService_Withdraw(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	service, repo, _ := setupService()

	repo.userRepo.balances[userID] = &entity.UserBalance{Current: 50, Withdrawn: 0}

	// Withdraw more than balance
	err := service.Withdraw(ctx, userID, "ORD999", 100)
	if !errors.Is(err, exception.ErrNotEnoughBalance) {
		t.Fatal("expected ErrNotEnoughBalance")
	}

	// Valid withdrawal
	err = service.Withdraw(ctx, userID, "ORD999", 30)
	if err != nil {
		t.Fatal(err)
	}

	balance, _ := service.GetUserBalance(ctx, userID)
	if balance.Current != 20 || balance.Withdrawn != 30 {
		t.Fatal("balance not updated correctly")
	}

	withdrawals, _ := service.GetUserWithdrawals(ctx, userID)
	if len(withdrawals) != 1 || *withdrawals[0].Sum != 30 {
		t.Fatal("withdrawal not recorded correctly")
	}
}

func TestLoyaltyService_GetUserWithdrawals(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	service, repo, _ := setupService()

	// Add withdrawals manually
	sum1, sum2 := 10.0, 20.0
	repo.withdrawalRepo.withdrawals[userID] = []*entity.Withdrawal{
		{UserID: userID, OrderNumber: "ORD1", Sum: &sum1},
		{UserID: userID, OrderNumber: "ORD2", Sum: &sum2},
	}

	withdrawals, err := service.GetUserWithdrawals(ctx, userID)
	if err != nil {
		t.Fatal(err)
	}
	if len(withdrawals) != 2 {
		t.Fatal("expected 2 withdrawals")
	}
}
