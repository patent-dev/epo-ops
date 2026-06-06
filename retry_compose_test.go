package epo_ops

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestRetry_401AfterRetryable verifies that a 401 encountered after one or more
// retryable 5xx responses still triggers a token clear + refresh + retry. The
// 401 guard must not be consumed by the earlier backoff retries.
func TestRetry_401AfterRetryable(t *testing.T) {
	authCalls := 0
	authServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		authCalls++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"test_token_12345","expires_in":"3600"}`))
	}))
	defer authServer.Close()

	// Sequence: 503 (retryable) -> 401 (refresh) -> 200.
	requestCount := 0
	opsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requestCount++
		switch requestCount {
		case 1:
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`busy`))
		case 2:
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`expired`))
		default:
			w.Header().Set("Content-Type", "application/xml")
			_, _ = w.Write(loadTestData("biblio.xml"))
		}
	}))
	defer opsServer.Close()

	config := &Config{
		ConsumerKey:    "test",
		ConsumerSecret: "test",
		BaseURL:        opsServer.URL,
		RetryDelay:     time.Nanosecond,
	}
	config.AuthURL = authServer.URL + "/auth/accesstoken"

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	if _, err := client.GetBiblio(context.Background(), "publication", "docdb", "EP.1000000.B1"); err != nil {
		t.Fatalf("expected success after 503->401->200, got: %v", err)
	}

	if requestCount != 3 {
		t.Errorf("expected 3 API requests (503, 401, 200), got %d", requestCount)
	}
	// Initial token + one refresh triggered by the 401.
	if authCalls != 2 {
		t.Errorf("expected 2 auth calls (initial + refresh), got %d", authCalls)
	}
}

// TestRetry_HonorsRetryAfterHeader verifies that retryableRequest waits at least
// the Retry-After duration before retrying a retryable response.
func TestRetry_HonorsRetryAfterHeader(t *testing.T) {
	config := DefaultConfig()
	config.ConsumerKey = "test"
	config.ConsumerSecret = "test"
	config.RetryDelay = time.Nanosecond // exponential fallback would be ~instant

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	attempt := 0
	start := time.Now()
	resp, err := client.retryableRequest(context.Background(), func() (*http.Response, error) {
		attempt++
		if attempt == 1 {
			h := http.Header{}
			h.Set("Retry-After", "1")
			return &http.Response{
				StatusCode: http.StatusServiceUnavailable,
				Header:     h,
				Body:       io.NopCloser(strings.NewReader("busy")),
			}, nil
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("ok")),
		}, nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if elapsed := time.Since(start); elapsed < time.Second {
		t.Errorf("expected to wait at least 1s for Retry-After, waited %v", elapsed)
	}
}

func TestBackoffDuration(t *testing.T) {
	t.Run("retry-after wins and is capped", func(t *testing.T) {
		got := backoffDuration(time.Second, 0, 5*time.Second)
		if got != 5*time.Second {
			t.Errorf("expected 5s, got %v", got)
		}
		if got := backoffDuration(time.Second, 0, 10*time.Minute); got != maxBackoff {
			t.Errorf("expected cap %v, got %v", maxBackoff, got)
		}
	})

	t.Run("exponential is capped at maxBackoff", func(t *testing.T) {
		// A large base and attempt would overflow far past the cap without one.
		if got := backoffDuration(time.Minute, 10, 0); got > maxBackoff {
			t.Errorf("expected <= %v, got %v", maxBackoff, got)
		}
	})

	t.Run("jitter stays within +/-20%", func(t *testing.T) {
		base := 10 * time.Second
		low := time.Duration(float64(base) * 0.8)
		high := time.Duration(float64(base) * 1.2)
		for i := 0; i < 100; i++ {
			got := backoffDuration(base, 0, 0)
			if got < low || got > high {
				t.Fatalf("backoff %v outside jitter window [%v, %v]", got, low, high)
			}
		}
	})
}

func TestParseRetryAfter(t *testing.T) {
	if got := parseRetryAfter(""); got != 0 {
		t.Errorf("empty: expected 0, got %v", got)
	}
	if got := parseRetryAfter("5"); got != 5*time.Second {
		t.Errorf("seconds: expected 5s, got %v", got)
	}
	if got := parseRetryAfter("-3"); got != 0 {
		t.Errorf("negative seconds: expected 0, got %v", got)
	}
	if got := parseRetryAfter("garbage"); got != 0 {
		t.Errorf("garbage: expected 0, got %v", got)
	}
	future := time.Now().Add(2 * time.Second).UTC().Format(http.TimeFormat)
	if got := parseRetryAfter(future); got <= 0 {
		t.Errorf("http date: expected positive, got %v", got)
	}
}
