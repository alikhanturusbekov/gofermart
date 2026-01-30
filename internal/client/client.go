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
	Client  *http.Client
}

func NewClient(baseURL string, timeout time.Duration) *Client {
	return &Client{
		BaseURL: baseURL,
		Client: &http.Client{
			Timeout: timeout,
		},
	}
}
