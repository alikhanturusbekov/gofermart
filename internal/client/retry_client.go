package client

import (
	"context"
	"net/http"
	"time"
)

const (
	defaultMaxRetries = 3
	defaultRetryDelay = 5 * time.Second
)

// RetryClient client with embedded retry mechanism
type RetryClient struct {
	client     *http.Client
	maxRetries int
	delay      time.Duration
}

func NewRetryClient(base *http.Client, maxRetries int, delay time.Duration) *RetryClient {
	return &RetryClient{
		client:     base,
		maxRetries: maxRetries,
		delay:      delay,
	}
}

// Do makes request with retries
func (c *RetryClient) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	var lastErr error

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		req = req.Clone(ctx)

		resp, err := c.client.Do(req)
		if err == nil && !isRetryableResponse(resp) {
			return resp, nil
		}

		lastErr = err

		timer := time.NewTimer(time.Duration(attempt) * c.delay)

		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}

	return nil, lastErr
}

// isRetryableResponse does response indicate that retry can be made
func isRetryableResponse(resp *http.Response) bool {
	if resp == nil {
		return true
	}

	switch resp.StatusCode {
	case http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}
