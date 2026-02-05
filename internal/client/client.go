package client

import (
	"net/http"
	"time"
)

const (
	DefaultTimeout = 5 * time.Second
)

type Client struct {
	BaseURL string
	Client  *RetryClient
}

func NewClient(baseURL string, timeout time.Duration) *Client {
	baseClient := &http.Client{
		Timeout: timeout,
	}

	retryClient := NewRetryClient(baseClient, defaultMaxRetries, defaultRetryDelay)

	return &Client{
		BaseURL: baseURL,
		Client:  retryClient,
	}
}
