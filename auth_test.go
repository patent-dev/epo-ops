package epo_ops

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// newTokenServer returns a mock OAuth2 endpoint that responds with the given
// JSON body and records the number of calls.
func newTokenServer(body string, callCount *int) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if callCount != nil {
			*callCount++
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
}

func TestRequestToken_RejectsMalformedExpiresIn(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{"trailing garbage", `{"access_token":"abc","expires_in":"3600abc"}`},
		{"negative", `{"access_token":"abc","expires_in":"-5"}`},
		{"zero", `{"access_token":"abc","expires_in":"0"}`},
		{"non-numeric", `{"access_token":"abc","expires_in":"soon"}`},
		{"empty", `{"access_token":"abc","expires_in":""}`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server := newTokenServer(tc.body, nil)
			defer server.Close()

			auth := NewAuthenticator("key", "secret", nil)
			auth.authURL = server.URL

			_, err := auth.GetToken(context.Background())
			if err == nil {
				t.Fatalf("expected error for expires_in %q, got nil", tc.body)
			}
			// The token must not be cached when expires_in is invalid.
			if auth.token != "" {
				t.Errorf("expected token to remain empty, got %q", auth.token)
			}
		})
	}
}

func TestRequestToken_AcceptsValidExpiresIn(t *testing.T) {
	server := newTokenServer(`{"access_token":"good_token","expires_in":"3600"}`, nil)
	defer server.Close()

	auth := NewAuthenticator("key", "secret", nil)
	auth.authURL = server.URL

	token, err := auth.GetToken(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "good_token" {
		t.Errorf("expected good_token, got %q", token)
	}
	if !auth.tokenExpiry.After(time.Now()) {
		t.Errorf("expected future expiry, got %v", auth.tokenExpiry)
	}
}
