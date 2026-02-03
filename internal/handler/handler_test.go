package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/alikhanturusbekov/gofermart/internal/entity"
	"github.com/alikhanturusbekov/gofermart/internal/handler"
	"github.com/alikhanturusbekov/gofermart/internal/middleware"
	"github.com/alikhanturusbekov/gofermart/internal/repository/postgres"
	"github.com/alikhanturusbekov/gofermart/internal/service"
	"github.com/google/uuid"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Mocking services

type MockAuthService struct {
	RegisterFn func(ctx context.Context, login, password string) (string, error)
	LoginFn    func(ctx context.Context, login, password string) (string, error)
}

func (m *MockAuthService) Register(ctx context.Context, login, password string) (string, error) {
	return m.RegisterFn(ctx, login, password)
}

func (m *MockAuthService) Login(ctx context.Context, login, password string) (string, error) {
	return m.LoginFn(ctx, login, password)
}

type MockLoyaltyService struct {
	UploadOrderFn        func(ctx context.Context, userID uuid.UUID, number string) error
	GetUserOrdersFn      func(ctx context.Context, userID uuid.UUID) ([]*entity.Order, error)
	GetUserBalanceFn     func(ctx context.Context, userID uuid.UUID) (*entity.UserBalance, error)
	WithdrawFn           func(ctx context.Context, userID uuid.UUID, orderNumber string, withdrawalAmount float64) error
	GetUserWithdrawalsFn func(ctx context.Context, userID uuid.UUID) ([]*entity.Withdrawal, error)
}

func (m *MockLoyaltyService) UploadOrder(ctx context.Context, userID uuid.UUID, number string) error {
	return m.UploadOrderFn(ctx, userID, number)
}

func (m *MockLoyaltyService) GetUserOrders(ctx context.Context, userID uuid.UUID) ([]*entity.Order, error) {
	return m.GetUserOrdersFn(ctx, userID)
}

func (m *MockLoyaltyService) GetUserBalance(ctx context.Context, userID uuid.UUID) (*entity.UserBalance, error) {
	return m.GetUserBalanceFn(ctx, userID)
}

func (m *MockLoyaltyService) Withdraw(ctx context.Context, userID uuid.UUID, orderNumber string, withdrawalAmount float64) error {
	return m.WithdrawFn(ctx, userID, orderNumber, withdrawalAmount)
}

func (m *MockLoyaltyService) GetUserWithdrawals(ctx context.Context, userID uuid.UUID) ([]*entity.Withdrawal, error) {
	return m.GetUserWithdrawalsFn(ctx, userID)
}

func makeRequest(method, body string, userID uuid.UUID) *http.Request {
	req := httptest.NewRequest(method, "/", bytes.NewBufferString(body))
	ctx := context.WithValue(req.Context(), middleware.UserIDKey, userID)
	return req.WithContext(ctx)
}

// Tests start here

func TestRegister(t *testing.T) {
	mockAuth := &MockAuthService{
		RegisterFn: func(ctx context.Context, login, password string) (string, error) {
			if login == "exists" {
				return "", postgres.ErrUserAlreadyExists
			}
			return "token", nil
		},
	}

	h := handler.NewHandler(mockAuth, nil)

	// Successful register
	reqBody := `{"login":"user","password":"pass"}`
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	h.Register(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", w.Code)
	}

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["token"] != "token" {
		t.Errorf("expected token, got %s", resp["token"])
	}

	// Conflict case
	reqBody = `{"login":"exists","password":"pass"}`
	req = httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(reqBody))
	w = httptest.NewRecorder()

	h.Register(w, req)
	if w.Code != http.StatusConflict {
		t.Errorf("expected 409 Conflict, got %d", w.Code)
	}
}

func TestLogin(t *testing.T) {
	mockAuth := &MockAuthService{
		LoginFn: func(ctx context.Context, login, password string) (string, error) {
			if login == "fail" {
				return "", errors.New("invalid")
			}
			return "token", nil
		},
	}

	h := handler.NewHandler(mockAuth, nil)

	// Successful login
	reqBody := `{"login":"user","password":"pass"}`
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	h.Login(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", w.Code)
	}

	// Failed login
	reqBody = `{"login":"fail","password":"pass"}`
	req = httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(reqBody))
	w = httptest.NewRecorder()

	h.Login(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized, got %d", w.Code)
	}
}

func TestUploadOrder(t *testing.T) {
	userID := uuid.New()
	mockLoyalty := &MockLoyaltyService{
		UploadOrderFn: func(ctx context.Context, uid uuid.UUID, number string) error {
			if number == "4111111111111111" {
				return service.ErrOrderExistsByUser
			} else if number == "4222222222222" {
				return service.ErrOrderExistsByOther
			}
			return nil
		},
	}
	h := handler.NewHandler(nil, mockLoyalty)

	req := makeRequest(http.MethodPost, "79927398713", userID) // Valid Luhn
	w := httptest.NewRecorder()
	h.UploadOrder(w, req)
	if w.Code != http.StatusAccepted {
		t.Errorf("expected 202 Accepted, got %d", w.Code)
	}

	req = makeRequest(http.MethodPost, "4111111111111111", userID)
	w = httptest.NewRecorder()
	h.UploadOrder(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", w.Code)
	}

	req = makeRequest(http.MethodPost, "4222222222222", userID)
	w = httptest.NewRecorder()
	h.UploadOrder(w, req)
	if w.Code != http.StatusConflict {
		t.Errorf("expected 409 Conflict, got %d", w.Code)
	}

	req = makeRequest(http.MethodPost, "abc123", userID)
	w = httptest.NewRecorder()
	h.UploadOrder(w, req)
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422 Unprocessable Entity, got %d", w.Code)
	}
}

func TestGetUserOrders(t *testing.T) {
	userID := uuid.New()
	mockLoyalty := &MockLoyaltyService{
		GetUserOrdersFn: func(ctx context.Context, uid uuid.UUID) ([]*entity.Order, error) {
			if uid == userID {
				return []*entity.Order{{Number: "ORD1"}}, nil
			}
			return nil, nil
		},
	}
	h := handler.NewHandler(nil, mockLoyalty)

	req := makeRequest(http.MethodGet, "", userID)
	w := httptest.NewRecorder()
	h.GetUserOrders(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", w.Code)
	}

	var orders []*entity.Order
	json.NewDecoder(w.Body).Decode(&orders)
	if len(orders) != 1 || orders[0].Number != "ORD1" {
		t.Errorf("unexpected orders %+v", orders)
	}
}

func TestGetUserBalance(t *testing.T) {
	userID := uuid.New()
	mockLoyalty := &MockLoyaltyService{
		GetUserBalanceFn: func(ctx context.Context, uid uuid.UUID) (*entity.UserBalance, error) {
			return &entity.UserBalance{Current: 100, Withdrawn: 20}, nil
		},
	}
	h := handler.NewHandler(nil, mockLoyalty)

	req := makeRequest(http.MethodGet, "", userID)
	w := httptest.NewRecorder()
	h.GetUserBalance(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", w.Code)
	}

	var balance entity.UserBalance
	json.NewDecoder(w.Body).Decode(&balance)
	if balance.Current != 100 || balance.Withdrawn != 20 {
		t.Errorf("unexpected balance %+v", balance)
	}
}

func TestWithdraw(t *testing.T) {
	userID := uuid.New()
	mockLoyalty := &MockLoyaltyService{
		WithdrawFn: func(ctx context.Context, uid uuid.UUID, order string, sum float64) error {
			if sum > 50 {
				return service.ErrNotEnoughBalance
			}
			return nil
		},
	}
	h := handler.NewHandler(nil, mockLoyalty)

	body := `{"order":"79927398713","sum":40}`
	req := makeRequest(http.MethodPost, body, userID)
	w := httptest.NewRecorder()
	h.Withdraw(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", w.Code)
	}

	body = `{"order":"79927398713","sum":60}`
	req = makeRequest(http.MethodPost, body, userID)
	w = httptest.NewRecorder()
	h.Withdraw(w, req)
	if w.Code != http.StatusPaymentRequired {
		t.Errorf("expected 402 Payment Required, got %d", w.Code)
	}

	body = `{"order":"abc123","sum":10}`
	req = makeRequest(http.MethodPost, body, userID)
	w = httptest.NewRecorder()
	h.Withdraw(w, req)
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422 Unprocessable Entity, got %d", w.Code)
	}
}

func TestGetUserWithdrawals(t *testing.T) {
	userID := uuid.New()
	mockLoyalty := &MockLoyaltyService{
		GetUserWithdrawalsFn: func(ctx context.Context, uid uuid.UUID) ([]*entity.Withdrawal, error) {
			return []*entity.Withdrawal{
				{OrderNumber: "ORD1", Sum: floatPtr(10)},
			}, nil
		},
	}
	h := handler.NewHandler(nil, mockLoyalty)

	req := makeRequest(http.MethodGet, "", userID)
	w := httptest.NewRecorder()
	h.GetUserWithdrawals(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", w.Code)
	}

	var withdrawals []*entity.Withdrawal
	json.NewDecoder(w.Body).Decode(&withdrawals)
	if len(withdrawals) != 1 || *withdrawals[0].Sum != 10 {
		t.Errorf("unexpected withdrawals %+v", withdrawals)
	}
}

func floatPtr(f float64) *float64 {
	return &f
}
