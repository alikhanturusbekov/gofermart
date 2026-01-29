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

type AuthenticationService struct {
	repository repository.Repository
	secretKey  string
}

func NewAuthenticationService(repository repository.Repository, secretKey string) *AuthenticationService {
	return &AuthenticationService{
		repository: repository,
		secretKey:  secretKey,
	}
}

// Register registers the user to the database
func (as *AuthenticationService) Register(ctx context.Context, login, password string) (string, error) {
	// Hashes the password
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	// Adds user to the database
	user, err := as.repository.User().Create(ctx, login, string(passwordHash))
	if err != nil {
		return "", err
	}

	// Generates authentication token
	token, err := as.generateToken(user.ID)
	if err != nil {
		return "", err
	}

	return token, nil
}

// Login authenticates the user
func (as *AuthenticationService) Login(ctx context.Context, login, password string) (string, error) {
	// Searches for the user
	user, err := as.repository.User().GetByLogin(ctx, login)
	if err != nil {
		return "", errors.New("user not found")
	}

	// Checks the password
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return "", errors.New("incorrect password")
	}

	// Generates authentication token
	token, err := as.generateToken(user.ID)
	if err != nil {
		return "", err
	}

	return token, nil
}

// generateToken generates JWT token
func (as *AuthenticationService) generateToken(userID uuid.UUID) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID.String(),
		"exp":     time.Now().Add(time.Hour * 24).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(as.secretKey))
}
