package database

import (
	"context"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pkg/errors"
	"time"
)

const (
	DefaultDBRetries    = 3
	DefaultDBRetryDelay = 50 * time.Millisecond
)

type RetryFunc func() error

// WithRetry adds retries to database queries
func WithRetry(
	ctx context.Context,
	attempts int,
	delay time.Duration,
	fn RetryFunc,
) error {
	var err error

	for attempt := 0; attempt < attempts; attempt++ {
		if err = fn(); err == nil {
			return nil
		}

		if !isRetryableDBError(err) {
			return err
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Duration(attempt) * delay):
		}
	}

	return err
}

// IsRetryableDBError checks if the error from database is retryable
func isRetryableDBError(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "40001", "40P01":
			return true
		}
	}
	return false
}
