package worker

import (
	"bytes"
	"context"
	"fmt"
	"github.com/alikhanturusbekov/gofermart/internal/client"
	"github.com/pkg/errors"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/alikhanturusbekov/gofermart/internal/entity"
	repoMocks "github.com/alikhanturusbekov/gofermart/internal/service/mocks"
)

func TestParseRetryAfter(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected time.Duration
	}{
		{
			name:     "empty header",
			value:    "",
			expected: defaultRetryAfter,
		},
		{
			name:     "valid seconds",
			value:    "1",
			expected: 1 * time.Second,
		},
		{
			name:     "invalid value",
			value:    "abc",
			expected: defaultRetryAfter,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := parseRetryAfter(tt.value)
			require.Equal(t, tt.expected, d)
		})
	}
}

func TestHandleSuccessfulResponse_InvalidJSON(t *testing.T) {
	ctx := context.Background()

	order := &entity.Order{
		Number: "123",
		Status: entity.StatusProcessing,
	}

	repo := repoMocks.NewRepository(t)

	resp := &http.Response{
		Body: http.NoBody,
	}

	worker := &OrderProcessor{
		repository: repo,
		in:         make(chan entity.OrderProcessTask, 1),
	}

	worker.handleSuccessfulResponse(ctx, order, resp)
}

func TestHandleSuccessfulResponse_InvalidStatus(t *testing.T) {
	ctx := context.Background()

	order := &entity.Order{
		Number: "123",
		Status: entity.StatusProcessing,
	}

	updated := &entity.Order{
		Number: "123",
		Status: entity.StatusInvalid,
	}

	repo := repoMocks.NewRepository(t)
	orderRepo := repoMocks.NewOrderRepository(t)

	// handleSuccessfulResponse DOES call repository.Order()
	repo.
		On("Order").
		Return(orderRepo).
		Once()

	orderRepo.
		On(
			"UpdateOrderStatus",
			ctx,
			order.Number,
			entity.StatusInvalid,
		).
		Return(updated, nil).
		Once()

	resp := &http.Response{
		Body: io.NopCloser(bytes.NewBufferString(`{"status":"INVALID"}`)),
	}

	worker := &OrderProcessor{
		repository: repo,
		in:         make(chan entity.OrderProcessTask, 1),
	}

	worker.handleSuccessfulResponse(ctx, order, resp)

	repo.AssertExpectations(t)
	orderRepo.AssertExpectations(t)
}

func TestProcessOrder_OrderNotFound(t *testing.T) {
	ctx := context.Background()

	repo := repoMocks.NewRepository(t)
	orderRepo := repoMocks.NewOrderRepository(t)
	repo.On("Order").Return(orderRepo)

	orderRepo.
		On("GetByNumber", ctx, "123").
		Return(nil, nil).
		Once()

	worker := &OrderProcessor{
		repository: repo,
	}

	worker.processOrder(ctx, entity.OrderProcessTask{Number: "123"})
	orderRepo.AssertExpectations(t)
}

func TestProcessOrder_AlreadyProcessed(t *testing.T) {
	ctx := context.Background()

	order := &entity.Order{
		Number: "123",
		Status: entity.StatusProcessed,
	}

	repo := repoMocks.NewRepository(t)
	orderRepo := repoMocks.NewOrderRepository(t)
	repo.On("Order").Return(orderRepo)

	orderRepo.
		On("GetByNumber", ctx, "123").
		Return(order, nil).
		Once()

	worker := &OrderProcessor{
		repository: repo,
		in:         make(chan entity.OrderProcessTask, 1),
	}

	worker.processOrder(ctx, entity.OrderProcessTask{Number: "123"})
	orderRepo.AssertExpectations(t)
}

func TestProcessOrder_InvalidStatusSkipped(t *testing.T) {
	ctx := context.Background()

	order := &entity.Order{
		Number: "123",
		Status: entity.StatusInvalid,
	}

	repo := repoMocks.NewRepository(t)
	orderRepo := repoMocks.NewOrderRepository(t)

	repo.On("Order").Return(orderRepo)
	orderRepo.On("GetByNumber", ctx, "123").Return(order, nil)

	worker := &OrderProcessor{repository: repo, in: make(chan entity.OrderProcessTask, 1)}

	worker.processOrder(ctx, entity.OrderProcessTask{Number: "123"})
}

func TestProcessOrder_Registered(t *testing.T) {
	ctx := context.Background()

	order := &entity.Order{
		Number: "123",
		Status: entity.StatusNew,
	}

	repo := repoMocks.NewRepository(t)
	orderRepo := repoMocks.NewOrderRepository(t)

	repo.On("Order").Return(orderRepo)
	orderRepo.On("GetByNumber", ctx, "123").Return(order, nil)
	orderRepo.On("UpdateOrderStatus", ctx, "123", entity.StatusProcessing).
		Return(order, nil)

	newClient := newTestClient(
		http.StatusOK,
		`{"status":"REGISTERED"}`,
		http.Header{},
	)

	worker := &OrderProcessor{
		repository: repo,
		client:     newClient,
		in:         make(chan entity.OrderProcessTask, 1),
	}

	worker.processOrder(ctx, entity.OrderProcessTask{Number: "123"})
}

func TestProcessOrder_TooManyRequests(t *testing.T) {
	ctx := context.Background()

	order := &entity.Order{
		Number: "123",
		Status: entity.StatusNew,
	}

	repo := repoMocks.NewRepository(t)
	orderRepo := repoMocks.NewOrderRepository(t)

	repo.On("Order").Return(orderRepo)
	orderRepo.On("GetByNumber", ctx, "123").Return(order, nil)
	orderRepo.On("UpdateOrderStatus", ctx, "123", entity.StatusProcessing).
		Return(order, nil)

	headers := http.Header{}
	headers.Set("Retry-After", "120")

	newClient := newTestClient(
		http.StatusTooManyRequests,
		"",
		headers,
	)

	worker := &OrderProcessor{
		repository: repo,
		client:     newClient,
		in:         make(chan entity.OrderProcessTask, 1),
	}

	worker.processOrder(ctx, entity.OrderProcessTask{Number: "123"})
}

func TestProcessOrder_HTTPError(t *testing.T) {
	ctx := context.Background()

	order := &entity.Order{
		Number: "123",
		Status: entity.StatusNew,
	}

	repo := repoMocks.NewRepository(t)
	orderRepo := repoMocks.NewOrderRepository(t)

	repo.On("Order").Return(orderRepo)
	orderRepo.On("GetByNumber", ctx, "123").Return(order, nil)
	orderRepo.On("UpdateOrderStatus", ctx, "123", entity.StatusProcessing).
		Return(order, nil)

	newClient := newTestClientWithError(fmt.Errorf("error"))

	worker := &OrderProcessor{
		repository: repo,
		client:     newClient,
		in:         make(chan entity.OrderProcessTask, 1),
	}

	worker.sleepUntil.Store(time.Time{})

	worker.processOrder(ctx, entity.OrderProcessTask{Number: "123"})
}

func newTestClient(
	status int,
	body string,
	headers http.Header,
) *client.Client {
	if headers == nil {
		headers = http.Header{}
	}

	// Fake transport to intercept HTTP calls
	transport := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		return nil, errors.New("server error")
	})

	httpClient := &http.Client{
		Transport: transport,
		Timeout:   time.Second,
	}

	retryClient := client.NewRetryClient(
		httpClient,
		0, // no retries in tests
		time.Millisecond,
	)

	return &client.Client{
		BaseURL: "http://example.com",
		Client:  retryClient,
	}
}

func newTestClientWithError(err error) *client.Client {
	transport := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		return nil, err
	})

	httpClient := &http.Client{Transport: transport}

	retryClient := client.NewRetryClient(httpClient, 0, time.Millisecond)

	return &client.Client{
		BaseURL: "http://example.com",
		Client:  retryClient,
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}
