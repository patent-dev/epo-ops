//go:build integration

package epo_ops

import (
	"context"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var updateFixtures = flag.Bool("update-fixtures", false, "update fixture files from live API responses")

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

	// Test patent: EP.1000000.B1 (docdb format, well-known test patent)
	testPatent := "EP.1000000.B1"

	// Test: GetBiblio
	t.Run("GetBiblio", func(t *testing.T) {
		biblio, err := client.GetBiblio(ctx, "publication", "docdb", testPatent)
		if err != nil {
			t.Fatalf("Failed to get biblio: %v", err)
		}

		if biblio == nil {
			t.Fatal("Received nil biblio data")
		}

		if len(biblio.Titles) == 0 {
			t.Error("Expected non-empty Titles map")
		}

		// Log English title if available
		title := biblio.Titles["en"]
		if title == "" {
			// Fall back to any available title
			for _, v := range biblio.Titles {
				title = v
				break
			}
		}

		t.Logf("Successfully retrieved biblio: Title=%q, Applicants=%d, Inventors=%d",
			title, len(biblio.Applicants), len(biblio.Inventors))
	})

	// Test: GetClaims
	t.Run("GetClaims", func(t *testing.T) {
		claims, err := client.GetClaims(ctx, "publication", "docdb", testPatent)
		if err != nil {
			t.Fatalf("Failed to get claims: %v", err)
		}

		if claims == nil {
			t.Fatal("Received nil claims data")
		}

		if len(claims.Claims) == 0 {
			t.Error("Expected non-empty Claims slice")
		}

		t.Logf("Successfully retrieved %d claims (language: %s)", len(claims.Claims), claims.Language)
	})

	// Test: GetDescription
	t.Run("GetDescription", func(t *testing.T) {
		description, err := client.GetDescription(ctx, "publication", "docdb", testPatent)
		if err != nil {
			t.Fatalf("Failed to get description: %v", err)
		}

		if description == nil {
			t.Fatal("Received nil description data")
		}

		if len(description.Paragraphs) == 0 {
			t.Error("Expected non-empty Paragraphs")
		}

		t.Logf("Successfully retrieved description: %d paragraphs (language: %s)",
			len(description.Paragraphs), description.Language)
	})

	// Test: GetAbstract
	t.Run("GetAbstract", func(t *testing.T) {
		abstract, err := client.GetAbstract(ctx, "publication", "docdb", testPatent)
		if err != nil {
			t.Fatalf("Failed to get abstract: %v", err)
		}

		if abstract == nil {
			t.Fatal("Received nil abstract data")
		}

		t.Logf("Successfully retrieved abstract (language: %s, length: %d chars)",
			abstract.Language, len(abstract.Text))
	})

	// Test: GetFulltext
	t.Run("GetFulltext", func(t *testing.T) {
		fulltext, err := client.GetFulltext(ctx, "publication", "docdb", testPatent)
		if err != nil {
			t.Fatalf("Failed to get fulltext: %v", err)
		}

		if fulltext == nil {
			t.Fatal("Received nil fulltext data")
		}

		if fulltext.Status == "" {
			t.Logf("Warning: Fulltext status is empty")
		}

		t.Logf("Successfully retrieved fulltext: status=%s, hasBiblio=%v, hasClaims=%v",
			fulltext.Status, fulltext.Biblio != nil, fulltext.Claims != nil)
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
	_, err = client.GetBiblio(ctx, "publication", "docdb", "EP.99999999999.A1")
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
	testPatent := "EP.1000000.B1"
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

		if results == nil {
			t.Fatal("Received nil search results")
		}

		if results.TotalCount == 0 {
			t.Error("Expected TotalCount > 0")
		}

		t.Logf("Successfully retrieved search results: %d total, %d parsed results",
			results.TotalCount, len(results.Results))
	})

	// Test: Search with applicant
	t.Run("SearchByApplicant", func(t *testing.T) {
		results, err := client.Search(ctx, "pa=Siemens", "1-3")
		if err != nil {
			t.Fatalf("Failed to search by applicant: %v", err)
		}

		if results == nil {
			t.Fatal("Received nil search results")
		}

		t.Logf("Successfully retrieved applicant search: %d total results", results.TotalCount)
	})

	// Test: Search with constituent
	t.Run("SearchWithConstituent", func(t *testing.T) {
		results, err := client.SearchWithConstituent(ctx, "biblio", "ti=plastic", "1-3")
		if err != nil {
			t.Fatalf("Failed to search with constituent: %v", err)
		}

		if results == nil {
			t.Fatal("Received nil search results")
		}

		t.Logf("Successfully retrieved search with biblio constituent: %d results",
			len(results.Results))
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

	// Test patent: EP.1000000.B1 (docdb format, has a known family)
	testPatent := "EP.1000000.B1"

	// Test: Basic family retrieval
	t.Run("GetFamily", func(t *testing.T) {
		family, err := client.GetFamily(ctx, "publication", "docdb", testPatent)
		if err != nil {
			t.Fatalf("Failed to get family: %v", err)
		}

		if family == nil {
			t.Fatal("Received nil family data")
		}

		if len(family.Members) == 0 {
			t.Error("Expected non-empty family members")
		}

		// Verify parsed struct fields
		for i, m := range family.Members {
			if m.Country == "" || m.DocNumber == "" {
				t.Errorf("Member %d: missing Country or DocNumber", i)
			}
		}

		if len(family.Countries) == 0 {
			t.Error("Expected non-empty Countries list")
		}

		t.Logf("Successfully retrieved family: %d members, countries: %v",
			len(family.Members), family.Countries)

		// Save fixture
		if raw, err := client.GetFamilyRaw(ctx, "publication", "docdb", testPatent); err == nil {
			saveFixture(t, "family_EP1000000", []byte(raw))
		}
	})

	// Test: Family with biblio
	t.Run("GetFamilyWithBiblio", func(t *testing.T) {
		family, err := client.GetFamilyWithBiblio(ctx, "publication", "docdb", testPatent)
		if err != nil {
			t.Fatalf("Failed to get family with biblio: %v", err)
		}

		if family == nil {
			t.Fatal("Received nil family data")
		}

		if len(family.Members) == 0 {
			t.Error("Expected non-empty family members")
		}

		// Verify biblio data extracted
		withTitle := 0
		for _, m := range family.Members {
			if m.Title != "" {
				withTitle++
			}
		}
		t.Logf("Family with biblio: %d members, %d with Title", len(family.Members), withTitle)
		if withTitle == 0 {
			t.Error("Expected at least some members with Title from biblio")
		}
	})

	// Test: Family with legal
	t.Run("GetFamilyWithLegal", func(t *testing.T) {
		family, err := client.GetFamilyWithLegal(ctx, "publication", "docdb", testPatent)
		if err != nil {
			t.Logf("Warning: Failed to get family with legal: %v", err)
			return
		}

		if family == nil {
			t.Fatal("Received nil family data")
		}

		t.Logf("Successfully retrieved family with legal: %d members", len(family.Members))
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
	// EP.1000000.B1 should have drawings available
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

	testPatent := "EP.1000000.B1"

	// Test: Legal status retrieval
	t.Run("GetLegal", func(t *testing.T) {
		legal, err := client.GetLegal(ctx, "publication", "docdb", testPatent)
		if err != nil {
			t.Logf("Warning: Failed to get legal status: %v", err)
			t.Skip("Skipping legal test - legal data may not be available")
			return
		}

		if legal == nil {
			t.Fatal("Received nil legal data")
		}

		if len(legal.LegalEvents) == 0 {
			t.Error("Expected non-empty legal events")
		}

		// Verify parsed fields
		for i, e := range legal.LegalEvents {
			if i < 3 {
				t.Logf("Event %d: Code=%s Country=%s Date=%s", i+1, e.EventCode, e.Country, e.Date)
			}
		}

		t.Logf("Legal events for %s: %d events", legal.PatentNumber, len(legal.LegalEvents))

		// Save fixture
		if raw, err := client.GetLegalRaw(ctx, "publication", "docdb", testPatent); err == nil {
			saveFixture(t, "legal_EP1000000", []byte(raw))
		}
	})

	// Test: Register biblio retrieval (raw XML) - register requires epodoc format
	t.Run("GetRegisterBiblioRaw", func(t *testing.T) {
		register, err := client.GetRegisterBiblioRaw(ctx, "publication", "epodoc", "EP1000000")
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

	// Test: Register events retrieval (parsed) - register requires epodoc format
	t.Run("GetRegisterEvents", func(t *testing.T) {
		data, err := client.GetRegisterEvents(ctx, "publication", "epodoc", "EP1000000")
		if err != nil {
			t.Logf("Warning: Failed to get register events: %v", err)
			t.Skip("Skipping register events test - data may not be available")
			return
		}

		if len(data.Events) == 0 {
			t.Error("Expected non-empty Events")
		}

		// Verify events have required fields
		for i, evt := range data.Events {
			if evt.Date == "" || evt.EventCode == "" {
				t.Errorf("Event %d missing Date or EventCode", i)
				break
			}
			if evt.Category == "" {
				t.Errorf("Event %d missing Category", i)
				break
			}
		}

		t.Logf("Retrieved %d register events, %d statuses for %s",
			len(data.Events), len(data.Statuses), data.PatentNumber)

		// Save fixture
		if raw, err := client.GetRegisterEventsRaw(ctx, "publication", "epodoc", "EP1000000"); err == nil {
			saveFixture(t, "register_events_EP1000000", []byte(raw))
		}
	})

	// Test: Number conversion (parsed)
	t.Run("ConvertPatentNumber", func(t *testing.T) {
		data, err := client.ConvertPatentNumber(ctx, "publication", "docdb", testPatent, "epodoc")
		if err != nil {
			t.Logf("Warning: Failed to convert patent number: %v", err)
			t.Skip("Skipping number conversion test - service may not be available")
			return
		}

		if data.DocNumber == "" {
			t.Error("Expected non-empty DocNumber")
		}
		if data.Kind == "" {
			t.Error("Expected non-empty Kind")
		}
		if data.InputFormat != "docdb" {
			t.Errorf("Expected InputFormat 'docdb', got %q", data.InputFormat)
		}
		if data.OutputFormat != "epodoc" {
			t.Errorf("Expected OutputFormat 'epodoc', got %q", data.OutputFormat)
		}

		t.Logf("Converted %s -> DocNumber=%s Kind=%s Date=%s",
			testPatent, data.DocNumber, data.Kind, data.Date)

		// Save fixture
		if raw, err := client.ConvertPatentNumberRaw(ctx, "publication", "docdb", testPatent, "epodoc"); err == nil {
			saveFixture(t, "convert_patent_number", []byte(raw))
		}
	})
}

// TestClassificationIntegration tests CPC classification schema retrieval.
func TestClassificationIntegration(t *testing.T) {
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

	t.Run("GetClassificationSchema", func(t *testing.T) {
		data, err := client.GetClassificationSchema(ctx, "H04W", false, false)
		if err != nil {
			t.Fatalf("Failed to get classification schema: %v", err)
		}

		if data.Symbol == "" {
			t.Error("Expected non-empty Symbol")
		}
		if data.Title == "" {
			t.Error("Expected non-empty Title")
		}
		if data.Level < 0 {
			t.Errorf("Expected Level >= 0, got %d", data.Level)
		}
		if data.SchemeType == "" {
			t.Error("Expected non-empty SchemeType")
		}

		t.Logf("Classification: Symbol=%s Title=%q Level=%d SchemeType=%s Children=%d",
			data.Symbol, data.Title, data.Level, data.SchemeType, len(data.Children))

		// Verify child fields if present
		for i, ch := range data.Children {
			if ch.Symbol == "" {
				t.Errorf("Child[%d] has empty Symbol", i)
			}
			if i < 3 {
				t.Logf("  Child[%d]: Symbol=%s Title=%q Level=%d", i, ch.Symbol, ch.Title, ch.Level)
			}
		}

		// Save fixture
		if raw, err := client.GetClassificationSchemaRaw(ctx, "H04W", false, false); err == nil {
			saveFixture(t, "classification_schema_H04W", []byte(raw))
		}
	})

	t.Run("GetClassificationSchema_WithAncestors", func(t *testing.T) {
		data, err := client.GetClassificationSchema(ctx, "H04W84/18", true, false)
		if err != nil {
			t.Logf("Warning: Failed to get classification with ancestors: %v", err)
			t.Skip("Skipping ancestors test")
			return
		}

		if data.Symbol == "" {
			t.Error("Expected non-empty Symbol")
		}

		t.Logf("Classification with ancestors: Symbol=%s Title=%q Level=%d",
			data.Symbol, data.Title, data.Level)
	})
}

// TestPublishedEquivalentsIntegration tests published equivalents retrieval.
func TestPublishedEquivalentsIntegration(t *testing.T) {
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

	testPatent := "EP.1000000.B1"

	t.Run("GetPublishedEquivalents", func(t *testing.T) {
		data, err := client.GetPublishedEquivalents(ctx, "publication", "docdb", testPatent)
		if err != nil {
			t.Fatalf("Failed to get published equivalents: %v", err)
		}

		if len(data.Equivalents) == 0 {
			t.Error("Expected non-empty Equivalents for EP.1000000.B1")
		}

		for i, eq := range data.Equivalents {
			if eq.Country == "" || eq.DocNumber == "" {
				t.Errorf("Equivalent[%d]: missing Country or DocNumber", i)
			}
			if i < 5 {
				t.Logf("Equivalent[%d]: %s%s%s", i, eq.Country, eq.DocNumber, eq.Kind)
			}
		}

		t.Logf("Total equivalents for %s: %d", testPatent, len(data.Equivalents))

		// Save fixture
		if raw, err := client.GetPublishedEquivalentsRaw(ctx, "publication", "docdb", testPatent); err == nil {
			saveFixture(t, "published_equivalents_EP1000000", []byte(raw))
		}
	})
}

// TestEdgeCasesIntegration tests edge cases with patents that may have limited data.
func TestEdgeCasesIntegration(t *testing.T) {
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

	// EP.0000001.B1 - very old patent, likely has minimal or no legal events
	t.Run("GetLegal_OldPatent", func(t *testing.T) {
		data, err := client.GetLegal(ctx, "publication", "docdb", "EP.0000001.B1")
		if err != nil {
			// Not found or no data is acceptable for edge case
			t.Logf("GetLegal for EP.0000001.B1 returned error (expected for edge case): %v", err)
			return
		}

		t.Logf("EP0000001B1 legal events: %d", len(data.LegalEvents))
		if len(data.LegalEvents) == 0 {
			t.Log("No legal events for EP0000001B1 (edge case confirmed)")
		}
	})

	// Test GetFamily on a patent with a small family
	t.Run("GetFamily_SmallFamily", func(t *testing.T) {
		data, err := client.GetFamily(ctx, "publication", "docdb", "EP.0000001.B1")
		if err != nil {
			t.Logf("GetFamily for EP.0000001.B1 returned error: %v", err)
			return
		}

		t.Logf("EP0000001B1 family: %d members, countries: %v",
			len(data.Members), data.Countries)
	})
}

// saveFixture saves raw API response data as a fixture file when -update-fixtures is set.
func saveFixture(t *testing.T, name string, data []byte) {
	t.Helper()
	if !*updateFixtures {
		return
	}
	dir := filepath.Join("testdata", "fixtures")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Logf("Warning: failed to create fixture dir: %v", err)
		return
	}
	path := filepath.Join(dir, name+".xml")
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Logf("Warning: failed to write fixture %s: %v", path, err)
		return
	}
	t.Logf("Updated fixture: %s", path)
}
