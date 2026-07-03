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
	"time"

	"github.com/patent-dev/epo-ops/generated"
)

// Version is the library version. It surfaces through the default User-Agent.
const Version = "1.6.1"

// DefaultUserAgent identifies this library in outbound requests.
const DefaultUserAgent = "epo-ops-go/" + Version + " (patent.dev; +https://github.com/patent-dev/epo-ops)"

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

// uaTransport adds the User-Agent header to every outgoing request.
type uaTransport struct {
	base      http.RoundTripper
	userAgent string
}

func (t *uaTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	r := req.Clone(req.Context())
	r.Header.Set("User-Agent", t.userAgent)
	return t.base.RoundTrip(r)
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

	// Resolve the default User-Agent for all outbound traffic.
	userAgent := config.UserAgent
	if userAgent == "" {
		userAgent = DefaultUserAgent
	}

	// Resolve the base transport shared by API and token requests, so an injected
	// transport (e.g. egress rate limiting) governs all outbound EPO traffic. The
	// User-Agent is applied at this shared layer so token, data, and retry requests
	// all carry it while still funnelling through any injected transport.
	base := config.Transport
	if base == nil {
		base = http.DefaultTransport
	}
	base = &uaTransport{base: base, userAgent: userAgent}

	// Create base HTTP client (used for token requests)
	baseClient := &http.Client{
		Timeout:   config.Timeout,
		Transport: base,
	}

	// Create authenticator
	authenticator := NewAuthenticator(config.ConsumerKey, config.ConsumerSecret, baseClient)

	// Override auth URL if specified in config (mainly for testing)
	if config.AuthURL != "" {
		authenticator.authURL = config.AuthURL
	}

	// Wire the optional token store so tokens persist across clients.
	authenticator.store = config.TokenStore

	// Create HTTP client with auth transport
	httpClient := &http.Client{
		Timeout: config.Timeout,
		Transport: &authTransport{
			base:          base,
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
	// Execute with retry logic. The retry loop handles 401 token refresh
	// (clear token + refresh + retry once) so it composes correctly with the
	// backoff retries for transient 5xx responses.
	resp, err := c.retryableRequest(ctx, fn)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

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
		return nil, c.handleErrorResponse(resp.StatusCode, resp.Header, body)
	}

	// Some endpoints return HTTP 200 with a JSON error envelope wrapped in
	// an XML processing instruction. Surface it as a typed error so XML
	// parsers downstream are not asked to parse JSON.
	if jsonErr := parseEPOJSONErrorBody(body); jsonErr != nil {
		return nil, jsonErr
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
func (c *Client) handleErrorResponse(statusCode int, header http.Header, body []byte) error {
	retryAfter := ""
	if header != nil {
		retryAfter = header.Get("Retry-After")
	}

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
		case "SERVER.RateLimitExceeded", "SERVER.QuotaPerWeekExceeded", "HTTP.429":
			return &QuotaExceededError{
				Message: opsErr.Message,
			}
		case "HTTP.503":
			return &ServiceUnavailableError{
				StatusCode: statusCode,
				Message:    opsErr.Message,
				RetryAfter: retryAfter,
			}
		default:
			// A 403 only means quota/rate limiting when the EPO error code
			// explicitly says so; a bare 403 is a forbidden/access error.
			if statusCode == http.StatusForbidden {
				if isQuotaErrorCode(opsErr.Code) {
					return &QuotaExceededError{Message: opsErr.Message}
				}
				return &ForbiddenError{StatusCode: statusCode, Message: opsErr.Message}
			}
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
	case http.StatusForbidden:
		// Without a parseable quota code, treat a bare 403 as forbidden rather
		// than quota exceeded.
		return &ForbiddenError{
			StatusCode: statusCode,
			Message:    string(body),
		}
	case http.StatusTooManyRequests:
		return &QuotaExceededError{
			Message: string(body),
		}
	case http.StatusServiceUnavailable:
		return &ServiceUnavailableError{
			StatusCode: statusCode,
			Message:    string(body),
			RetryAfter: retryAfter,
		}
	default:
		return fmt.Errorf("HTTP %d: %s", statusCode, string(body))
	}
}

// isQuotaErrorCode reports whether an EPO error code denotes a quota or rate
// limit condition (the only cases where a 403 should map to QuotaExceededError).
func isQuotaErrorCode(code string) bool {
	switch code {
	case "SERVER.RateLimitExceeded", "SERVER.QuotaPerWeekExceeded",
		"SERVER.QuotaPerHourExceeded", "HTTP.429":
		return true
	default:
		return strings.Contains(code, "Quota") || strings.Contains(code, "RateLimit")
	}
}

// formatBulkBody joins patent numbers into the newline-separated body used by
// the EPO OPS bulk POST endpoints.
func formatBulkBody(numbers []string) string {
	return strings.Join(numbers, "\n")
}

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
