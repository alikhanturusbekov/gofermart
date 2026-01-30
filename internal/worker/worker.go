package worker

import (
	"context"
	"encoding/json"
	"github.com/alikhanturusbekov/gofermart/internal/client"
	"github.com/alikhanturusbekov/gofermart/internal/entity"
	"github.com/alikhanturusbekov/gofermart/internal/repository"
	"net/http"
)

const (
	DefaultBufferSize = 500
)

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

		if result.Status == string(entity.StatusRegistered) {
			_, _ = w.repository.Order().UpdateOrderStatus(ctx, order.Number, entity.StatusInvalid)
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
				return
			}
		}

		if result.Status == string(entity.StatusInvalid) {
			_, _ = w.repository.Order().UpdateOrderStatus(ctx, order.Number, entity.StatusInvalid)
		}

	case http.StatusNoContent:
		_, _ = w.repository.Order().UpdateOrderStatus(ctx, order.Number, entity.StatusNew)

	case http.StatusTooManyRequests:
		retryAfter := resp.Header.Get("Retry-After")
		_ = retryAfter

	default:
	}
}
