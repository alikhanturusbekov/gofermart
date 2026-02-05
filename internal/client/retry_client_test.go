package client

import (
	"context"
	"github.com/pkg/errors"
	"io"
	"net/http"
	"testing"
	"time"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestIsRetryableResponse(t *testing.T) {
	tests := []struct {
		name string
		resp *http.Response
		want bool
	}{
		{"nil response", nil, true},
		{"500", &http.Response{StatusCode: http.StatusInternalServerError}, true},
		{"502", &http.Response{StatusCode: http.StatusBadGateway}, true},
		{"503", &http.Response{StatusCode: http.StatusServiceUnavailable}, true},
		{"504", &http.Response{StatusCode: http.StatusGatewayTimeout}, true},
		{"200", &http.Response{StatusCode: http.StatusOK}, false},
		{"400", &http.Response{StatusCode: http.StatusBadRequest}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isRetryableResponse(tt.resp); got != tt.want {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRetryClient_Do_NoRetryOn200(t *testing.T) {
	httpClient := &http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(nil),
			}, nil
		}),
	}

	c := NewRetryClient(httpClient, 3, 0)

	req, _ := http.NewRequest("GET", "http://example.com", nil)

	resp, err := c.Do(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
}

func TestRetryClient_Do_RetryExhausted(t *testing.T) {
	calls := 0

	httpClient := &http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			calls++
			return nil, errors.New("network error")
		}),
	}

	c := NewRetryClient(httpClient, 1, 0)

	req, _ := http.NewRequest("GET", "http://example.com", nil)

	resp, err := c.Do(context.Background(), req)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if resp != nil {
		t.Fatal("expected nil response")
	}
	if calls != 2 { // attempt 0 + attempt 1
		t.Fatalf("calls = %d, want 2", calls)
	}
}

func TestRetryClient_Do_ContextCancelled(t *testing.T) {
	httpClient := &http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			return nil, errors.New("network error")
		}),
	}

	c := NewRetryClient(httpClient, 5, time.Hour) // delay won't matter

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	req, _ := http.NewRequest("GET", "http://example.com", nil)

	resp, err := c.Do(ctx, req)
	if err == nil {
		t.Fatal("expected context error")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v", err)
	}
	if resp != nil {
		t.Fatal("expected nil response")
	}
}
