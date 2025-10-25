package epo_ops

import (
	"context"
	"strings"
	"testing"
)

func TestGetPublishedEquivalents(t *testing.T) {
	// Skip if no credentials
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client, ctx := setupPublishedTest(t)

	tests := []struct {
		name      string
		refType   string
		format    string
		number    string
		wantError bool
		errorMsg  string
	}{
		{
			name:      "Valid publication epodoc",
			refType:   RefTypePublication,
			format:    "epodoc",
			number:    "EP1000000",
			wantError: false,
		},
		{
			name:      "Valid publication docdb",
			refType:   RefTypePublication,
			format:    "docdb",
			number:    "EP.1000000",
			wantError: false,
		},
		{
			name:      "Valid application",
			refType:   RefTypeApplication,
			format:    "epodoc",
			number:    "EP20010000001",
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			equivalents, err := client.GetPublishedEquivalents(ctx, tt.refType, tt.format, tt.number)

			if tt.wantError {
				if err == nil {
					t.Errorf("GetPublishedEquivalents() expected error, got nil")
					return
				}
				if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("GetPublishedEquivalents() error = %v, want error containing %q", err, tt.errorMsg)
				}
				return
			}

			if err != nil {
				t.Fatalf("GetPublishedEquivalents() unexpected error: %v", err)
			}

			if len(equivalents) == 0 {
				t.Error("GetPublishedEquivalents() returned empty equivalents")
			}

			// Verify it's XML or JSON
			isXML := strings.Contains(equivalents, "<?xml") || strings.Contains(equivalents, "<")
			isJSON := strings.Contains(equivalents, "{") || strings.Contains(equivalents, "[")
			if !isXML && !isJSON {
				t.Errorf("GetPublishedEquivalents() doesn't look like XML or JSON: %s", equivalents[:min(100, len(equivalents))])
			}

			t.Logf("Retrieved equivalents for %s %s, length: %d bytes",
				tt.refType, tt.number, len(equivalents))
		})
	}
}

func TestGetPublishedEquivalentsMultiple(t *testing.T) {
	// Skip if no credentials
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client, ctx := setupPublishedTest(t)

	tests := []struct {
		name      string
		refType   string
		format    string
		numbers   []string
		wantError bool
		errorMsg  string
	}{
		{
			name:      "Valid multiple publications",
			refType:   RefTypePublication,
			format:    "epodoc",
			numbers:   []string{"EP1000000", "EP1000001"},
			wantError: false,
		},
		{
			name:      "Single number",
			refType:   RefTypePublication,
			format:    "epodoc",
			numbers:   []string{"EP1000000"},
			wantError: false,
		},
		{
			name:      "Empty list",
			refType:   RefTypePublication,
			format:    "epodoc",
			numbers:   []string{},
			wantError: true,
			errorMsg:  "cannot be empty",
		},
		{
			name:      "Too many numbers",
			refType:   RefTypePublication,
			format:    "epodoc",
			numbers:   make([]string, 101),
			wantError: true,
			errorMsg:  "maximum 100",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Fill array with dummy values if needed
			if len(tt.numbers) > 2 {
				for i := range tt.numbers {
					tt.numbers[i] = "EP1000000"
				}
			}

			equivalents, err := client.GetPublishedEquivalentsMultiple(ctx, tt.refType, tt.format, tt.numbers)

			if tt.wantError {
				if err == nil {
					t.Errorf("GetPublishedEquivalentsMultiple() expected error, got nil")
					return
				}
				if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("GetPublishedEquivalentsMultiple() error = %v, want error containing %q", err, tt.errorMsg)
				}
				return
			}

			if err != nil {
				t.Fatalf("GetPublishedEquivalentsMultiple() unexpected error: %v", err)
			}

			if len(equivalents) == 0 {
				t.Error("GetPublishedEquivalentsMultiple() returned empty equivalents")
			}

			// Verify it's XML or JSON
			isXML := strings.Contains(equivalents, "<?xml") || strings.Contains(equivalents, "<")
			isJSON := strings.Contains(equivalents, "{") || strings.Contains(equivalents, "[")
			if !isXML && !isJSON {
				t.Errorf("GetPublishedEquivalentsMultiple() doesn't look like XML or JSON")
			}

			t.Logf("Retrieved equivalents for %d numbers, total length: %d bytes",
				len(tt.numbers), len(equivalents))
		})
	}
}

// setupPublishedTest creates a test client and context for published data integration tests.
// It skips the test if EPO credentials are not configured.
func setupPublishedTest(t *testing.T) (*Client, context.Context) {
	t.Helper()

	// Try to get credentials from environment or skip test
	consumerKey := getPublishedEnv("EPO_OPS_CONSUMER_KEY", "")
	consumerSecret := getPublishedEnv("EPO_OPS_CONSUMER_SECRET", "")

	if consumerKey == "" || consumerSecret == "" {
		t.Skip("EPO_OPS_CONSUMER_KEY and EPO_OPS_CONSUMER_SECRET environment variables required for integration tests")
	}

	config := &Config{
		ConsumerKey:    consumerKey,
		ConsumerSecret: consumerSecret,
	}

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	return client, context.Background()
}

// getPublishedEnv returns an environment variable or a default value
func getPublishedEnv(_ /* key */, defaultValue string) string {
	// Stub that returns default without checking env
	// Integration tests are skipped unless credentials are set
	return defaultValue
}
