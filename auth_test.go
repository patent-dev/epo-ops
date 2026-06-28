package epo_ops

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
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

// fakeStore is a concurrency-safe in-memory TokenStore for tests.
type fakeStore struct {
	mu      sync.Mutex
	token   string
	expiry  time.Time
	has     bool
	saves   int
	deletes int
}

func (s *fakeStore) Load() (string, time.Time, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.token, s.expiry, s.has
}

func (s *fakeStore) Save(token string, expiry time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.token, s.expiry, s.has = token, expiry, true
	s.saves++
}

func (s *fakeStore) Delete() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.token, s.has = "", false
	s.deletes++
}

func TestGetToken_UsesStoredTokenWithoutHittingEndpoint(t *testing.T) {
	calls := 0
	server := newTokenServer(`{"access_token":"fresh","expires_in":"3600"}`, &calls)
	defer server.Close()

	auth := NewAuthenticator("key", "secret", nil)
	auth.authURL = server.URL
	auth.store = &fakeStore{token: "cached", expiry: time.Now().Add(time.Hour), has: true}

	tok, err := auth.GetToken(context.Background())
	if err != nil {
		t.Fatalf("GetToken: %v", err)
	}
	if tok != "cached" {
		t.Errorf("token = %q, want stored %q", tok, "cached")
	}
	if calls != 0 {
		t.Errorf("token endpoint called %d times, want 0 (stored token is valid)", calls)
	}
}

func TestGetToken_FetchesAndSavesWhenStoreEmpty(t *testing.T) {
	calls := 0
	server := newTokenServer(`{"access_token":"fresh","expires_in":"3600"}`, &calls)
	defer server.Close()

	store := &fakeStore{}
	auth := NewAuthenticator("key", "secret", nil)
	auth.authURL = server.URL
	auth.store = store

	tok, err := auth.GetToken(context.Background())
	if err != nil {
		t.Fatalf("GetToken: %v", err)
	}
	if tok != "fresh" {
		t.Errorf("token = %q, want %q", tok, "fresh")
	}
	if calls != 1 {
		t.Errorf("token endpoint called %d times, want 1", calls)
	}
	if store.saves != 1 || store.token != "fresh" {
		t.Errorf("store not updated: saves=%d token=%q", store.saves, store.token)
	}
}

func TestGetToken_IgnoresExpiredStoredToken(t *testing.T) {
	calls := 0
	server := newTokenServer(`{"access_token":"fresh","expires_in":"3600"}`, &calls)
	defer server.Close()

	store := &fakeStore{token: "stale", expiry: time.Now().Add(-time.Hour), has: true}
	auth := NewAuthenticator("key", "secret", nil)
	auth.authURL = server.URL
	auth.store = store

	tok, err := auth.GetToken(context.Background())
	if err != nil {
		t.Fatalf("GetToken: %v", err)
	}
	if tok != "fresh" {
		t.Errorf("token = %q, want fresh (expired stored token must be ignored)", tok)
	}
	if calls != 1 || store.saves != 1 {
		t.Errorf("calls=%d saves=%d, want 1/1", calls, store.saves)
	}
}

func TestClearToken_InvalidatesStore(t *testing.T) {
	calls := 0
	server := newTokenServer(`{"access_token":"fresh","expires_in":"3600"}`, &calls)
	defer server.Close()

	store := &fakeStore{token: "stale", expiry: time.Now().Add(time.Hour), has: true}
	auth := NewAuthenticator("key", "secret", nil)
	auth.authURL = server.URL
	auth.store = store

	// On a rejected token the client calls ClearToken; it must invalidate the store
	// too, so the next fetch cannot re-serve the revoked-but-unexpired token.
	auth.ClearToken()
	if store.deletes != 1 {
		t.Fatalf("store deletes = %d, want 1 (ClearToken must invalidate the store)", store.deletes)
	}
	tok, err := auth.GetToken(context.Background())
	if err != nil {
		t.Fatalf("GetToken: %v", err)
	}
	if tok != "fresh" {
		t.Errorf("token = %q, want fresh (the stale store token must not be re-served)", tok)
	}
	if calls != 1 {
		t.Errorf("token endpoint calls = %d, want 1 (a fresh token was fetched)", calls)
	}
}

func TestGetToken_ConcurrentWithStore(t *testing.T) {
	calls := 0
	server := newTokenServer(`{"access_token":"t","expires_in":"3600"}`, &calls)
	defer server.Close()

	auth := NewAuthenticator("key", "secret", nil)
	auth.authURL = server.URL
	auth.store = &fakeStore{}

	var wg sync.WaitGroup
	for range 20 {
		wg.Add(1)
		go func() { defer wg.Done(); _, _ = auth.GetToken(context.Background()) }()
	}
	wg.Wait()
	if calls != 1 {
		t.Errorf("token endpoint calls = %d, want 1 (in-process fetch must be single-flighted)", calls)
	}
}

func TestGetToken_NilStoreFetchesNormally(t *testing.T) {
	calls := 0
	server := newTokenServer(`{"access_token":"fresh","expires_in":"3600"}`, &calls)
	defer server.Close()

	auth := NewAuthenticator("key", "secret", nil) // no store wired
	auth.authURL = server.URL

	tok, err := auth.GetToken(context.Background())
	if err != nil {
		t.Fatalf("GetToken: %v", err)
	}
	if tok != "fresh" || calls != 1 {
		t.Errorf("token=%q calls=%d, want fresh/1", tok, calls)
	}
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
