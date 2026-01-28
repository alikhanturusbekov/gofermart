package handler

import (
	"encoding/json"
	"errors"
	"github.com/alikhanturusbekov/gofermart/internal/repository"
	"github.com/alikhanturusbekov/gofermart/internal/service"
	"github.com/google/uuid"
	"net/http"
)

type Handler struct {
	authenticationService *service.AuthenticationService
}

func NewHandler(authenticationService *service.AuthenticationService) *Handler {
	return &Handler{
		authenticationService: authenticationService,
	}
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Login    string `json:"login"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	userID, err := h.authenticationService.Register(r.Context(), request.Login, request.Password)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrLoginExists):
			http.Error(w, err.Error(), http.StatusConflict)
			return
		default:
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusOK) // According to task 200, not 201
	json.NewEncoder(w).Encode(map[string]uuid.UUID{"user_id": *userID})
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Login    string `json:"login"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	token, err := h.authenticationService.Login(r.Context(), request.Login, request.Password)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	w.Header().Set("Authorization", "Bearer "+token)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"token": token})
}
