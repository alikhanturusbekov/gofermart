package handler

import (
	"encoding/json"
	"errors"
	"github.com/alikhanturusbekov/gofermart/internal/exception"
	"github.com/alikhanturusbekov/gofermart/internal/middleware"
	"github.com/alikhanturusbekov/gofermart/internal/service"
	"github.com/alikhanturusbekov/gofermart/internal/validation"
	"github.com/google/uuid"
	"io"
	"net/http"
	"strings"
)

type Handler struct {
	authService    *service.AuthService
	loyaltyService *service.LoyaltyService
}

// NewHandler gets new main handler
func NewHandler(authService *service.AuthService, loyaltyService *service.LoyaltyService) *Handler {
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
		case errors.Is(err, exception.ErrRecordExists):
			http.Error(w, err.Error(), http.StatusConflict)
			return
		default:
			http.Error(w, err.Error(), http.StatusInternalServerError)
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
	// Gets userID from token
	userID := r.Context().Value(middleware.UserIDKey).(uuid.UUID)

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
		case errors.Is(err, exception.ErrOrderExistsByUser):
			w.WriteHeader(http.StatusOK)
			return
		case errors.Is(err, exception.ErrOrderExistsByOther):
			http.Error(w, err.Error(), http.StatusConflict)
			return
		default:
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusAccepted)
}

// GetUserOrders gets all orders owned by user
func (h *Handler) GetUserOrders(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(middleware.UserIDKey).(uuid.UUID)

	orders, err := h.loyaltyService.GetUserOrders(r.Context(), userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
	userID := r.Context().Value(middleware.UserIDKey).(uuid.UUID)

	userBalance, err := h.loyaltyService.GetUserBalance(r.Context(), userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(userBalance)
}

// Withdraw gets points from balance for specific order
func (h *Handler) Withdraw(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(middleware.UserIDKey).(uuid.UUID)

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
		case errors.Is(err, exception.ErrNotEnoughBalance):
			w.WriteHeader(http.StatusPaymentRequired)
			return
		default:
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
}

// GetUserWithdrawals gets all withdrawals owned by user
func (h *Handler) GetUserWithdrawals(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(middleware.UserIDKey).(uuid.UUID)

	withdrawals, err := h.loyaltyService.GetUserWithdrawals(r.Context(), userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
