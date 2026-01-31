package service

import (
	"context"
	"database/sql"
	"errors"
	"github.com/alikhanturusbekov/gofermart/internal/entity"
	"github.com/alikhanturusbekov/gofermart/internal/repository"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	"testing"
)

type mockUserRepository struct {
	createUser func(ctx context.Context, login, password string) (*entity.User, error)
	getByLogin func(ctx context.Context, login string) (*entity.User, error)
}

func (m *mockUserRepository) CreateUserWithBalance(
	ctx context.Context,
	login, password string,
) (*entity.User, error) {
	return m.createUser(ctx, login, password)
}

func (m *mockUserRepository) GetByLogin(
	ctx context.Context,
	login string,
) (*entity.User, error) {
	return m.getByLogin(ctx, login)
}

func (m *mockUserRepository) GetUserBalance(
	ctx context.Context,
	userID uuid.UUID,
) (*entity.UserBalance, error) {
	return nil, errors.New("unexpected call to GetUserBalance")
}

func (m *mockUserRepository) AddUserBalanceTx(
	ctx context.Context,
	tx *sql.Tx,
	userID uuid.UUID,
	accrual *float64,
) error {
	return errors.New("unexpected call to AddUserBalanceTx")
}

func (m *mockUserRepository) SubtractUserBalanceTx(
	ctx context.Context,
	tx *sql.Tx,
	userID uuid.UUID,
	withdrawalAmount float64,
) error {
	return errors.New("unexpected call to SubtractUserBalanceTx")
}

type mockRepository struct {
	user repository.UserRepository
}

func (m *mockRepository) User() repository.UserRepository {
	return m.user
}

func (m *mockRepository) Order() repository.OrderRepository {
	panic("Order() not used in auth service test")
}

func (m *mockRepository) Withdrawal() repository.WithdrawalRepository {
	panic("Withdrawal() not used in auth service test")
}

func (m *mockRepository) BeginTx(ctx context.Context) (*sql.Tx, error) {
	panic("BeginTx() not used in auth service test")
}

func TestAuthService_Register_Success(t *testing.T) {
	userID := uuid.New()

	userRepo := &mockUserRepository{
		createUser: func(ctx context.Context, login, password string) (*entity.User, error) {
			require.Equal(t, "test", login)
			require.NotEmpty(t, password) // hashed password

			return &entity.User{
				ID:    userID,
				Login: login,
			}, nil
		},
	}

	repo := &mockRepository{user: userRepo}
	service := NewAuthService(repo, "secret")

	token, err := service.Register(context.Background(), "test", "password")

	require.NoError(t, err)
	require.NotEmpty(t, token)
}

func TestAuthService_Register_RepoError(t *testing.T) {
	userRepo := &mockUserRepository{
		createUser: func(ctx context.Context, login, password string) (*entity.User, error) {
			return nil, errors.New("db error")
		},
	}

	repo := &mockRepository{user: userRepo}
	service := NewAuthService(repo, "secret")

	token, err := service.Register(context.Background(), "test", "password")

	require.Error(t, err)
	require.Empty(t, token)
}

func TestAuthService_Login_Success(t *testing.T) {
	password := "password"
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	userRepo := &mockUserRepository{
		getByLogin: func(ctx context.Context, login string) (*entity.User, error) {
			return &entity.User{
				ID:       uuid.New(),
				Login:    login,
				Password: string(hash),
			}, nil
		},
	}

	repo := &mockRepository{user: userRepo}
	service := NewAuthService(repo, "secret")

	token, err := service.Login(context.Background(), "test", password)

	require.NoError(t, err)
	require.NotEmpty(t, token)
}

func TestAuthService_Login_UserNotFound(t *testing.T) {
	userRepo := &mockUserRepository{
		getByLogin: func(ctx context.Context, login string) (*entity.User, error) {
			return nil, errors.New("not found")
		},
	}

	repo := &mockRepository{user: userRepo}
	service := NewAuthService(repo, "secret")

	token, err := service.Login(context.Background(), "test", "password")

	require.Error(t, err)
	require.Empty(t, token)
}

func TestAuthService_Login_WrongPassword(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("correct"), bcrypt.DefaultCost)

	userRepo := &mockUserRepository{
		getByLogin: func(ctx context.Context, login string) (*entity.User, error) {
			return &entity.User{
				ID:       uuid.New(),
				Login:    login,
				Password: string(hash),
			}, nil
		},
	}

	repo := &mockRepository{user: userRepo}
	service := NewAuthService(repo, "secret")

	token, err := service.Login(context.Background(), "test", "wrong")

	require.Error(t, err)
	require.Empty(t, token)
}
