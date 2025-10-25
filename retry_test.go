package epo_ops

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"syscall"
	"testing"
	"time"
)

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "AuthError not retryable",
			err:      &AuthError{Message: "auth failed"},
			expected: false,
		},
		{
			name:     "NotFoundError not retryable",
			err:      &NotFoundError{Message: "not found"},
			expected: false,
		},
		{
			name:     "QuotaExceededError not retryable",
			err:      &QuotaExceededError{Message: "quota exceeded"},
			expected: false,
		},
		{
			name:     "ServiceUnavailableError retryable",
			err:      &ServiceUnavailableError{StatusCode: 503, Message: "service unavailable"},
			expected: true,
		},
		{
			name:     "EOF retryable",
			err:      io.EOF,
			expected: true,
		},
		{
			name:     "UnexpectedEOF retryable",
			err:      io.ErrUnexpectedEOF,
			expected: true,
		},
		{
			name:     "net.OpError retryable",
			err:      &net.OpError{Op: "dial", Err: syscall.ECONNREFUSED},
			expected: true,
		},
		{
			name:     "unknown error not retryable",
			err:      errors.New("unknown error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isRetryableError(tt.err)
			if result != tt.expected {
				t.Errorf("isRetryableError(%v) = %v, expected %v", tt.err, result, tt.expected)
			}
		})
	}
}

func TestIsRetryableStatusCode(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		expected   bool
	}{
		{
			name:       "200 OK not retryable",
			statusCode: http.StatusOK,
			expected:   false,
		},
		{
			name:       "400 Bad Request not retryable",
			statusCode: http.StatusBadRequest,
			expected:   false,
		},
		{
			name:       "401 Unauthorized not retryable",
			statusCode: http.StatusUnauthorized,
			expected:   false,
		},
		{
			name:       "404 Not Found not retryable",
			statusCode: http.StatusNotFound,
			expected:   false,
		},
		{
			name:       "408 Request Timeout retryable",
			statusCode: http.StatusRequestTimeout,
			expected:   true,
		},
		{
			name:       "429 Too Many Requests not retryable",
			statusCode: http.StatusTooManyRequests,
			expected:   false,
		},
		{
			name:       "500 Internal Server Error retryable",
			statusCode: http.StatusInternalServerError,
			expected:   true,
		},
		{
			name:       "502 Bad Gateway retryable",
			statusCode: http.StatusBadGateway,
			expected:   true,
		},
		{
			name:       "503 Service Unavailable retryable",
			statusCode: http.StatusServiceUnavailable,
			expected:   true,
		},
		{
			name:       "504 Gateway Timeout retryable",
			statusCode: http.StatusGatewayTimeout,
			expected:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isRetryableStatusCode(tt.statusCode)
			if result != tt.expected {
				t.Errorf("isRetryableStatusCode(%d) = %v, expected %v", tt.statusCode, result, tt.expected)
			}
		})
	}
}

func TestRetryableRequest(t *testing.T) {
	t.Run("Success on first attempt", func(t *testing.T) {
		config := DefaultConfig()
		config.ConsumerKey = "test"
		config.ConsumerSecret = "test"

		client, err := NewClient(config)
		if err != nil {
			t.Fatalf("Failed to create client: %v", err)
		}

		attemptCount := 0
		resp, err := client.retryableRequest(context.Background(), func() (*http.Response, error) {
			attemptCount++
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(nil),
			}, nil
		})

		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got: %d", resp.StatusCode)
		}

		if attemptCount != 1 {
			t.Errorf("Expected 1 attempt, got: %d", attemptCount)
		}
	})

	t.Run("Retry on 503 then success", func(t *testing.T) {
		config := DefaultConfig()
		config.ConsumerKey = "test"
		config.ConsumerSecret = "test"
		config.RetryDelay = 1 // 1 nanosecond for fast tests

		client, err := NewClient(config)
		if err != nil {
			t.Fatalf("Failed to create client: %v", err)
		}

		attemptCount := 0
		resp, err := client.retryableRequest(context.Background(), func() (*http.Response, error) {
			attemptCount++
			if attemptCount == 1 {
				return &http.Response{
					StatusCode: http.StatusServiceUnavailable,
					Body:       io.NopCloser(nil),
				}, nil
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(nil),
			}, nil
		})

		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got: %d", resp.StatusCode)
		}

		if attemptCount != 2 {
			t.Errorf("Expected 2 attempts, got: %d", attemptCount)
		}
	})

	t.Run("No retry on 404", func(t *testing.T) {
		config := DefaultConfig()
		config.ConsumerKey = "test"
		config.ConsumerSecret = "test"

		client, err := NewClient(config)
		if err != nil {
			t.Fatalf("Failed to create client: %v", err)
		}

		attemptCount := 0
		resp, err := client.retryableRequest(context.Background(), func() (*http.Response, error) {
			attemptCount++
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       io.NopCloser(nil),
			}, nil
		})

		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("Expected status 404, got: %d", resp.StatusCode)
		}

		if attemptCount != 1 {
			t.Errorf("Expected 1 attempt (no retry), got: %d", attemptCount)
		}
	})

	t.Run("Max retries exhausted", func(t *testing.T) {
		config := DefaultConfig()
		config.ConsumerKey = "test"
		config.ConsumerSecret = "test"
		config.MaxRetries = 2
		config.RetryDelay = 1 * time.Nanosecond // 1 nanosecond for fast tests

		client, err := NewClient(config)
		if err != nil {
			t.Fatalf("Failed to create client: %v", err)
		}

		t.Logf("Client config: MaxRetries=%d, RetryDelay=%v", client.config.MaxRetries, client.config.RetryDelay)

		attemptCount := 0
		testErr := &net.OpError{Op: "dial", Err: errors.New("connection refused")}

		// Test isRetryableError directly first
		if !isRetryableError(testErr) {
			t.Fatalf("testErr should be retryable, but isRetryableError returned false")
		}

		_, err = client.retryableRequest(context.Background(), func() (*http.Response, error) {
			attemptCount++
			t.Logf("Attempt %d", attemptCount)
			return nil, testErr
		})

		t.Logf("Final error: %v", err)

		if err == nil {
			t.Error("Expected error after max retries, got nil")
		}

		// Should try initial + MaxRetries times
		expectedAttempts := 3 // 1 initial + 2 retries
		if attemptCount != expectedAttempts {
			t.Errorf("Expected %d attempts, got: %d", expectedAttempts, attemptCount)
		}
	})
}
