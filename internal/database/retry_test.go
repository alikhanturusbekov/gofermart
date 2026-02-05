package database

import (
	"context"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pkg/errors"
	"testing"
)

func TestIsRetryableDBError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "retryable serialization failure",
			err:  &pgconn.PgError{Code: "40001"},
			want: true,
		},
		{
			name: "retryable deadlock",
			err:  &pgconn.PgError{Code: "40P01"},
			want: true,
		},
		{
			name: "non-retryable pg error",
			err:  &pgconn.PgError{Code: "23505"},
			want: false,
		},
		{
			name: "non-pg error",
			err:  errors.New("some error"),
			want: false,
		},
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isRetryableDBError(tt.err); got != tt.want {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWithRetry_SuccessFirstAttempt(t *testing.T) {
	ctx := context.Background()

	calls := 0
	err := WithRetry(ctx, 3, 0, func() error {
		calls++
		return nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 1 {
		t.Fatalf("calls = %d, want 1", calls)
	}
}

func TestWithRetry_RetryableThenSuccess(t *testing.T) {
	ctx := context.Background()

	calls := 0
	err := WithRetry(ctx, 3, 0, func() error {
		calls++
		if calls < 2 {
			return &pgconn.PgError{Code: "40001"}
		}
		return nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 2 {
		t.Fatalf("calls = %d, want 2", calls)
	}
}

func TestWithRetry_NonRetryableError(t *testing.T) {
	ctx := context.Background()

	calls := 0
	expected := errors.New("fatal error")

	err := WithRetry(ctx, 5, 0, func() error {
		calls++
		return expected
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, expected) {
		t.Fatalf("err = %v", err)
	}
	if calls != 1 {
		t.Fatalf("calls = %d, want 1", calls)
	}
}
