package worker

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/alikhanturusbekov/gofermart/internal/client"
	"github.com/alikhanturusbekov/gofermart/internal/entity"
	"github.com/alikhanturusbekov/gofermart/internal/repository"
)

const (
	DefaultBufferSize = 500
	DefaultWorkers    = 5
	defaultRetryAfter = 60 * time.Second
)

type OrderProcessor struct {
	repository repository.Repository
	client     *client.Client
	in         chan entity.OrderProcessTask
	workers    int

	sleepUntil atomic.Value // value to stop workers altogether
	wg         sync.WaitGroup
}

// NewOrderProcessWorker creates a new worker pool
func NewOrderProcessWorker(
	repository repository.Repository,
	client *client.Client,
	bufferSize int,
	workers int,
) *OrderProcessor {
	return &OrderProcessor{
		repository: repository,
		client:     client,
		in:         make(chan entity.OrderProcessTask, bufferSize),
		workers:    workers,
	}
}

// Enqueue adds a task to the worker queue
func (w *OrderProcessor) Enqueue(task entity.OrderProcessTask) {
	w.in <- task
}

// Run starts all workers
func (w *OrderProcessor) Run(ctx context.Context) {
	for i := 0; i < w.workers; i++ {
		w.wg.Add(1)
		go w.worker(ctx)
	}
}

// Shutdown waits for all workers to finish
func (w *OrderProcessor) Shutdown() {
	w.wg.Wait()
}

// worker is a single goroutine processing orders
func (w *OrderProcessor) worker(ctx context.Context) {
	defer w.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case task := <-w.in:
			w.waitIfRateLimited(ctx)
			w.processOrder(ctx, task)
		}
	}
}

// waitIfRateLimited sleeps until time in retry-after or context is done
func (w *OrderProcessor) waitIfRateLimited(ctx context.Context) {
	v := w.sleepUntil.Load()
	if v == nil {
		return
	}

	until := v.(time.Time)
	d := time.Until(until)
	if d <= 0 {
		return
	}

	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return
	case <-timer.C:
		return
	}
}

// setRateLimit sets rate limit until the specific time
func (w *OrderProcessor) setRateLimit(delay time.Duration) {
	until := time.Now().Add(delay)
	w.sleepUntil.Store(until)
}

// processOrder handles a single order task
func (w *OrderProcessor) processOrder(ctx context.Context, task entity.OrderProcessTask) {
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

	resp, err := w.client.Client.Do(ctx, req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		w.handleSuccessfulResponse(ctx, order, resp)

	case http.StatusNoContent:
		_, _ = w.repository.Order().UpdateOrderStatus(ctx, order.Number, entity.StatusNew)

	case http.StatusTooManyRequests:
		delay := parseRetryAfter(resp.Header.Get("Retry-After"))
		w.setRateLimit(delay)
		w.Enqueue(task)

	default:
		return
	}
}

// handleSuccessfulResponse parses successful response and updates order
func (w *OrderProcessor) handleSuccessfulResponse(ctx context.Context, order *entity.Order, resp *http.Response) {
	var result struct {
		Status  string   `json:"status"`
		Accrual *float64 `json:"accrual,omitempty"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return
	}

	switch entity.OrderStatus(result.Status) {
	case entity.StatusRegistered, entity.StatusProcessing:
		return
	case entity.StatusInvalid:
		_, _ = w.repository.Order().UpdateOrderStatus(ctx, order.Number, entity.StatusInvalid)
	case entity.StatusProcessed:
		w.processAccrual(ctx, order, result.Accrual)
	}
}

// processAccrual marks order processed and adds points to user
func (w *OrderProcessor) processAccrual(ctx context.Context, order *entity.Order, accrual *float64) {
	tx, err := w.repository.BeginTx(ctx)
	if err != nil {
		return
	}
	defer tx.Rollback()

	if _, err := w.repository.Order().MarkOrderProcessedTx(ctx, tx, order.Number, accrual); err != nil {
		return
	}

	if err := w.repository.User().AddUserBalanceTx(ctx, tx, order.UserID, accrual); err != nil {
		return
	}

	if err := tx.Commit(); err != nil {
		_, _ = w.repository.Order().UpdateOrderStatus(ctx, order.Number, entity.StatusInvalid)
	}
}

// parseRetryAfter parses retry-after header into duration
func parseRetryAfter(value string) time.Duration {
	if value == "" {
		return defaultRetryAfter
	}

	if seconds, err := strconv.Atoi(value); err == nil {
		return time.Duration(seconds) * time.Second
	}

	return defaultRetryAfter
}
