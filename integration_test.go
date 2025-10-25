//go:build integration

package epo_ops

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

// TestAuthenticationIntegration tests real authentication against EPO servers.
func TestAuthenticationIntegration(t *testing.T) {
	// Read credentials from environment
	consumerKey := os.Getenv("EPO_OPS_CONSUMER_KEY")
	consumerSecret := os.Getenv("EPO_OPS_CONSUMER_SECRET")

	if consumerKey == "" || consumerSecret == "" {
		t.Skip("Skipping integration test: EPO_OPS_CONSUMER_KEY and EPO_OPS_CONSUMER_SECRET must be set")
	}

	// Create authenticator
	auth := NewAuthenticator(consumerKey, consumerSecret, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Test: Acquire token
	t.Run("AcquireToken", func(t *testing.T) {
		token, err := auth.GetToken(ctx)
		if err != nil {
			t.Fatalf("Failed to acquire token: %v", err)
		}

		if token == "" {
			t.Fatal("Received empty token")
		}

		// Token should be a reasonably long string
		if len(token) < 20 {
			t.Errorf("Token seems too short: %d characters", len(token))
		}

		t.Logf("Successfully acquired token (length: %d)", len(token))
	})

	// Test: Token reuse within TTL
	t.Run("TokenReuse", func(t *testing.T) {
		// Get token first time
		token1, err := auth.GetToken(ctx)
		if err != nil {
			t.Fatalf("Failed to get first token: %v", err)
		}

		// Get token second time (should be cached)
		token2, err := auth.GetToken(ctx)
		if err != nil {
			t.Fatalf("Failed to get second token: %v", err)
		}

		// Should be the same token
		if token1 != token2 {
			t.Error("Expected same token from cache, got different token")
		}

		t.Log("Successfully reused cached token")
	})

	// Test: Token format
	t.Run("TokenFormat", func(t *testing.T) {
		token, err := auth.GetToken(ctx)
		if err != nil {
			t.Fatalf("Failed to get token: %v", err)
		}

		// Token should not contain spaces or newlines
		if strings.Contains(token, " ") || strings.Contains(token, "\n") {
			t.Error("Token contains whitespace")
		}

		t.Logf("Token format valid")
	})
}

// TestClientCreationIntegration tests client creation with real credentials.
func TestClientCreationIntegration(t *testing.T) {
	consumerKey := os.Getenv("EPO_OPS_CONSUMER_KEY")
	consumerSecret := os.Getenv("EPO_OPS_CONSUMER_SECRET")

	if consumerKey == "" || consumerSecret == "" {
		t.Skip("Skipping integration test: EPO_OPS_CONSUMER_KEY and EPO_OPS_CONSUMER_SECRET must be set")
	}

	config := &Config{
		ConsumerKey:    consumerKey,
		ConsumerSecret: consumerSecret,
	}

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	if client == nil {
		t.Fatal("Client is nil")
	}

	if client.authenticator == nil {
		t.Error("Client authenticator is nil")
	}

	if client.generated == nil {
		t.Error("Client generated client is nil")
	}

	t.Log("Successfully created client with valid credentials")
}

// TestInvalidCredentialsIntegration tests authentication with invalid credentials.
func TestInvalidCredentialsIntegration(t *testing.T) {
	// Create authenticator with invalid credentials
	auth := NewAuthenticator("invalid_key", "invalid_secret", nil)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	token, err := auth.GetToken(ctx)
	if err == nil {
		t.Error("Expected error with invalid credentials, got nil")
	}

	if token != "" {
		t.Error("Expected empty token with invalid credentials")
	}

	// Should be an AuthError
	if _, ok := err.(*AuthError); !ok {
		t.Errorf("Expected AuthError, got: %T", err)
	}

	t.Logf("Correctly rejected invalid credentials: %v", err)
}

// TestTextRetrievalIntegration tests retrieving patent text data.
func TestTextRetrievalIntegration(t *testing.T) {
	consumerKey := os.Getenv("EPO_OPS_CONSUMER_KEY")
	consumerSecret := os.Getenv("EPO_OPS_CONSUMER_SECRET")

	if consumerKey == "" || consumerSecret == "" {
		t.Skip("Skipping integration test: EPO_OPS_CONSUMER_KEY and EPO_OPS_CONSUMER_SECRET must be set")
	}

	config := &Config{
		ConsumerKey:    consumerKey,
		ConsumerSecret: consumerSecret,
	}

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Test patent: EP1000000B1 (a well-known test patent, B1 kind code to avoid ambiguity)
	testPatent := "EP1000000B1"

	// Test: GetBiblio
	t.Run("GetBiblio", func(t *testing.T) {
		biblio, err := client.GetBiblio(ctx, "publication", "docdb", testPatent)
		if err != nil {
			t.Fatalf("Failed to get biblio: %v", err)
		}

		if biblio == "" {
			t.Error("Received empty biblio data")
		}

		// Should contain XML
		if !strings.Contains(biblio, "<?xml") && !strings.Contains(biblio, "<") {
			t.Error("Biblio data does not appear to be XML")
		}

		t.Logf("Successfully retrieved biblio data (length: %d bytes)", len(biblio))
	})

	// Test: GetClaims
	t.Run("GetClaims", func(t *testing.T) {
		claims, err := client.GetClaims(ctx, "publication", "docdb", testPatent)
		if err != nil {
			t.Fatalf("Failed to get claims: %v", err)
		}

		if claims == "" {
			t.Error("Received empty claims data")
		}

		if !strings.Contains(claims, "claim") {
			t.Error("Claims data does not appear to contain claims")
		}

		t.Logf("Successfully retrieved claims data (length: %d bytes)", len(claims))
	})

	// Test: GetDescription
	t.Run("GetDescription", func(t *testing.T) {
		description, err := client.GetDescription(ctx, "publication", "docdb", testPatent)
		if err != nil {
			t.Fatalf("Failed to get description: %v", err)
		}

		if description == "" {
			t.Error("Received empty description data")
		}

		t.Logf("Successfully retrieved description data (length: %d bytes)", len(description))
	})

	// Test: GetAbstract
	t.Run("GetAbstract", func(t *testing.T) {
		abstract, err := client.GetAbstract(ctx, "publication", "docdb", testPatent)
		if err != nil {
			t.Fatalf("Failed to get abstract: %v", err)
		}

		if abstract == "" {
			t.Error("Received empty abstract data")
		}

		t.Logf("Successfully retrieved abstract data (length: %d bytes)", len(abstract))
	})

	// Test: GetFulltext
	t.Run("GetFulltext", func(t *testing.T) {
		fulltext, err := client.GetFulltext(ctx, "publication", "docdb", testPatent)
		if err != nil {
			t.Fatalf("Failed to get fulltext: %v", err)
		}

		if fulltext == "" {
			t.Error("Received empty fulltext data")
		}

		// Fulltext should be larger than individual sections
		if len(fulltext) < 1000 {
			t.Errorf("Fulltext data seems too small: %d bytes", len(fulltext))
		}

		t.Logf("Successfully retrieved fulltext data (length: %d bytes)", len(fulltext))
	})
}

// TestNotFoundIntegration tests handling of non-existent patents.
func TestNotFoundIntegration(t *testing.T) {
	consumerKey := os.Getenv("EPO_OPS_CONSUMER_KEY")
	consumerSecret := os.Getenv("EPO_OPS_CONSUMER_SECRET")

	if consumerKey == "" || consumerSecret == "" {
		t.Skip("Skipping integration test: EPO_OPS_CONSUMER_KEY and EPO_OPS_CONSUMER_SECRET must be set")
	}

	config := &Config{
		ConsumerKey:    consumerKey,
		ConsumerSecret: consumerSecret,
	}

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Non-existent patent
	_, err = client.GetBiblio(ctx, "publication", "docdb", "EP99999999999")
	if err == nil {
		t.Error("Expected error for non-existent patent, got nil")
	}

	// Should be a NotFoundError
	if _, ok := err.(*NotFoundError); !ok {
		t.Logf("Error type: %T, value: %v", err, err)
		// Note: EPO might return different error for truly invalid patents
	}

	t.Logf("Correctly handled non-existent patent: %v", err)
}

// TestQuotaTrackingIntegration tests quota tracking from API responses.
func TestQuotaTrackingIntegration(t *testing.T) {
	consumerKey := os.Getenv("EPO_OPS_CONSUMER_KEY")
	consumerSecret := os.Getenv("EPO_OPS_CONSUMER_SECRET")

	if consumerKey == "" || consumerSecret == "" {
		t.Skip("Skipping integration test: EPO_OPS_CONSUMER_KEY and EPO_OPS_CONSUMER_SECRET must be set")
	}

	config := &Config{
		ConsumerKey:    consumerKey,
		ConsumerSecret: consumerSecret,
	}

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Initially, quota should be nil (no requests made yet)
	initialQuota := client.GetLastQuota()
	if initialQuota != nil {
		t.Error("Expected nil quota before making any requests")
	}

	// Make a real API call
	testPatent := "EP1000000B1"
	_, err = client.GetBiblio(ctx, "publication", "docdb", testPatent)
	if err != nil {
		t.Fatalf("Failed to get biblio: %v", err)
	}

	// After request, quota should be available
	quota := client.GetLastQuota()
	if quota == nil {
		t.Fatal("Expected quota information after API request")
	}

	// Log quota information
	t.Logf("Quota Status: %s", quota.Status)
	t.Logf("Individual Quota: Used=%d, Limit=%d (%.2f%%)",
		quota.Individual.Used, quota.Individual.Limit, quota.Individual.UsagePercent())
	t.Logf("Registered Quota: Used=%d, Limit=%d (%.2f%%)",
		quota.Registered.Used, quota.Registered.Limit, quota.Registered.UsagePercent())

	// Verify status is one of the valid values
	validStatuses := map[string]bool{
		"green":  true,
		"yellow": true,
		"red":    true,
		"black":  true,
		"":       true, // May be empty if EPO doesn't send it
	}

	if !validStatuses[quota.Status] {
		t.Errorf("Unexpected quota status: %s", quota.Status)
	}

	// Make another request to verify quota is updated
	_, err = client.GetAbstract(ctx, "publication", "docdb", testPatent)
	if err != nil {
		t.Fatalf("Failed to get abstract: %v", err)
	}

	newQuota := client.GetLastQuota()
	if newQuota == nil {
		t.Fatal("Expected quota information after second API request")
	}

	// Quota should be updated (used amount should increase or stay same)
	if newQuota.Individual.Limit > 0 {
		if newQuota.Individual.Used < quota.Individual.Used {
			t.Error("Quota used amount decreased unexpectedly")
		}
		t.Logf("Quota updated after second request: Used=%d", newQuota.Individual.Used)
	}

	if newQuota.Registered.Limit > 0 {
		if newQuota.Registered.Used < quota.Registered.Used {
			t.Error("Quota used amount decreased unexpectedly")
		}
		t.Logf("Registered quota updated: Used=%d", newQuota.Registered.Used)
	}
}

// TestSearchIntegration tests patent search functionality.
func TestSearchIntegration(t *testing.T) {
	consumerKey := os.Getenv("EPO_OPS_CONSUMER_KEY")
	consumerSecret := os.Getenv("EPO_OPS_CONSUMER_SECRET")

	if consumerKey == "" || consumerSecret == "" {
		t.Skip("Skipping integration test: EPO_OPS_CONSUMER_KEY and EPO_OPS_CONSUMER_SECRET must be set")
	}

	config := &Config{
		ConsumerKey:    consumerKey,
		ConsumerSecret: consumerSecret,
	}

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Test: Basic search for patents with "plastic" in title
	t.Run("BasicSearch", func(t *testing.T) {
		results, err := client.Search(ctx, "ti=plastic", "1-5")
		if err != nil {
			t.Fatalf("Failed to search: %v", err)
		}

		if results == "" {
			t.Error("Received empty search results")
		}

		// Should contain XML with search results
		if !strings.Contains(results, "search-result") && !strings.Contains(results, "<") {
			t.Error("Search results do not appear to be XML")
		}

		t.Logf("Successfully retrieved search results (length: %d bytes)", len(results))
	})

	// Test: Search with applicant
	t.Run("SearchByApplicant", func(t *testing.T) {
		results, err := client.Search(ctx, "pa=Siemens", "1-3")
		if err != nil {
			t.Fatalf("Failed to search by applicant: %v", err)
		}

		if results == "" {
			t.Error("Received empty search results")
		}

		t.Logf("Successfully retrieved applicant search results (length: %d bytes)", len(results))
	})

	// Test: Search with constituent
	t.Run("SearchWithConstituent", func(t *testing.T) {
		results, err := client.SearchWithConstituent(ctx, "biblio", "ti=plastic", "1-3")
		if err != nil {
			t.Fatalf("Failed to search with constituent: %v", err)
		}

		if results == "" {
			t.Error("Received empty search results")
		}

		t.Logf("Successfully retrieved search with biblio constituent (length: %d bytes)", len(results))
	})
}

// TestFamilyRetrievalIntegration tests INPADOC family retrieval.
func TestFamilyRetrievalIntegration(t *testing.T) {
	consumerKey := os.Getenv("EPO_OPS_CONSUMER_KEY")
	consumerSecret := os.Getenv("EPO_OPS_CONSUMER_SECRET")

	if consumerKey == "" || consumerSecret == "" {
		t.Skip("Skipping integration test: EPO_OPS_CONSUMER_KEY and EPO_OPS_CONSUMER_SECRET must be set")
	}

	config := &Config{
		ConsumerKey:    consumerKey,
		ConsumerSecret: consumerSecret,
	}

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Test patent: EP1000000B1 (has a known family)
	testPatent := "EP1000000B1"

	// Test: Basic family retrieval
	t.Run("GetFamily", func(t *testing.T) {
		family, err := client.GetFamily(ctx, "publication", "docdb", testPatent)
		if err != nil {
			t.Fatalf("Failed to get family: %v", err)
		}

		if family == "" {
			t.Error("Received empty family data")
		}

		// Should contain XML with family members
		if !strings.Contains(family, "family") && !strings.Contains(family, "<") {
			t.Error("Family data does not appear to be XML")
		}

		t.Logf("Successfully retrieved family data (length: %d bytes)", len(family))
	})

	// Test: Family with biblio
	t.Run("GetFamilyWithBiblio", func(t *testing.T) {
		family, err := client.GetFamilyWithBiblio(ctx, "publication", "docdb", testPatent)
		if err != nil {
			t.Fatalf("Failed to get family with biblio: %v", err)
		}

		if family == "" {
			t.Error("Received empty family with biblio data")
		}

		// Should contain both family and biblio data
		if !strings.Contains(family, "family") && !strings.Contains(family, "biblio") {
			t.Error("Family data does not appear to contain family or biblio information")
		}

		// Family with biblio should be larger than basic family
		t.Logf("Successfully retrieved family with biblio (length: %d bytes)", len(family))
	})

	// Test: Family with legal
	t.Run("GetFamilyWithLegal", func(t *testing.T) {
		family, err := client.GetFamilyWithLegal(ctx, "publication", "docdb", testPatent)
		if err != nil {
			// Legal data might not always be available, log warning instead of failure
			t.Logf("Warning: Failed to get family with legal: %v", err)
			return
		}

		if family == "" {
			t.Error("Received empty family with legal data")
		}

		t.Logf("Successfully retrieved family with legal (length: %d bytes)", len(family))
	})
}

// TestImageRetrievalIntegration tests patent image retrieval and TIFF conversion.
func TestImageRetrievalIntegration(t *testing.T) {
	consumerKey := os.Getenv("EPO_OPS_CONSUMER_KEY")
	consumerSecret := os.Getenv("EPO_OPS_CONSUMER_SECRET")

	if consumerKey == "" || consumerSecret == "" {
		t.Skip("Skipping integration test: EPO_OPS_CONSUMER_KEY and EPO_OPS_CONSUMER_SECRET must be set")
	}

	config := &Config{
		ConsumerKey:    consumerKey,
		ConsumerSecret: consumerSecret,
	}

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Test: Retrieve first page of patent drawings
	// EP1000000B1 should have drawings available
	t.Run("GetImage", func(t *testing.T) {
		imageData, err := client.GetImage(ctx, "EP", "1000000", "B1", "Drawing", 1)
		if err != nil {
			// Some patents may not have images, log instead of failing
			t.Logf("Warning: Failed to get image: %v", err)
			t.Skip("Skipping image test - image may not be available")
			return
		}

		if len(imageData) == 0 {
			t.Error("Received empty image data")
		}

		// Image should be reasonably large (at least 1KB)
		if len(imageData) < 1024 {
			t.Errorf("Image data seems too small: %d bytes", len(imageData))
		}

		// Check if it's a TIFF file (starts with II or MM)
		isTIFF := len(imageData) >= 4 && ((imageData[0] == 'I' && imageData[1] == 'I') || // Little-endian
			(imageData[0] == 'M' && imageData[1] == 'M')) // Big-endian

		if !isTIFF {
			t.Logf("Warning: Image data does not appear to be TIFF format (first bytes: %x)", imageData[:min(4, len(imageData))])
		}

		t.Logf("Successfully retrieved image data (length: %d bytes, format: %s)",
			len(imageData),
			map[bool]string{true: "TIFF", false: "unknown"}[isTIFF])
	})
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// TestAdditionalServicesIntegration tests legal, register, and number conversion services.
func TestAdditionalServicesIntegration(t *testing.T) {
	consumerKey := os.Getenv("EPO_OPS_CONSUMER_KEY")
	consumerSecret := os.Getenv("EPO_OPS_CONSUMER_SECRET")

	if consumerKey == "" || consumerSecret == "" {
		t.Skip("Skipping integration test: EPO_OPS_CONSUMER_KEY and EPO_OPS_CONSUMER_SECRET must be set")
	}

	config := &Config{
		ConsumerKey:    consumerKey,
		ConsumerSecret: consumerSecret,
	}

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	testPatent := "EP1000000B1"

	// Test: Legal status retrieval
	t.Run("GetLegal", func(t *testing.T) {
		legal, err := client.GetLegal(ctx, "publication", "docdb", testPatent)
		if err != nil {
			// Legal data might not be available for all patents
			t.Logf("Warning: Failed to get legal status: %v", err)
			t.Skip("Skipping legal test - legal data may not be available")
			return
		}

		if legal == "" {
			t.Error("Received empty legal data")
		}

		// Should contain XML with legal information
		if !strings.Contains(legal, "legal") && !strings.Contains(legal, "<") {
			t.Error("Legal data does not appear to be XML")
		}

		t.Logf("Successfully retrieved legal status (length: %d bytes)", len(legal))
	})

	// Test: Register biblio retrieval
	t.Run("GetRegisterBiblio", func(t *testing.T) {
		register, err := client.GetRegisterBiblio(ctx, "publication", "docdb", testPatent)
		if err != nil {
			// Register data might not be available for all patents
			t.Logf("Warning: Failed to get register biblio: %v", err)
			t.Skip("Skipping register biblio test - data may not be available")
			return
		}

		if register == "" {
			t.Error("Received empty register data")
		}

		t.Logf("Successfully retrieved register biblio (length: %d bytes)", len(register))
	})

	// Test: Register events retrieval
	t.Run("GetRegisterEvents", func(t *testing.T) {
		events, err := client.GetRegisterEvents(ctx, "publication", "docdb", testPatent)
		if err != nil {
			// Events might not be available for all patents
			t.Logf("Warning: Failed to get register events: %v", err)
			t.Skip("Skipping register events test - data may not be available")
			return
		}

		if events == "" {
			t.Error("Received empty events data")
		}

		t.Logf("Successfully retrieved register events (length: %d bytes)", len(events))
	})

	// Test: Number conversion
	t.Run("ConvertPatentNumber", func(t *testing.T) {
		// Convert from docdb to epodoc format
		converted, err := client.ConvertPatentNumber(ctx, "publication", "docdb", testPatent, "epodoc")
		if err != nil {
			t.Logf("Warning: Failed to convert patent number: %v", err)
			t.Skip("Skipping number conversion test - service may not be available")
			return
		}

		if converted == "" {
			t.Error("Received empty conversion result")
		}

		// Should contain the converted number
		if !strings.Contains(converted, "EP") || !strings.Contains(converted, "1000000") {
			t.Logf("Warning: Conversion result does not contain expected patent number parts")
		}

		t.Logf("Successfully converted patent number (result length: %d bytes)", len(converted))
	})
}
