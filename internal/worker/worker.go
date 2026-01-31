package worker

import (
	"context"
	"encoding/json"
	"github.com/alikhanturusbekov/gofermart/internal/client"
	"github.com/alikhanturusbekov/gofermart/internal/entity"
	"github.com/alikhanturusbekov/gofermart/internal/repository"
	"net/http"
	"strconv"
	"time"
)

const (
	DefaultBufferSize = 500
)

type OrderProcessor interface {
	Enqueue(task entity.OrderProcessTask)
}

type OrderProcessWorker struct {
	repository repository.Repository
	client     *client.Client
	in         chan entity.OrderProcessTask
}

func NewOrderProcessWorker(
	repository repository.Repository,
	client *client.Client,
	bufferSize int,
) *OrderProcessWorker {
	return &OrderProcessWorker{
		repository: repository,
		client:     client,
		in:         make(chan entity.OrderProcessTask, bufferSize),
	}
}

func (w *OrderProcessWorker) Enqueue(task entity.OrderProcessTask) {
	w.in <- task
}

func (w *OrderProcessWorker) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return

		case task := <-w.in:
			w.processOrder(ctx, task)
		}
	}
}

func (w *OrderProcessWorker) processOrder(ctx context.Context, task entity.OrderProcessTask) {
	order, err := w.repository.Order().GetByNumber(ctx, task.Number)
	if err != nil || order == nil {
		return
	}

	if order.Status == entity.StatusProcessed || order.Status == entity.StatusInvalid {
		return
	}

	_, _ = w.repository.Order().UpdateOrderStatus(ctx, order.Number, entity.StatusProcessing)

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		w.client.BaseURL+"/api/orders/"+order.Number,
		nil,
	)
	if err != nil {
		return
	}

	resp, err := w.client.Client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	switch resp.StatusCode {

	case http.StatusOK:
		var result struct {
			Status  string   `json:"status"`
			Accrual *float64 `json:"accrual,omitempty"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&result)

		if result.Status == string(entity.StatusRegistered) {
			return
		}

		if result.Status == string(entity.StatusInvalid) {
			_, err = w.repository.Order().UpdateOrderStatus(ctx, order.Number, entity.StatusInvalid)
			if err != nil {
				return
			}
		}

		if result.Status == string(entity.StatusProcessed) {
			tx, err := w.repository.BeginTx(ctx)
			if err != nil {
				return
			}
			defer tx.Rollback()

			_, err = w.repository.Order().MarkOrderProcessedTx(ctx, tx, order.Number, result.Accrual)
			if err != nil {
				tx.Rollback()
				return
			}

			err = w.repository.User().AddUserBalanceTx(ctx, tx, order.UserID, result.Accrual)
			if err != nil {
				tx.Rollback()
				return
			}

			err = tx.Commit()
			if err != nil {
				tx.Rollback()
				_, _ = w.repository.Order().UpdateOrderStatus(ctx, order.Number, entity.StatusInvalid)
				return
			}
		}

	case http.StatusNoContent:
		_, err = w.repository.Order().UpdateOrderStatus(ctx, order.Number, entity.StatusNew)
		if err != nil {
			return
		}

	case http.StatusTooManyRequests:
		delay := parseRetryAfter(resp.Header.Get("Retry-After"))
		w.retryLater(task, delay)

	default:
	}
}

// parseRetryAfter gets time after which order look up must be retried
func parseRetryAfter(value string) time.Duration {
	if value == "" {
		return time.Second
	}

	if seconds, err := strconv.Atoi(value); err == nil {
		return time.Duration(seconds) * time.Second
	}

	if t, err := http.ParseTime(value); err == nil {
		d := time.Until(t)
		if d > 0 {
			return d
		}
	}

	return time.Second
}

// retryLater sends order process task after waiting time duration
func (w *OrderProcessWorker) retryLater(task entity.OrderProcessTask, delay time.Duration) {
	go func() {
		time.Sleep(delay)
		w.Enqueue(task)
	}()
}
