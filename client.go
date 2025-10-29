// Package epo_ops provides a Go client for the European Patent Office's Open Patent Services (OPS) API v3.2.
//
// This library provides an idiomatic Go interface to interact with the EPO's Open Patent Services,
// allowing you to retrieve patent bibliographic data, claims, descriptions, search for patents,
// get patent family information, download images, and more.
//
// Example usage:
//
//	config := &ops.Config{
//	    ConsumerKey:    "your-consumer-key",
//	    ConsumerSecret: "your-consumer-secret",
//	}
//
//	client, err := ops.NewClient(config)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	ctx := context.Background()
//	biblio, err := client.GetBiblio(ctx, "publication", "docdb", "EP1000000")
//	if err != nil {
//	    log.Fatal(err)
//	}
package epo_ops

//go:generate oapi-codegen -package generated -generate types openapi.yaml -o generated/types_gen.go
//go:generate oapi-codegen -package generated -generate client openapi.yaml -o generated/client_gen.go

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/patent-dev/epo-ops/generated"
)

// Client is the main EPO OPS API client.
type Client struct {
	config        *Config
	httpClient    *http.Client
	authenticator *Authenticator
	generated     *generated.Client
	quota         *quotaTracker
}

// getAcceptHeader returns the appropriate Accept header value based on the endpoint type.
// The EPO OPS API requires different Accept headers for different service endpoints.
func getAcceptHeader(endpoint string) string {
	switch endpoint {
	case EndpointBiblio, EndpointAbstract:
		return "application/exchange+xml"
	case EndpointFulltext, EndpointClaims, EndpointDescription:
		return "application/fulltext+xml"
	case EndpointFamily, EndpointLegal, EndpointSearch:
		return "application/ops+xml"
	case EndpointRegister:
		return "application/register+xml"
	case EndpointImages:
		return "application/tiff"
	default:
		return "application/xml"
	}
}

// getEndpointFromPath extracts the endpoint type from the URL path.
// This is used to determine the appropriate Accept header.
func getEndpointFromPath(path string) string {
	if strings.Contains(path, "/published-data/publication/") {
		// Parse the constituent (biblio, abstract, claims, description, fulltext)
		parts := strings.Split(path, "/")
		for i, part := range parts {
			if part == "publication" && i+3 < len(parts) {
				constituent := parts[i+3]
				switch constituent {
				case "biblio":
					return EndpointBiblio
				case "abstract":
					return EndpointAbstract
				case "claims":
					return EndpointClaims
				case "description":
					return EndpointDescription
				case "fulltext":
					return EndpointFulltext
				}
			}
		}
	}
	if strings.Contains(path, "/family/") {
		return EndpointFamily
	}
	if strings.Contains(path, "/legal") {
		return EndpointLegal
	}
	if strings.Contains(path, "/register") {
		return EndpointRegister
	}
	if strings.Contains(path, "/published-data/search") {
		return EndpointSearch
	}
	if strings.Contains(path, "/published-data/images") {
		return EndpointImages
	}
	return ""
}

// authTransport wraps an http.RoundTripper to add OAuth2 Bearer token to requests.
type authTransport struct {
	base          http.RoundTripper
	authenticator *Authenticator
}

func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Get valid token
	token, err := t.authenticator.GetToken(req.Context())
	if err != nil {
		return nil, err
	}

	// Clone request to avoid modifying original
	req2 := req.Clone(req.Context())
	req2.Header.Set("Authorization", "Bearer "+token)

	// Set Accept header based on endpoint type
	endpoint := getEndpointFromPath(req.URL.Path)
	if endpoint != "" {
		acceptHeader := getAcceptHeader(endpoint)
		req2.Header.Set("Accept", acceptHeader)
	}

	// Perform request
	return t.base.RoundTrip(req2)
}

// NewClient creates a new EPO OPS API client.
func NewClient(config *Config) (*Client, error) {
	if config == nil {
		config = DefaultConfig()
	}

	// Validate required fields
	if config.ConsumerKey == "" {
		return nil, &ConfigError{Message: "ConsumerKey is required"}
	}
	if config.ConsumerSecret == "" {
		return nil, &ConfigError{Message: "ConsumerSecret is required"}
	}

	// Set defaults if not provided
	if config.BaseURL == "" {
		config.BaseURL = "https://ops.epo.org/3.2/rest-services"
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}
	if config.RetryDelay == 0 {
		config.RetryDelay = 1 * time.Second
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	// Create base HTTP client
	baseClient := &http.Client{
		Timeout: config.Timeout,
	}

	// Create authenticator
	authenticator := NewAuthenticator(config.ConsumerKey, config.ConsumerSecret, baseClient)

	// Override auth URL if specified in config (mainly for testing)
	if config.AuthURL != "" {
		authenticator.authURL = config.AuthURL
	}

	// Create HTTP client with auth transport
	httpClient := &http.Client{
		Timeout: config.Timeout,
		Transport: &authTransport{
			base:          http.DefaultTransport,
			authenticator: authenticator,
		},
	}

	// Create generated client
	genClient, err := generated.NewClient(config.BaseURL, generated.WithHTTPClient(httpClient))
	if err != nil {
		return nil, err
	}

	return &Client{
		config:        config,
		httpClient:    httpClient,
		authenticator: authenticator,
		generated:     genClient,
		quota:         &quotaTracker{},
	}, nil
}

// executeRequest is a common helper that executes an HTTP request with retry logic and 401 handling.
// Returns the response body as bytes.
func (c *Client) executeRequest(ctx context.Context, fn func() (*http.Response, error)) ([]byte, error) {
	var retriedAfter401 atomic.Bool

	// Wrapper that handles 401 token refresh
	requestWithAuth := func() (*http.Response, error) {
		resp, err := fn()

		// Special handling for 401 errors: clear token and retry once
		// Use atomic swap to ensure only one retry happens even with concurrent requests
		if err == nil && resp.StatusCode == http.StatusUnauthorized && !retriedAfter401.Swap(true) {
			_ = resp.Body.Close() // Ignore close error, we're retrying the request

			// Clear cached token to force refresh on next attempt
			c.authenticator.ClearToken()

			// Retry the request immediately (token will be refreshed by authTransport)
			resp, err = fn()
		}

		return resp, err
	}

	// Execute with retry logic
	resp, err := c.retryableRequest(ctx, requestWithAuth)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Parse and store quota information from headers
	quotaInfo := ParseQuotaHeaders(resp.Header)
	c.quota.Update(quotaInfo)

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, c.handleErrorResponse(resp.StatusCode, body)
	}

	return body, nil
}

// makeRequest executes an HTTP request with retry logic and returns the response body as a string.
func (c *Client) makeRequest(ctx context.Context, fn func() (*http.Response, error)) (string, error) {
	body, err := c.executeRequest(ctx, fn)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

// makeBinaryRequest executes an HTTP request with retry logic and returns the response body as bytes.
// This is used for binary data like images.
func (c *Client) makeBinaryRequest(ctx context.Context, fn func() (*http.Response, error)) ([]byte, error) {
	return c.executeRequest(ctx, fn)
}

// handleErrorResponse converts HTTP error responses into appropriate error types.
func (c *Client) handleErrorResponse(statusCode int, body []byte) error {
	// Try to parse structured XML error first
	opsErr, err := parseErrorXML(body, statusCode)
	if err == nil && opsErr != nil {
		// Map specific error codes to appropriate error types
		switch opsErr.Code {
		case "CLIENT.InvalidReference", "SERVER.EntityNotFound", "HTTP.404":
			return &NotFoundError{
				Message: opsErr.Message,
			}
		case "CLIENT.InvalidAccessToken", "CLIENT.MissingAccessToken", "HTTP.401":
			return &AuthError{
				StatusCode: statusCode,
				Message:    opsErr.Message,
			}
		case "SERVER.RateLimitExceeded", "SERVER.QuotaPerWeekExceeded", "HTTP.429", "HTTP.403":
			return &QuotaExceededError{
				Message: opsErr.Message,
			}
		case "HTTP.503":
			return &ServiceUnavailableError{
				StatusCode: statusCode,
				Message:    opsErr.Message,
			}
		default:
			// Return the parsed OPSError for other codes
			return opsErr
		}
	}

	// Fall back to status-code-based error handling if XML parsing fails
	switch statusCode {
	case http.StatusNotFound:
		return &NotFoundError{
			Message: string(body),
		}
	case http.StatusUnauthorized:
		return &AuthError{
			StatusCode: statusCode,
			Message:    string(body),
		}
	case http.StatusTooManyRequests, http.StatusForbidden:
		return &QuotaExceededError{
			Message: string(body),
		}
	case http.StatusServiceUnavailable:
		return &ServiceUnavailableError{
			StatusCode: statusCode,
			Message:    string(body),
		}
	default:
		return fmt.Errorf("HTTP %d: %s", statusCode, string(body))
	}
}
func formatBulkBody(numbers []string) string {
	return strings.Join(numbers, "\n")
}

// GetBiblioMultiple retrieves bibliographic data for multiple patents (bulk operation).
// Uses POST endpoint for efficient batch retrieval of up to 100 patents in one request.
//
// GetLastQuota returns the last quota information from API responses.
// Returns nil if no API calls have been made yet.
//
// Quota tracking helps monitor fair use limits (4GB/week for non-paying users).
// The returned QuotaInfo includes:
//   - Status: "green" (<50%), "yellow" (50-75%), "red" (>75%), "black" (blocked)
//   - Individual: Quota for individual users
//   - Registered: Quota for registered/paying users
//   - Images: Separate quota for image downloads
func (c *Client) GetLastQuota() *QuotaInfo {
	return c.quota.Get()
}

// GetUsageStats retrieves usage statistics from the EPO OPS Data Usage API.
//
// The Data Usage API provides historical usage data for quota monitoring and analysis.
// Usage statistics are updated within 10 minutes of each hour and aligned on midnight
// UTC/GMT boundaries. This API does not count against quotas.
//
// Parameters:
//   - timeRange: Time range in one of two formats:
//   - Single date: "dd/mm/yyyy" (e.g., "01/01/2024")
//   - Date range: "dd/mm/yyyy~dd/mm/yyyy" (e.g., "01/01/2024~07/01/2024")
//
// Returns:
//   - UsageStats containing usage entries with timestamps, response sizes, and message counts
//   - error if the time range format is invalid or the request fails
//
// Example:
//
//	// Get usage for a specific date
//	stats, err := client.GetUsageStats(ctx, "01/01/2024")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Get usage for a date range
//	stats, err := client.GetUsageStats(ctx, "01/01/2024~07/01/2024")
//	for _, entry := range stats.Entries {
//	    fmt.Printf("Time: %d, Size: %d bytes, Messages: %d\n",
//	        entry.Timestamp, entry.TotalResponseSize, entry.MessageCount)
//	}
func (c *Client) GetUsageStats(ctx context.Context, timeRange string) (*UsageStats, error) {
	// Validate time range format
	if err := ValidateTimeRange(timeRange); err != nil {
		return nil, err
	}

	// Use generated client stub (endpoint now included in OpenAPI spec via convert-openapi.sh)
	params := &generated.GetUsageStatisticsParams{
		TimeRange: timeRange,
	}

	// Execute request using generated stub
	jsonData, err := c.makeRequest(ctx, func() (*http.Response, error) {
		return c.generated.GetUsageStatistics(ctx, params)
	})

	if err != nil {
		return nil, err
	}

	// Parse JSON response
	return parseUsageStats(jsonData, timeRange)
}
