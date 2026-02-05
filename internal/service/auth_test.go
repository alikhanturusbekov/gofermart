package service_test

import (
	"context"
	"errors"
	"github.com/alikhanturusbekov/gofermart/internal/entity"
	"github.com/alikhanturusbekov/gofermart/internal/service"
	"github.com/alikhanturusbekov/gofermart/internal/service/mocks"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	"testing"
)

func TestAuthService_Register_Success(t *testing.T) {
	userID := uuid.New()

	mockUserRepo := mocks.NewUserRepository(t)
	mockRepo := mocks.NewRepository(t)
	mockRepo.On("User").Return(mockUserRepo)

	mockUserRepo.On("CreateUserWithBalance", mock.Anything, "test", mock.Anything).Return(&entity.User{
		ID:    userID,
		Login: "test",
	}, nil)

	svc := service.NewAuthService(mockRepo, "secret")

	token, err := svc.Register(context.Background(), "test", "password")
	require.NoError(t, err)
	require.NotEmpty(t, token)
}

func TestAuthService_Register_RepoError(t *testing.T) {
	mockUserRepo := mocks.NewUserRepository(t)
	mockRepo := mocks.NewRepository(t)
	mockRepo.On("User").Return(mockUserRepo)

	mockUserRepo.On("CreateUserWithBalance", mock.Anything, "test", mock.Anything).
		Return(nil, errors.New("db error"))

	svc := service.NewAuthService(mockRepo, "secret")

	token, err := svc.Register(context.Background(), "test", "password")
	require.Error(t, err)
	require.Empty(t, token)
}

func TestAuthService_Login_Success(t *testing.T) {
	password := "password"
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	mockUserRepo := mocks.NewUserRepository(t)
	mockRepo := mocks.NewRepository(t)
	mockRepo.On("User").Return(mockUserRepo)

	mockUserRepo.On("GetByLogin", mock.Anything, "test").Return(&entity.User{
		ID:       uuid.New(),
		Login:    "test",
		Password: string(hash),
	}, nil)

	svc := service.NewAuthService(mockRepo, "secret")

	token, err := svc.Login(context.Background(), "test", password)
	require.NoError(t, err)
	require.NotEmpty(t, token)
}

func TestAuthService_Login_UserNotFound(t *testing.T) {
	mockUserRepo := mocks.NewUserRepository(t)
	mockRepo := mocks.NewRepository(t)
	mockRepo.On("User").Return(mockUserRepo)

	mockUserRepo.On("GetByLogin", mock.Anything, "test").Return(nil, errors.New("not found"))

	svc := service.NewAuthService(mockRepo, "secret")

	token, err := svc.Login(context.Background(), "test", "password")
	require.Error(t, err)
	require.Empty(t, token)
}

func TestAuthService_Login_WrongPassword(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("correct"), bcrypt.DefaultCost)

	mockUserRepo := mocks.NewUserRepository(t)
	mockRepo := mocks.NewRepository(t)
	mockRepo.On("User").Return(mockUserRepo)

	mockUserRepo.On("GetByLogin", mock.Anything, "test").Return(&entity.User{
		ID:       uuid.New(),
		Login:    "test",
		Password: string(hash),
	}, nil)

	svc := service.NewAuthService(mockRepo, "secret")

	token, err := svc.Login(context.Background(), "test", "wrong")
	require.Error(t, err)
	require.Empty(t, token)
}
