package epo_ops

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
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

// Authenticator handles OAuth2 authentication for the EPO OPS API.
type Authenticator struct {
	authURL        string
	consumerKey    string
	consumerSecret string
	token          string
	tokenExpiry    time.Time
	httpClient     *http.Client
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
	defer resp.Body.Close()

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

	// Parse expires_in (returned as string by EPO API)
	var expiresInSeconds int
	if _, err := fmt.Sscanf(tokenResp.ExpiresIn, "%d", &expiresInSeconds); err != nil {
		return "", fmt.Errorf("failed to parse expires_in: %w", err)
	}

	// Cache token with expiry
	a.token = tokenResp.AccessToken
	a.tokenExpiry = time.Now().Add(time.Duration(expiresInSeconds) * time.Second)

	return a.token, nil
}

// ClearToken clears the cached token, forcing a refresh on next request.
func (a *Authenticator) ClearToken() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.token = ""
	a.tokenExpiry = time.Time{}
}
