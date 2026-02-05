package handler_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/alikhanturusbekov/gofermart/internal/entity"
	"github.com/alikhanturusbekov/gofermart/internal/handler"
	"github.com/alikhanturusbekov/gofermart/internal/handler/mocks"
	"github.com/alikhanturusbekov/gofermart/internal/middleware"
	"github.com/alikhanturusbekov/gofermart/internal/repository/postgres"
	"github.com/alikhanturusbekov/gofermart/internal/service"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"net/http"
	"net/http/httptest"
	"testing"
)

func makeRequest(method, body string, userID uuid.UUID) *http.Request {
	req := httptest.NewRequest(method, "/", bytes.NewBufferString(body))
	ctx := middleware.WithUserID(req.Context(), userID)
	return req.WithContext(ctx)
}

func TestRegister(t *testing.T) {
	mockAuth := mocks.NewAuthService(t)

	mockAuth.On("Register", mock.Anything, "user", "pass").Return("token", nil)
	mockAuth.On("Register", mock.Anything, "exists", "pass").Return("", postgres.ErrUserAlreadyExists)

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
	mockAuth := mocks.NewAuthService(t)
	mockAuth.On("Login", mock.Anything, "user", "pass").Return("token", nil)
	mockAuth.On("Login", mock.Anything, "fail", "pass").Return("", errors.New("invalid"))

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
	mockLoyalty := mocks.NewLoyaltyService(t)

	mockLoyalty.On("UploadOrder", mock.Anything, userID, "79927398713").Return(nil)
	mockLoyalty.On("UploadOrder", mock.Anything, userID, "4111111111111111").Return(service.ErrOrderExistsByUser)
	mockLoyalty.On("UploadOrder", mock.Anything, userID, "4222222222222").Return(service.ErrOrderExistsByOther)

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
	mockLoyalty := mocks.NewLoyaltyService(t)
	mockLoyalty.On("GetUserOrders", mock.Anything, userID).Return([]*entity.Order{{Number: "ORD1"}}, nil)

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
	mockLoyalty := mocks.NewLoyaltyService(t)
	mockLoyalty.On("GetUserBalance", mock.Anything, userID).Return(&entity.UserBalance{Current: 100, Withdrawn: 20}, nil)

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
	mockLoyalty := mocks.NewLoyaltyService(t)
	mockLoyalty.On("Withdraw", mock.Anything, userID, "79927398713", 40.0).Return(nil)
	mockLoyalty.On("Withdraw", mock.Anything, userID, "79927398713", 60.0).Return(service.ErrNotEnoughBalance)

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
	mockLoyalty := mocks.NewLoyaltyService(t)
	mockLoyalty.On("GetUserWithdrawals", mock.Anything, userID).Return([]*entity.Withdrawal{
		{OrderNumber: "ORD1", Sum: floatPtr(10)},
	}, nil)
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
