package epo_ops

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"syscall"
	"time"
)

// retryableRequest executes a function with retry logic and exponential backoff.
func (c *Client) retryableRequest(ctx context.Context, fn func() (*http.Response, error)) (*http.Response, error) {
	var lastErr error
	var resp *http.Response

	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		// Execute request
		resp, lastErr = fn()

		// If no error and status is OK or non-retryable, return immediately
		if lastErr == nil {
			if !isRetryableStatusCode(resp.StatusCode) {
				return resp, nil
			}
			// Close the body if we're going to retry
			if resp.Body != nil {
				_ = resp.Body.Close() // Ignore close error since we're retrying anyway
			}
			lastErr = &ServiceUnavailableError{
				StatusCode: resp.StatusCode,
				Message:    "retryable status code",
			}
		}

		// Check if error is retryable
		if !isRetryableError(lastErr) {
			return resp, lastErr
		}

		// Don't sleep after the last attempt
		if attempt < c.config.MaxRetries {
			// Exponential backoff: 1s, 2s, 4s, 8s, etc.
			// Cap shift amount to prevent overflow (max 2^10 = 1024x multiplier)
			shift := attempt
			if shift < 0 {
				shift = 0
			} else if shift > 10 {
				shift = 10
			}
			backoff := c.config.RetryDelay * time.Duration(1<<shift)

			// Sleep with context cancellation support
			select {
			case <-time.After(backoff):
				// Continue to next retry
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
	}

	return resp, lastErr
}

// isRetryableError determines if an error should trigger a retry.
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Check for specific error types that should not be retried
	var authErr *AuthError
	if errors.As(err, &authErr) {
		// Don't retry auth errors (credentials are wrong)
		return false
	}

	var notFoundErr *NotFoundError
	if errors.As(err, &notFoundErr) {
		// Don't retry 404s
		return false
	}

	var quotaErr *QuotaExceededError
	if errors.As(err, &quotaErr) {
		// Don't retry quota exceeded errors
		return false
	}

	// Retry service unavailable errors
	var serviceErr *ServiceUnavailableError
	if errors.As(err, &serviceErr) {
		return true
	}

	// Check for connection errors (must come before net.Error check since OpError implements net.Error)
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return true
	}

	// Check for network errors (primarily timeouts)
	var netErr net.Error
	if errors.As(err, &netErr) {
		// Retry on timeout errors
		// Note: We don't use Temporary() as it's deprecated and ill-defined
		return netErr.Timeout()
	}

	// Check for EOF errors (connection closed)
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}

	// Check for syscall errors - these might be wrapped
	if errors.Is(err, syscall.ECONNREFUSED) ||
		errors.Is(err, syscall.ECONNRESET) ||
		errors.Is(err, syscall.ETIMEDOUT) ||
		errors.Is(err, syscall.EPIPE) {
		return true
	}

	// Check error message for connection errors (as fallback)
	errMsg := err.Error()
	if strings.Contains(errMsg, "connection refused") ||
		strings.Contains(errMsg, "connection reset") ||
		strings.Contains(errMsg, "broken pipe") {
		return true
	}

	// Default: don't retry unknown errors
	return false
}

// isRetryableStatusCode determines if an HTTP status code should trigger a retry.
func isRetryableStatusCode(statusCode int) bool {
	switch statusCode {
	case http.StatusRequestTimeout, // 408
		http.StatusInternalServerError, // 500
		http.StatusBadGateway,          // 502
		http.StatusServiceUnavailable,  // 503
		http.StatusGatewayTimeout:      // 504
		return true
	default:
		return false
	}
}
