package epo_ops

import (
	"context"
	"errors"
	"io"
	"math/rand"
	"net"
	"net/http"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// maxBackoff is the absolute cap on the wait between retries, regardless of
// the configured base delay or any Retry-After value.
const maxBackoff = 60 * time.Second

// retryableRequest executes a function with retry logic and exponential backoff.
//
// A 401 response triggers exactly one token clear + refresh + retry (without a
// backoff delay), which composes with the backoff retries used for transient
// 5xx responses: a stale token encountered after one or more 5xx retries is
// still refreshed once.
func (c *Client) retryableRequest(ctx context.Context, fn func() (*http.Response, error)) (*http.Response, error) {
	var lastErr error
	var resp *http.Response
	var retriedAfter401 bool

	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		// Execute request
		resp, lastErr = fn()

		if lastErr == nil {
			// Handle 401 once: clear the cached token and retry immediately so
			// authTransport fetches a fresh one. This does not consume a backoff
			// attempt and only happens a single time per call.
			if resp.StatusCode == http.StatusUnauthorized && !retriedAfter401 {
				retriedAfter401 = true
				if resp.Body != nil {
					_ = resp.Body.Close() // Ignore close error, we're retrying
				}
				c.authenticator.ClearToken()
				attempt--
				continue
			}

			// If status is OK or non-retryable, return immediately
			if !isRetryableStatusCode(resp.StatusCode) {
				return resp, nil
			}

			// Honor Retry-After before closing the body.
			retryAfter := parseRetryAfter(resp.Header.Get("Retry-After"))

			// Close the body if we're going to retry
			if resp.Body != nil {
				_ = resp.Body.Close() // Ignore close error since we're retrying anyway
			}
			lastErr = &ServiceUnavailableError{
				StatusCode: resp.StatusCode,
				Message:    "retryable status code",
				RetryAfter: resp.Header.Get("Retry-After"),
			}

			if attempt < c.config.MaxRetries {
				if err := c.waitBeforeRetry(ctx, attempt, retryAfter); err != nil {
					return nil, err
				}
			}
			continue
		}

		// Check if error is retryable
		if !isRetryableError(lastErr) {
			return resp, lastErr
		}

		// Don't sleep after the last attempt
		if attempt < c.config.MaxRetries {
			if err := c.waitBeforeRetry(ctx, attempt, 0); err != nil {
				return nil, err
			}
		}
	}

	return resp, lastErr
}

// waitBeforeRetry sleeps before the next retry. When retryAfter is positive it
// is honored (capped); otherwise an exponential backoff with jitter is used.
func (c *Client) waitBeforeRetry(ctx context.Context, attempt int, retryAfter time.Duration) error {
	backoff := backoffDuration(c.config.RetryDelay, attempt, retryAfter)
	select {
	case <-time.After(backoff):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// backoffDuration computes the wait before a retry. When retryAfter is positive
// it is used (capped at maxBackoff). Otherwise it falls back to exponential
// backoff (base * 2^attempt) with +/-20% jitter, capped at maxBackoff.
func backoffDuration(base time.Duration, attempt int, retryAfter time.Duration) time.Duration {
	if retryAfter > 0 {
		if retryAfter > maxBackoff {
			return maxBackoff
		}
		return retryAfter
	}

	// Cap shift amount to prevent overflow (max 2^10 = 1024x multiplier).
	shift := attempt
	if shift < 0 {
		shift = 0
	} else if shift > 10 {
		shift = 10
	}
	backoff := base * time.Duration(1<<shift)
	if backoff > maxBackoff {
		backoff = maxBackoff
	}

	// Apply +/-20% jitter to avoid synchronized retries (thundering herd).
	jitter := 1.0 + (rand.Float64()*0.4 - 0.2)
	backoff = time.Duration(float64(backoff) * jitter)
	if backoff > maxBackoff {
		backoff = maxBackoff
	}
	return backoff
}

// parseRetryAfter parses a Retry-After header value, which may be either a
// delay in seconds or an HTTP date. Returns 0 when absent or unparseable.
func parseRetryAfter(value string) time.Duration {
	if value == "" {
		return 0
	}
	if seconds, err := strconv.Atoi(strings.TrimSpace(value)); err == nil {
		if seconds <= 0 {
			return 0
		}
		return time.Duration(seconds) * time.Second
	}
	if t, err := http.ParseTime(value); err == nil {
		if d := time.Until(t); d > 0 {
			return d
		}
	}
	return 0
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
