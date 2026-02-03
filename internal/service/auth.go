package service

import (
	"context"
	"errors"
	"github.com/alikhanturusbekov/gofermart/internal/repository"
	"github.com/dgrijalva/jwt-go"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"time"
)

// AuthService service to work with authentication
type AuthService struct {
	repository repository.Repository
	secretKey  string
}

func NewAuthService(repository repository.Repository, secretKey string) *AuthService {
	return &AuthService{
		repository: repository,
		secretKey:  secretKey,
	}
}

// Register registers the user to the database
func (s *AuthService) Register(ctx context.Context, login, password string) (string, error) {
	// Hashes the password
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	// Adds user to the database
	user, err := s.repository.User().CreateUserWithBalance(ctx, login, string(passwordHash))
	if err != nil {
		return "", err
	}

	// Generates authentication token
	token, err := s.generateToken(user.ID)
	if err != nil {
		return "", err
	}

	return token, nil
}

// Login authenticates the user
func (s *AuthService) Login(ctx context.Context, login, password string) (string, error) {
	// Searches for the user
	user, err := s.repository.User().GetByLogin(ctx, login)
	if err != nil {
		return "", errors.New("user not found")
	}

	// Checks the password
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return "", errors.New("incorrect password")
	}

	// Generates authentication token
	token, err := s.generateToken(user.ID)
	if err != nil {
		return "", err
	}

	return token, nil
}

// generateToken generates JWT token
func (s *AuthService) generateToken(userID uuid.UUID) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID.String(),
		"exp":     time.Now().Add(time.Hour * 24).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.secretKey))
}
