package handler

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/alikhanturusbekov/gofermart/internal/entity"
	"github.com/alikhanturusbekov/gofermart/internal/middleware/auth"
	"github.com/alikhanturusbekov/gofermart/internal/repository/postgres"
	"github.com/alikhanturusbekov/gofermart/internal/service"
	"github.com/alikhanturusbekov/gofermart/internal/validation"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"io"
	"net/http"
	"strings"
)

// AuthService to work with authentication
type AuthService interface {
	Register(ctx context.Context, login, password string) (string, error)
	Login(ctx context.Context, login, password string) (string, error)
}

// LoyaltyService to work with orders, bonuses and withdrawals
type LoyaltyService interface {
	UploadOrder(ctx context.Context, userID uuid.UUID, number string) error
	GetUserOrders(ctx context.Context, userID uuid.UUID) ([]*entity.Order, error)
	GetUserBalance(ctx context.Context, userID uuid.UUID) (*entity.UserBalance, error)
	Withdraw(ctx context.Context, userID uuid.UUID, orderNumber string, withdrawalAmount float64) error
	GetUserWithdrawals(ctx context.Context, userID uuid.UUID) ([]*entity.Withdrawal, error)
}

type Handler struct {
	authService    AuthService
	loyaltyService LoyaltyService
}

// NewHandler gets new main handler
func NewHandler(authService AuthService, loyaltyService LoyaltyService) *Handler {
	return &Handler{
		authService:    authService,
		loyaltyService: loyaltyService,
	}
}

// Register registers and returns authentication token for user
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Login    string `json:"login"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	token, err := h.authService.Register(r.Context(), request.Login, request.Password)
	if err != nil {
		switch {
		case errors.Is(err, postgres.ErrUserAlreadyExists):
			http.Error(w, err.Error(), http.StatusConflict)
			return
		default:
			logrus.
				WithError(err).
				WithField("login", request.Login).
				Error("error while registering user")
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Authorization", "Bearer "+token)
	w.WriteHeader(http.StatusOK) // According to task 200, not 201
	json.NewEncoder(w).Encode(map[string]string{"token": token})
}

// Login returns authentication token for user
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Login    string `json:"login"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	token, err := h.authService.Login(r.Context(), request.Login, request.Password)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	w.Header().Set("Authorization", "Bearer "+token)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"token": token})
}

// UploadOrder uploads an order for user
func (h *Handler) UploadOrder(w http.ResponseWriter, r *http.Request) {
	// Gets userID from context
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}

	// Reads order number
	body, err := io.ReadAll(r.Body)
	if err != nil || len(body) == 0 {
		http.Error(w, "incorrect or empty input for order number", http.StatusBadRequest)
		return
	}
	orderNumber := strings.TrimSpace(string(body))

	// Validates order number
	if valid := validation.ValidateOrderNumber(orderNumber); !valid {
		http.Error(w, "invalid order number", http.StatusUnprocessableEntity)
		return
	}

	// Uploads user order
	err = h.loyaltyService.UploadOrder(r.Context(), userID, orderNumber)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrOrderExistsByUser):
			w.WriteHeader(http.StatusOK)
			return
		case errors.Is(err, service.ErrOrderExistsByOther):
			http.Error(w, err.Error(), http.StatusConflict)
			return
		default:
			logrus.
				WithError(err).
				WithField("userID", userID).
				WithField("orderNumber", orderNumber).
				Error("error while uploading order by user")

			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusAccepted)
}

// GetUserOrders gets all orders owned by user
func (h *Handler) GetUserOrders(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}

	orders, err := h.loyaltyService.GetUserOrders(r.Context(), userID)
	if err != nil {
		logrus.
			WithError(err).
			WithField("userID", userID).
			Error("error while getting all orders by user")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if len(orders) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(orders)
}

// GetUserBalance gets user balance
func (h *Handler) GetUserBalance(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}

	userBalance, err := h.loyaltyService.GetUserBalance(r.Context(), userID)
	if err != nil {
		logrus.
			WithError(err).
			WithField("userID", userID).
			Error("error while getting user balance")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(userBalance)
}

// Withdraw gets points from balance for specific order
func (h *Handler) Withdraw(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}

	var request struct {
		Order string  `json:"order"`
		Sum   float64 `json:"sum"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validates order number
	if valid := validation.ValidateOrderNumber(request.Order); !valid {
		http.Error(w, "invalid order number", http.StatusUnprocessableEntity)
		return
	}

	// Withdraws from user balance
	err := h.loyaltyService.Withdraw(r.Context(), userID, request.Order, request.Sum)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrNotEnoughBalance):
			w.WriteHeader(http.StatusPaymentRequired)
			return
		default:
			logrus.
				WithError(err).
				WithField("order", request.Order).
				WithField("sum", request.Sum).
				WithField("userID", userID).
				Error("error while withdrawing points from user balance")
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
}

// GetUserWithdrawals gets all withdrawals owned by user
func (h *Handler) GetUserWithdrawals(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}

	withdrawals, err := h.loyaltyService.GetUserWithdrawals(r.Context(), userID)
	if err != nil {
		logrus.
			WithError(err).
			WithField("userID", userID).
			Error("error while getting user withdrawals")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if len(withdrawals) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(withdrawals)
}
