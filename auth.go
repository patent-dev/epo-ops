package epo_ops

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	// defaultAuthURL is the default EPO OPS OAuth2 token endpoint
	defaultAuthURL = "https://ops.epo.org/3.2/auth/accesstoken"

	// tokenRefreshBuffer is the time before expiry when we should refresh the token
	tokenRefreshBuffer = 5 * time.Minute
)

// TokenStore persists an OAuth2 access token and its expiry across Authenticator
// instances (e.g. across separate CLI invocations or stateless server requests) so
// the token endpoint is not hit on every new client. Implementations must be safe
// for concurrent use. Nil = in-memory only.
type TokenStore interface {
	// Load returns a cached token and its expiry; ok is false when absent.
	Load() (token string, expiry time.Time, ok bool)
	// Save records a freshly obtained token and its expiry.
	Save(token string, expiry time.Time)
}

// Authenticator handles OAuth2 authentication for the EPO OPS API.
type Authenticator struct {
	authURL        string
	consumerKey    string
	consumerSecret string
	token          string
	tokenExpiry    time.Time
	httpClient     *http.Client
	store          TokenStore
	mu             sync.RWMutex
}

// tokenResponse represents the JSON response from the OAuth2 token endpoint.
type tokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   string `json:"expires_in"` // Seconds until token expires (returned as string by EPO API)
	TokenType   string `json:"token_type"` // Should be "Bearer"
}

// NewAuthenticator creates a new Authenticator.
func NewAuthenticator(consumerKey, consumerSecret string, httpClient *http.Client) *Authenticator {
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: 30 * time.Second,
		}
	}

	return &Authenticator{
		authURL:        defaultAuthURL,
		consumerKey:    consumerKey,
		consumerSecret: consumerSecret,
		httpClient:     httpClient,
	}
}

// GetToken returns a valid access token, refreshing it if necessary.
func (a *Authenticator) GetToken(ctx context.Context) (string, error) {
	// Check if we have a valid cached token
	a.mu.RLock()
	if a.token != "" && time.Now().Add(tokenRefreshBuffer).Before(a.tokenExpiry) {
		token := a.token
		a.mu.RUnlock()
		return token, nil
	}
	a.mu.RUnlock()

	// Need to acquire or refresh token
	a.mu.Lock()
	defer a.mu.Unlock()

	// Double-check after acquiring write lock (another goroutine might have refreshed)
	if a.token != "" && time.Now().Add(tokenRefreshBuffer).Before(a.tokenExpiry) {
		return a.token, nil
	}

	// Try a persisted token (warm across processes/requests) before hitting the
	// token endpoint.
	if a.store != nil {
		if tok, exp, ok := a.store.Load(); ok && tok != "" && time.Now().Add(tokenRefreshBuffer).Before(exp) {
			a.token, a.tokenExpiry = tok, exp
			return a.token, nil
		}
	}

	// Request new token
	return a.requestToken(ctx)
}

// requestToken requests a new access token from the EPO OAuth2 endpoint.
// Must be called with write lock held.
func (a *Authenticator) requestToken(ctx context.Context) (string, error) {
	// Create form data for token request
	data := url.Values{}
	data.Set("grant_type", "client_credentials")

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", a.authURL, strings.NewReader(data.Encode()))
	if err != nil {
		return "", fmt.Errorf("failed to create token request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Set Authorization header with Basic Auth (base64 encoded consumer key:secret)
	auth := base64.StdEncoding.EncodeToString([]byte(a.consumerKey + ":" + a.consumerSecret))
	req.Header.Set("Authorization", "Basic "+auth)

	// Send request
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to request token: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read token response: %w", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return "", &AuthError{
			StatusCode: resp.StatusCode,
			Message:    fmt.Sprintf("token request failed with status %d: %s", resp.StatusCode, string(body)),
		}
	}

	// Parse JSON response
	var tokenResp tokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("failed to parse token response: %w", err)
	}

	// Validate response
	if tokenResp.AccessToken == "" {
		return "", &AuthError{
			Message: "received empty access token",
		}
	}

	// Parse expires_in (returned as string by EPO API). Reject malformed or
	// non-positive values, which would otherwise cause a token refresh storm.
	expiresInSeconds, err := strconv.Atoi(tokenResp.ExpiresIn)
	if err != nil {
		return "", fmt.Errorf("failed to parse expires_in %q: %w", tokenResp.ExpiresIn, err)
	}
	if expiresInSeconds <= 0 {
		return "", fmt.Errorf("invalid expires_in %q: must be positive", tokenResp.ExpiresIn)
	}

	// Cache token with expiry
	a.token = tokenResp.AccessToken
	a.tokenExpiry = time.Now().Add(time.Duration(expiresInSeconds) * time.Second)
	if a.store != nil {
		a.store.Save(a.token, a.tokenExpiry)
	}

	return a.token, nil
}

// ClearToken clears the cached token, forcing a refresh on next request.
func (a *Authenticator) ClearToken() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.token = ""
	a.tokenExpiry = time.Time{}
}
