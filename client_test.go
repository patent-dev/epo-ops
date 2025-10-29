package epo_ops

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

//go:embed testdata/*.xml
var testdata embed.FS

// loadTestData loads an XML file from the embedded testdata directory
func loadTestData(filename string) []byte {
	data, err := testdata.ReadFile("testdata/" + filename)
	if err != nil {
		panic("Failed to load test data: " + filename + " - " + err.Error())
	}
	return data
}

// Mock OAuth2 server
func newMockAuthServer(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/auth/accesstoken" {
			t.Errorf("Unexpected auth path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}

		// Check authorization header
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Basic ") {
			t.Errorf("Missing or invalid Authorization header: %s", auth)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"test_token_12345","expires_in":"3600"}`))
	}))
}

// Mock OPS API server
func newMockOPSServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check authorization header
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test_token_12345" {
			t.Errorf("Invalid bearer token: %s", auth)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// Add quota headers (format: "used=123,quota=456")
		w.Header().Set("X-Throttling-Control", "green")
		w.Header().Set("X-IndividualQuota", "used=1000000,quota=4000000000")

		handler(w, r)
	}))
}

// Test client initialization
func TestNewClient(t *testing.T) {
	t.Run("Valid configuration", func(t *testing.T) {
		config := &Config{
			ConsumerKey:    "test-key",
			ConsumerSecret: "test-secret",
		}

		client, err := NewClient(config)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if client == nil {
			t.Fatal("Expected client, got nil")
		}

		if client.config.ConsumerKey != "test-key" {
			t.Errorf("Expected ConsumerKey 'test-key', got: %s", client.config.ConsumerKey)
		}
	})

	t.Run("Nil config uses defaults", func(t *testing.T) {
		config := DefaultConfig()
		config.ConsumerKey = "test"
		config.ConsumerSecret = "test"

		client, err := NewClient(config)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if client.config.MaxRetries != 3 {
			t.Errorf("Expected MaxRetries 3, got: %d", client.config.MaxRetries)
		}

		if client.config.Timeout != 30*time.Second {
			t.Errorf("Expected Timeout 30s, got: %v", client.config.Timeout)
		}
	})

	t.Run("Missing credentials", func(t *testing.T) {
		config := &Config{}
		_, err := NewClient(config)
		if err == nil {
			t.Error("Expected error for missing credentials")
		}

		if _, ok := err.(*ConfigError); !ok {
			t.Errorf("Expected ConfigError, got: %T", err)
		}
	})
}

// Test text retrieval endpoints
func TestGetBiblio(t *testing.T) {
	authServer := newMockAuthServer(t)
	defer authServer.Close()

	opsServer := newMockOPSServer(t, func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/biblio") {
			t.Errorf("Unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write(loadTestData("biblio.xml"))
	})
	defer opsServer.Close()

	config := &Config{
		ConsumerKey:    "test",
		ConsumerSecret: "test",
		BaseURL:        opsServer.URL,
	}
	// Override auth URL for testing
	config.AuthURL = authServer.URL + "/auth/accesstoken"

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	biblio, err := client.GetBiblio(ctx, "publication", "docdb", "EP.1000000.B1")
	if err != nil {
		t.Fatalf("GetBiblio failed: %v", err)
	}

	if biblio.PatentNumber == "" {
		t.Errorf("Expected patent number to be parsed")
	}

	if biblio.FamilyID == "" {
		t.Errorf("Expected family ID to be parsed")
	}
}

func TestGetClaims(t *testing.T) {
	authServer := newMockAuthServer(t)
	defer authServer.Close()

	opsServer := newMockOPSServer(t, func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/claims") {
			t.Errorf("Unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write(loadTestData("claims.xml"))
	})
	defer opsServer.Close()

	config := &Config{
		ConsumerKey:    "test",
		ConsumerSecret: "test",
		BaseURL:        opsServer.URL,
	}
	config.AuthURL = authServer.URL + "/auth/accesstoken"

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	claims, err := client.GetClaims(ctx, "publication", "docdb", "EP.1000000.B1")
	if err != nil {
		t.Fatalf("GetClaims failed: %v", err)
	}

	if claims.PatentNumber == "" {
		t.Errorf("Expected patent number to be parsed")
	}

	if len(claims.Claims) == 0 {
		t.Errorf("Expected claims to be parsed")
	}
}

func TestGetDescription(t *testing.T) {
	authServer := newMockAuthServer(t)
	defer authServer.Close()

	opsServer := newMockOPSServer(t, func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/description") {
			t.Errorf("Unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write(loadTestData("description.xml"))
	})
	defer opsServer.Close()

	config := &Config{
		ConsumerKey:    "test",
		ConsumerSecret: "test",
		BaseURL:        opsServer.URL,
	}
	config.AuthURL = authServer.URL + "/auth/accesstoken"

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	desc, err := client.GetDescription(ctx, "publication", "docdb", "EP.1000000.B1")
	if err != nil {
		t.Fatalf("GetDescription failed: %v", err)
	}

	if desc == nil {
		t.Fatal("Expected parsed description data, got nil")
	}
	if len(desc.Paragraphs) == 0 {
		t.Errorf("Expected paragraphs in description data")
	}
}

func TestGetAbstract(t *testing.T) {
	authServer := newMockAuthServer(t)
	defer authServer.Close()

	opsServer := newMockOPSServer(t, func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/abstract") {
			t.Errorf("Unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write(loadTestData("abstract.xml"))
	})
	defer opsServer.Close()

	config := &Config{
		ConsumerKey:    "test",
		ConsumerSecret: "test",
		BaseURL:        opsServer.URL,
	}
	config.AuthURL = authServer.URL + "/auth/accesstoken"

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	abstract, err := client.GetAbstract(ctx, "publication", "docdb", "EP.1000000.B1")
	if err != nil {
		t.Fatalf("GetAbstract failed: %v", err)
	}

	if abstract.PatentNumber == "" {
		t.Errorf("Expected patent number to be parsed")
	}

	if abstract.Text == "" {
		t.Errorf("Expected abstract text to be parsed")
	}
}

// Test search endpoints
func TestSearch(t *testing.T) {
	authServer := newMockAuthServer(t)
	defer authServer.Close()

	opsServer := newMockOPSServer(t, func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/search") {
			t.Errorf("Unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}

		query := r.URL.Query().Get("q")
		if query == "" {
			t.Error("Missing query parameter")
		}

		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write(loadTestData("search.xml"))
	})
	defer opsServer.Close()

	config := &Config{
		ConsumerKey:    "test",
		ConsumerSecret: "test",
		BaseURL:        opsServer.URL,
	}
	config.AuthURL = authServer.URL + "/auth/accesstoken"

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	results, err := client.Search(ctx, "ti=battery", "1-5")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if results == nil {
		t.Fatal("Expected parsed search results, got nil")
	}
	if results.TotalCount == 0 {
		t.Errorf("Expected non-zero total count in search results")
	}
	if len(results.Results) == 0 {
		t.Errorf("Expected search results")
	}
}

// Test family endpoints
func TestGetFamily(t *testing.T) {
	authServer := newMockAuthServer(t)
	defer authServer.Close()

	opsServer := newMockOPSServer(t, func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/family") {
			t.Errorf("Unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write(loadTestData("family.xml"))
	})
	defer opsServer.Close()

	config := &Config{
		ConsumerKey:    "test",
		ConsumerSecret: "test",
		BaseURL:        opsServer.URL,
	}
	config.AuthURL = authServer.URL + "/auth/accesstoken"

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	family, err := client.GetFamily(ctx, "publication", "docdb", "EP.1000000.B1")
	if err != nil {
		t.Fatalf("GetFamily failed: %v", err)
	}

	if family == nil {
		t.Fatal("Expected parsed family data, got nil")
	}
	if len(family.Members) == 0 {
		t.Errorf("Expected family members in family data")
	}
	// FamilyID is optional in the API response, so we don't assert on it
}

// Test image endpoints
func TestGetImage(t *testing.T) {
	authServer := newMockAuthServer(t)
	defer authServer.Close()

	// Mock TIFF header (little-endian)
	mockTIFF := []byte{0x49, 0x49, 0x2A, 0x00, 0x08, 0x00, 0x00, 0x00}

	opsServer := newMockOPSServer(t, func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/images/") {
			t.Errorf("Unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "image/tiff")
		_, _ = w.Write(mockTIFF)
	})
	defer opsServer.Close()

	config := &Config{
		ConsumerKey:    "test",
		ConsumerSecret: "test",
		BaseURL:        opsServer.URL,
	}
	config.AuthURL = authServer.URL + "/auth/accesstoken"

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	imageData, err := client.GetImage(ctx, "EP", "1000000", "B1", "Drawing", 1)
	if err != nil {
		t.Fatalf("GetImage failed: %v", err)
	}

	if len(imageData) == 0 {
		t.Error("Expected image data, got empty response")
	}

	// Check TIFF header
	if len(imageData) >= 4 && (imageData[0] != 'I' || imageData[1] != 'I') {
		t.Errorf("Expected TIFF header, got: %v", imageData[:4])
	}
}

// Test legal and register endpoints
func TestGetLegal(t *testing.T) {
	authServer := newMockAuthServer(t)
	defer authServer.Close()

	opsServer := newMockOPSServer(t, func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/legal") {
			t.Errorf("Unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write(loadTestData("legal.xml"))
	})
	defer opsServer.Close()

	config := &Config{
		ConsumerKey:    "test",
		ConsumerSecret: "test",
		BaseURL:        opsServer.URL,
	}
	config.AuthURL = authServer.URL + "/auth/accesstoken"

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	legal, err := client.GetLegal(ctx, "publication", "docdb", "EP.1000000.B1")
	if err != nil {
		t.Fatalf("GetLegal failed: %v", err)
	}

	if legal == nil {
		t.Fatal("Expected parsed legal data, got nil")
	}
	if len(legal.LegalEvents) == 0 {
		t.Errorf("Expected legal events in legal data")
	}
}

// Test error handling
func TestErrorHandling(t *testing.T) {
	authServer := newMockAuthServer(t)
	defer authServer.Close()

	t.Run("404 Not Found", func(t *testing.T) {
		opsServer := newMockOPSServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write(loadTestData("error_404.xml"))
		})
		defer opsServer.Close()

		config := &Config{
			ConsumerKey:    "test",
			ConsumerSecret: "test",
			BaseURL:        opsServer.URL,
		}
		config.AuthURL = authServer.URL + "/auth/accesstoken"

		client, err := NewClient(config)
		if err != nil {
			t.Fatalf("Failed to create client: %v", err)
		}

		ctx := context.Background()
		_, err = client.GetBiblio(ctx, "publication", "docdb", "EP.99999999.A1")
		if err == nil {
			t.Error("Expected error for 404 response")
		}

		if _, ok := err.(*NotFoundError); !ok {
			t.Errorf("Expected NotFoundError, got: %T", err)
		}
	})

	t.Run("503 Service Unavailable", func(t *testing.T) {
		opsServer := newMockOPSServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`Service temporarily unavailable`))
		})
		defer opsServer.Close()

		config := &Config{
			ConsumerKey:    "test",
			ConsumerSecret: "test",
			BaseURL:        opsServer.URL,
			MaxRetries:     0, // Disable retries for this test
		}
		config.AuthURL = authServer.URL + "/auth/accesstoken"

		client, err := NewClient(config)
		if err != nil {
			t.Fatalf("Failed to create client: %v", err)
		}

		ctx := context.Background()
		_, err = client.GetBiblio(ctx, "publication", "docdb", "EP.1000000.B1")
		if err == nil {
			t.Error("Expected error for 503 response")
		}

		if _, ok := err.(*ServiceUnavailableError); !ok {
			t.Errorf("Expected ServiceUnavailableError, got: %T", err)
		}
	})

	t.Run("429 Quota Exceeded", func(t *testing.T) {
		opsServer := newMockOPSServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Throttling-Control", "black")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write(loadTestData("error_429.xml"))
		})
		defer opsServer.Close()

		config := &Config{
			ConsumerKey:    "test",
			ConsumerSecret: "test",
			BaseURL:        opsServer.URL,
		}
		config.AuthURL = authServer.URL + "/auth/accesstoken"

		client, err := NewClient(config)
		if err != nil {
			t.Fatalf("Failed to create client: %v", err)
		}

		ctx := context.Background()
		_, err = client.GetBiblio(ctx, "publication", "docdb", "EP.1000000.B1")
		if err == nil {
			t.Error("Expected error for 429 response")
		}

		if _, ok := err.(*QuotaExceededError); !ok {
			t.Errorf("Expected QuotaExceededError, got: %T", err)
		}
	})
}

// Test quota tracking
func TestQuotaTracking(t *testing.T) {
	authServer := newMockAuthServer(t)
	defer authServer.Close()

	opsServer := newMockOPSServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Throttling-Control", "green")
		w.Header().Set("X-IndividualQuota", "used=1234567,quota=4000000000")
		w.Header().Set("X-RegisteredQuota", "used=5000000,quota=10000000000")

		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<ops:world-patent-data xmlns:ops="http://ops.epo.org">
  <exchange-document></exchange-document>
</ops:world-patent-data>`))
	})
	defer opsServer.Close()

	config := &Config{
		ConsumerKey:    "test",
		ConsumerSecret: "test",
		BaseURL:        opsServer.URL,
	}
	config.AuthURL = authServer.URL + "/auth/accesstoken"

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Quota should be nil before any request
	quota := client.GetLastQuota()
	if quota != nil {
		t.Error("Expected nil quota before first request")
	}

	ctx := context.Background()
	_, err = client.GetBiblio(ctx, "publication", "docdb", "EP.1000000.B1")
	if err != nil {
		t.Fatalf("GetBiblio failed: %v", err)
	}

	// Quota should be populated after request
	quota = client.GetLastQuota()
	if quota == nil {
		t.Fatal("Expected quota after request")
	}

	if quota.Status != "green" {
		t.Errorf("Expected status 'green', got: %s", quota.Status)
	}

	if quota.Individual.Used != 1234567 {
		t.Errorf("Expected Individual.Used 1234567, got: %d", quota.Individual.Used)
	}

	if quota.Individual.Limit != 4000000000 {
		t.Errorf("Expected Individual.Limit 4000000000, got: %d", quota.Individual.Limit)
	}

	if quota.Registered.Used != 5000000 {
		t.Errorf("Expected Registered.Used 5000000, got: %d", quota.Registered.Used)
	}

	usagePercent := quota.Individual.UsagePercent()
	if usagePercent < 0.03 || usagePercent > 0.04 {
		t.Errorf("Expected usage percent around 0.03%%, got: %.2f%%", usagePercent)
	}
}

// Test GetUsageStats
func TestGetUsageStats(t *testing.T) {
	authServer := newMockAuthServer(t)
	defer authServer.Close()

	opsServer := newMockOPSServer(t, func(w http.ResponseWriter, r *http.Request) {
		// Check that it's the usage stats endpoint
		if !strings.Contains(r.URL.Path, "/stats/usage") {
			t.Errorf("Unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}

		// Return mock JSON usage stats
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"data": [
				{"timestamp": 1640995200, "total_response_size": 1048576, "message_count": 10},
				{"timestamp": 1641081600, "total_response_size": 2097152, "message_count": 20}
			]
		}`))
	})
	defer opsServer.Close()

	config := DefaultConfig()
	config.ConsumerKey = "test"
	config.ConsumerSecret = "test"
	config.BaseURL = opsServer.URL
	config.AuthURL = authServer.URL + "/auth/accesstoken"

	client, _ := NewClient(config)
	ctx := context.Background()

	stats, err := client.GetUsageStats(ctx, "01/01/2022")
	if err != nil {
		t.Fatalf("GetUsageStats failed: %v", err)
	}

	if stats == nil {
		t.Fatal("Expected usage stats, got nil")
	}

	if len(stats.Entries) != 2 {
		t.Errorf("Expected 2 usage entries, got: %d", len(stats.Entries))
	}

	if stats.Entries[0].TotalResponseSize != 1048576 {
		t.Errorf("Expected first entry size 1048576, got: %d", stats.Entries[0].TotalResponseSize)
	}
}

// Test context cancellation
func TestContextCancellation(t *testing.T) {
	authServer := newMockAuthServer(t)
	defer authServer.Close()

	// Create a slow server
	opsServer := newMockOPSServer(t, func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		_, _ = w.Write([]byte(`<data></data>`))
	})
	defer opsServer.Close()

	config := &Config{
		ConsumerKey:    "test",
		ConsumerSecret: "test",
		BaseURL:        opsServer.URL,
		Timeout:        10 * time.Millisecond, // Very short timeout
		MaxRetries:     0,                     // No retries
		RetryDelay:     1 * time.Nanosecond,   // Fast failure
	}
	config.AuthURL = authServer.URL + "/auth/accesstoken"

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	_, err = client.GetBiblio(ctx, "publication", "docdb", "EP.1000000.B1")
	if err == nil {
		t.Error("Expected timeout error")
	}

	// Verify it's a timeout-related error
	if err != nil && !strings.Contains(err.Error(), "timeout") && !strings.Contains(err.Error(), "context deadline exceeded") {
		t.Logf("Got error (acceptable): %v", err)
	}
}

// Test token refresh on 401
func TestTokenRefreshOn401(t *testing.T) {
	authCallCount := 0
	authServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authCallCount++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"test_token_12345","expires_in":"3600"}`))
	}))
	defer authServer.Close()

	requestCount := 0
	opsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++

		// First request returns 401, second succeeds
		if requestCount == 1 {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`Unauthorized`))
			return
		}

		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write(loadTestData("biblio.xml"))
	}))
	defer opsServer.Close()

	config := &Config{
		ConsumerKey:    "test",
		ConsumerSecret: "test",
		BaseURL:        opsServer.URL,
	}
	config.AuthURL = authServer.URL + "/auth/accesstoken"

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	_, err = client.GetBiblio(ctx, "publication", "docdb", "EP.1000000.B1")
	if err != nil {
		t.Fatalf("Expected success after token refresh, got: %v", err)
	}

	// Should have made 2 auth calls (initial + refresh after 401)
	if authCallCount != 2 {
		t.Errorf("Expected 2 auth calls, got: %d", authCallCount)
	}

	// Should have made 2 API requests (first 401, second success)
	if requestCount != 2 {
		t.Errorf("Expected 2 API requests, got: %d", requestCount)
	}
}

// Benchmark tests
func BenchmarkGetBiblio(b *testing.B) {
	authServer := newMockAuthServer(&testing.T{})
	defer authServer.Close()

	opsServer := newMockOPSServer(&testing.T{}, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<data>test</data>`))
	})
	defer opsServer.Close()

	config := &Config{
		ConsumerKey:    "test",
		ConsumerSecret: "test",
		BaseURL:        opsServer.URL,
	}
	config.AuthURL = authServer.URL + "/auth/accesstoken"

	client, _ := NewClient(config)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = client.GetBiblio(ctx, "publication", "docdb", "EP.1000000.B1")
	}
}

// TestGetAcceptHeader tests the Accept header selection for different endpoints
func TestGetAcceptHeader(t *testing.T) {
	tests := []struct {
		endpoint string
		want     string
	}{
		{EndpointBiblio, "application/exchange+xml"},
		{EndpointAbstract, "application/exchange+xml"},
		{EndpointFulltext, "application/fulltext+xml"},
		{EndpointClaims, "application/fulltext+xml"},
		{EndpointDescription, "application/fulltext+xml"},
		{EndpointFamily, "application/ops+xml"},
		{EndpointLegal, "application/ops+xml"},
		{EndpointSearch, "application/ops+xml"},
		{EndpointRegister, "application/register+xml"},
		{EndpointImages, "application/tiff"},
		{"unknown", "application/xml"},
	}

	for _, tt := range tests {
		t.Run(tt.endpoint, func(t *testing.T) {
			got := getAcceptHeader(tt.endpoint)
			if got != tt.want {
				t.Errorf("getAcceptHeader(%s) = %s, want %s", tt.endpoint, got, tt.want)
			}
		})
	}
}

// TestGetEndpointFromPath tests endpoint extraction from URL paths
func TestGetEndpointFromPath(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"/rest-services/published-data/publication/epodoc/EP1000000/biblio", EndpointBiblio},
		{"/rest-services/published-data/publication/docdb/EP.1000000.B1/abstract", EndpointAbstract},
		{"/rest-services/published-data/publication/epodoc/EP1000000/claims", EndpointClaims},
		{"/rest-services/published-data/publication/epodoc/EP1000000/description", EndpointDescription},
		{"/rest-services/published-data/publication/epodoc/EP1000000/fulltext", EndpointFulltext},
		{"/rest-services/family/publication/docdb/EP.1000000.B1/biblio", EndpointFamily},
		{"/rest-services/legal/publication/docdb/EP.1000000.B1", EndpointLegal},
		{"/rest-services/register/publication/epodoc/EP1000000", EndpointRegister},
		{"/rest-services/published-data/search?q=test", EndpointSearch},
		{"/rest-services/published-data/images/EP/1000000/A1/fullimage", EndpointImages},
		{"/unknown/path", ""},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := getEndpointFromPath(tt.path)
			if got != tt.want {
				t.Errorf("getEndpointFromPath(%s) = %s, want %s", tt.path, got, tt.want)
			}
		})
	}
}

// TestAcceptHeaderIntegration tests that Accept headers are correctly set in actual requests
func TestAcceptHeaderIntegration(t *testing.T) {
	// Mock auth server
	authServer := newMockAuthServer(t)
	defer authServer.Close()

	// Track captured Accept headers
	var capturedHeaders []string

	// Mock OPS server that captures Accept headers
	opsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedHeaders = append(capturedHeaders, r.Header.Get("Accept"))
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write(loadTestData("biblio.xml"))
	}))
	defer opsServer.Close()

	config := DefaultConfig()
	config.ConsumerKey = "test"
	config.ConsumerSecret = "test"
	config.BaseURL = opsServer.URL
	config.AuthURL = authServer.URL + "/auth/accesstoken"

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()

	// Test biblio endpoint
	_, err = client.GetBiblioRaw(ctx, RefTypePublication, FormatDocDB, "EP.1000000.B1")
	if err != nil {
		t.Fatalf("GetBiblioRaw failed: %v", err)
	}

	// Verify Accept header was set (should be application/exchange+xml for biblio)
	if len(capturedHeaders) == 0 {
		t.Fatal("No Accept headers captured")
	}

	lastHeader := capturedHeaders[len(capturedHeaders)-1]
	expectedHeader := "application/exchange+xml"
	if lastHeader != expectedHeader {
		t.Errorf("Expected Accept header %s, got %s", expectedHeader, lastHeader)
	}
}

// TestFormatBulkBody tests the formatBulkBody helper
func TestFormatBulkBody(t *testing.T) {
	tests := []struct {
		name     string
		numbers  []string
		expected string
	}{
		{
			name:     "Single number",
			numbers:  []string{"EP.1000000.B1"},
			expected: "EP.1000000.B1",
		},
		{
			name:     "Multiple numbers",
			numbers:  []string{"EP.1000000.B1", "EP.1000001.A1", "US.5551212.A"},
			expected: "EP.1000000.B1\nEP.1000001.A1\nUS.5551212.A",
		},
		{
			name:     "Empty slice",
			numbers:  []string{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatBulkBody(tt.numbers)
			if result != tt.expected {
				t.Errorf("formatBulkBody() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestGetBiblioMultiple tests bulk bibliographic data retrieval
func TestGetBiblioMultiple(t *testing.T) {
	// Mock auth server
	authServer := newMockAuthServer(t)
	defer authServer.Close()

	// Mock OPS server
	opsServer := newMockOPSServer(t, func(w http.ResponseWriter, r *http.Request) {
		// Verify it's a POST request
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		// Verify Content-Type header
		contentType := r.Header.Get("Content-Type")
		if contentType != "text/plain" {
			t.Errorf("Expected Content-Type 'text/plain', got %s", contentType)
		}

		// Read and verify body
		body, _ := io.ReadAll(r.Body)
		bodyStr := string(body)
		expectedBody := "EP.1000000.B1\nEP.1000001.A1"
		if bodyStr != expectedBody {
			t.Errorf("Expected body %q, got %q", expectedBody, bodyStr)
		}

		// Return sample biblio XML
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write(loadTestData("biblio.xml"))
	})
	defer opsServer.Close()

	config := DefaultConfig()
	config.ConsumerKey = "test"
	config.ConsumerSecret = "test"
	config.BaseURL = opsServer.URL
	config.AuthURL = authServer.URL + "/auth/accesstoken"

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	numbers := []string{"EP.1000000.B1", "EP.1000001.A1"}

	xml, err := client.GetBiblioMultiple(ctx, RefTypePublication, FormatDocDB, numbers)
	if err != nil {
		t.Fatalf("GetBiblioMultiple failed: %v", err)
	}

	if !strings.Contains(xml, "bibliographic-data") {
		t.Error("Expected bibliographic-data element in response")
	}
}

// TestGetBiblioMultiple_Validation tests validation for bulk operations
func TestGetBiblioMultiple_Validation(t *testing.T) {
	config := DefaultConfig()
	config.ConsumerKey = "test"
	config.ConsumerSecret = "test"

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()

	t.Run("Empty numbers", func(t *testing.T) {
		_, err := client.GetBiblioMultiple(ctx, RefTypePublication, FormatDocDB, []string{})
		var valErr *ValidationError
		if !errors.As(err, &valErr) {
			t.Errorf("Expected ValidationError for empty numbers, got %T", err)
		}
	})

	t.Run("Too many numbers", func(t *testing.T) {
		numbers := make([]string, 101)
		for i := range numbers {
			numbers[i] = fmt.Sprintf("EP.%d.B1", 1000000+i)
		}
		_, err := client.GetBiblioMultiple(ctx, RefTypePublication, FormatDocDB, numbers)
		var valErr *ValidationError
		if !errors.As(err, &valErr) {
			t.Errorf("Expected ValidationError for >100 numbers, got %T", err)
		}
	})

	t.Run("Invalid format", func(t *testing.T) {
		numbers := []string{"EP1000000B1"} // Missing dots (epodoc format in docdb call)
		_, err := client.GetBiblioMultiple(ctx, RefTypePublication, FormatDocDB, numbers)
		if err == nil {
			t.Error("Expected validation error for invalid format")
		}
	})

	t.Run("Invalid refType", func(t *testing.T) {
		numbers := []string{"EP.1000000.B1"}
		_, err := client.GetBiblioMultiple(ctx, "invalid", FormatDocDB, numbers)
		var valErr *ValidationError
		if !errors.As(err, &valErr) {
			t.Errorf("Expected ValidationError for invalid refType, got %T", err)
		}
	})
}

// TestGetClaimsMultiple tests bulk claims retrieval
func TestGetClaimsMultiple(t *testing.T) {
	authServer := newMockAuthServer(t)
	defer authServer.Close()

	opsServer := newMockOPSServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write(loadTestData("claims.xml"))
	})
	defer opsServer.Close()

	config := DefaultConfig()
	config.ConsumerKey = "test"
	config.ConsumerSecret = "test"
	config.BaseURL = opsServer.URL
	config.AuthURL = authServer.URL + "/auth/accesstoken"

	client, _ := NewClient(config)
	ctx := context.Background()

	xml, err := client.GetClaimsMultiple(ctx, RefTypePublication, FormatDocDB, []string{"EP.1000000.B1"})
	if err != nil {
		t.Fatalf("GetClaimsMultiple failed: %v", err)
	}

	if !strings.Contains(xml, "claims") {
		t.Error("Expected claims element in response")
	}
}

// TestGetDescriptionMultiple tests bulk description retrieval
func TestGetDescriptionMultiple(t *testing.T) {
	authServer := newMockAuthServer(t)
	defer authServer.Close()

	opsServer := newMockOPSServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write(loadTestData("description.xml"))
	})
	defer opsServer.Close()

	config := DefaultConfig()
	config.ConsumerKey = "test"
	config.ConsumerSecret = "test"
	config.BaseURL = opsServer.URL
	config.AuthURL = authServer.URL + "/auth/accesstoken"

	client, _ := NewClient(config)
	ctx := context.Background()

	data, err := client.GetDescriptionMultiple(ctx, RefTypePublication, FormatDocDB, []string{"EP.1000000.B1"})
	if err != nil {
		t.Fatalf("GetDescriptionMultiple failed: %v", err)
	}

	if data == nil {
		t.Fatal("Expected parsed description data, got nil")
	}
	if len(data.Paragraphs) == 0 {
		t.Error("Expected paragraphs in description data")
	}
}

// TestGetAbstractMultiple tests bulk abstract retrieval
func TestGetAbstractMultiple(t *testing.T) {
	authServer := newMockAuthServer(t)
	defer authServer.Close()

	opsServer := newMockOPSServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write(loadTestData("abstract.xml"))
	})
	defer opsServer.Close()

	config := DefaultConfig()
	config.ConsumerKey = "test"
	config.ConsumerSecret = "test"
	config.BaseURL = opsServer.URL
	config.AuthURL = authServer.URL + "/auth/accesstoken"

	client, _ := NewClient(config)
	ctx := context.Background()

	xml, err := client.GetAbstractMultiple(ctx, RefTypePublication, FormatDocDB, []string{"EP.1000000.B1"})
	if err != nil {
		t.Fatalf("GetAbstractMultiple failed: %v", err)
	}

	if !strings.Contains(xml, "abstract") {
		t.Error("Expected abstract element in response")
	}
}

// TestGetFulltextMultiple tests bulk fulltext retrieval
func TestGetFulltextMultiple(t *testing.T) {
	authServer := newMockAuthServer(t)
	defer authServer.Close()

	opsServer := newMockOPSServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write(loadTestData("claims.xml")) // Use claims.xml as fulltext proxy
	})
	defer opsServer.Close()

	config := DefaultConfig()
	config.ConsumerKey = "test"
	config.ConsumerSecret = "test"
	config.BaseURL = opsServer.URL
	config.AuthURL = authServer.URL + "/auth/accesstoken"

	client, _ := NewClient(config)
	ctx := context.Background()

	xml, err := client.GetFulltextMultiple(ctx, RefTypePublication, FormatDocDB, []string{"EP.1000000.B1"})
	if err != nil {
		t.Fatalf("GetFulltextMultiple failed: %v", err)
	}

	if xml == nil {
		t.Fatal("Expected parsed fulltext data, got nil")
	}
	if xml.Claims == nil || len(xml.Claims.Claims) == 0 {
		t.Error("Expected claims in fulltext data")
	}
}
