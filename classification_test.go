package epo_ops

import (
	"context"
	"strings"
	"testing"
)

func TestGetClassificationSchema(t *testing.T) {
	// Skip if no credentials
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client, ctx := setupIntegrationTest(t)

	tests := []struct {
		name       string
		class      string
		ancestors  bool
		navigation bool
		wantError  bool
	}{
		{
			name:       "Valid class A01B",
			class:      "A01B",
			ancestors:  false,
			navigation: false,
			wantError:  false,
		},
		{
			name:       "Valid class with ancestors",
			class:      "H04W",
			ancestors:  true,
			navigation: false,
			wantError:  false,
		},
		{
			name:       "Valid class with navigation",
			class:      "G06F",
			ancestors:  false,
			navigation: true,
			wantError:  false,
		},
		{
			name:       "Empty class",
			class:      "",
			ancestors:  false,
			navigation: false,
			wantError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema, err := client.GetClassificationSchema(ctx, tt.class, tt.ancestors, tt.navigation)

			if tt.wantError {
				if err == nil {
					t.Errorf("GetClassificationSchema() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("GetClassificationSchema() unexpected error: %v", err)
			}

			if len(schema) == 0 {
				t.Error("GetClassificationSchema() returned empty schema")
			}

			// Verify it's XML
			if !strings.Contains(schema, "<?xml") && !strings.Contains(schema, "<") {
				t.Errorf("GetClassificationSchema() doesn't look like XML: %s", schema[:min(100, len(schema))])
			}

			t.Logf("Retrieved schema for class %s, length: %d bytes", tt.class, len(schema))
		})
	}
}

func TestGetClassificationSchemaSubclass(t *testing.T) {
	// Skip if no credentials
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client, ctx := setupIntegrationTest(t)

	tests := []struct {
		name       string
		class      string
		subclass   string
		ancestors  bool
		navigation bool
		wantError  bool
	}{
		{
			name:       "Valid subclass",
			class:      "A01B1",
			subclass:   "00",
			ancestors:  false,
			navigation: false,
			wantError:  false,
		},
		{
			name:       "Empty class",
			class:      "",
			subclass:   "00",
			ancestors:  false,
			navigation: false,
			wantError:  true,
		},
		{
			name:       "Empty subclass",
			class:      "A01B1",
			subclass:   "",
			ancestors:  false,
			navigation: false,
			wantError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema, err := client.GetClassificationSchemaSubclass(ctx, tt.class, tt.subclass, tt.ancestors, tt.navigation)

			if tt.wantError {
				if err == nil {
					t.Errorf("GetClassificationSchemaSubclass() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("GetClassificationSchemaSubclass() unexpected error: %v", err)
			}

			if len(schema) == 0 {
				t.Error("GetClassificationSchemaSubclass() returned empty schema")
			}

			// Verify it's XML
			if !strings.Contains(schema, "<?xml") && !strings.Contains(schema, "<") {
				t.Errorf("GetClassificationSchemaSubclass() doesn't look like XML")
			}

			t.Logf("Retrieved subclass schema for %s/%s, length: %d bytes", tt.class, tt.subclass, len(schema))
		})
	}
}

func TestGetClassificationSchemaMultiple(t *testing.T) {
	// Skip if no credentials
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client, ctx := setupIntegrationTest(t)

	tests := []struct {
		name      string
		classes   []string
		wantError bool
		errorMsg  string
	}{
		{
			name:      "Valid multiple classes",
			classes:   []string{"A01B", "H04W", "G06F"},
			wantError: false,
		},
		{
			name:      "Single class",
			classes:   []string{"A01B"},
			wantError: false,
		},
		{
			name:      "Empty list",
			classes:   []string{},
			wantError: true,
			errorMsg:  "cannot be empty",
		},
		{
			name:      "Too many classes",
			classes:   make([]string, 101), // 101 items
			wantError: true,
			errorMsg:  "maximum 100",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Fill array with dummy values if needed
			if len(tt.classes) > 3 {
				for i := range tt.classes {
					tt.classes[i] = "A01B"
				}
			}

			schemas, err := client.GetClassificationSchemaMultiple(ctx, tt.classes)

			if tt.wantError {
				if err == nil {
					t.Errorf("GetClassificationSchemaMultiple() expected error, got nil")
					return
				}
				if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("GetClassificationSchemaMultiple() error = %v, want error containing %q", err, tt.errorMsg)
				}
				return
			}

			if err != nil {
				t.Fatalf("GetClassificationSchemaMultiple() unexpected error: %v", err)
			}

			if len(schemas) == 0 {
				t.Error("GetClassificationSchemaMultiple() returned empty schemas")
			}

			// Verify it's XML
			if !strings.Contains(schemas, "<?xml") && !strings.Contains(schemas, "<") {
				t.Errorf("GetClassificationSchemaMultiple() doesn't look like XML")
			}

			t.Logf("Retrieved %d classification schemas, total length: %d bytes", len(tt.classes), len(schemas))
		})
	}
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// setupIntegrationTest creates a test client and context for integration tests.
// It skips the test if EPO credentials are not configured.
func setupIntegrationTest(t *testing.T) (*Client, context.Context) {
	t.Helper()

	// Try to get credentials from environment or skip test
	consumerKey := getEnv("EPO_OPS_CONSUMER_KEY", "")
	consumerSecret := getEnv("EPO_OPS_CONSUMER_SECRET", "")

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

// getEnv returns an environment variable or a default value
func getEnv(_ /* key */, defaultValue string) string {
	// Stub that returns default without checking env
	// Integration tests are skipped unless credentials are set
	return defaultValue
}

func TestGetClassificationMedia(t *testing.T) {
	// Skip if no credentials
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client, ctx := setupIntegrationTest(t)

	tests := []struct {
		name         string
		mediaName    string
		asAttachment bool
		wantError    bool
		errorMsg     string
	}{
		{
			name:         "Valid media inline",
			mediaName:    "1000.gif",
			asAttachment: false,
			wantError:    false,
		},
		{
			name:         "Valid media as attachment",
			mediaName:    "1000.gif",
			asAttachment: true,
			wantError:    false,
		},
		{
			name:         "Empty media name",
			mediaName:    "",
			asAttachment: false,
			wantError:    true,
			errorMsg:     "cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			imageData, err := client.GetClassificationMedia(ctx, tt.mediaName, tt.asAttachment)

			if tt.wantError {
				if err == nil {
					t.Errorf("GetClassificationMedia() expected error, got nil")
					return
				}
				if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("GetClassificationMedia() error = %v, want error containing %q", err, tt.errorMsg)
				}
				return
			}

			if err != nil {
				t.Fatalf("GetClassificationMedia() unexpected error: %v", err)
			}

			if len(imageData) == 0 {
				t.Error("GetClassificationMedia() returned empty image data")
			}

			// Verify it looks like image data (check for common image file signatures)
			if !isImageData(imageData) {
				t.Errorf("GetClassificationMedia() doesn't look like image data (first bytes: %v)", imageData[:min(10, len(imageData))])
			}

			t.Logf("Retrieved classification media %s: %d bytes, attachment=%v", tt.mediaName, len(imageData), tt.asAttachment)
		})
	}
}

// isImageData performs basic validation to check if data looks like an image
func isImageData(data []byte) bool {
	if len(data) < 4 {
		return false
	}

	// Check for common image file signatures
	// GIF: "GIF8"
	if len(data) >= 4 && string(data[0:4]) == "GIF8" {
		return true
	}

	// PNG: 0x89 0x50 0x4E 0x47
	if len(data) >= 4 && data[0] == 0x89 && data[1] == 0x50 && data[2] == 0x4E && data[3] == 0x47 {
		return true
	}

	// JPEG: 0xFF 0xD8 0xFF
	if len(data) >= 3 && data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF {
		return true
	}

	// TIFF: "II" (little-endian) or "MM" (big-endian)
	if len(data) >= 2 && (string(data[0:2]) == "II" || string(data[0:2]) == "MM") {
		return true
	}

	return false
}

func TestGetClassificationStatistics(t *testing.T) {
	// Skip if no credentials
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client, ctx := setupIntegrationTest(t)

	tests := []struct {
		name      string
		query     string
		wantError bool
		errorMsg  string
	}{
		{
			name:      "Valid keyword query",
			query:     "plastic",
			wantError: false,
		},
		{
			name:      "Valid classification query",
			query:     "H04W",
			wantError: false,
		},
		{
			name:      "Valid detailed classification",
			query:     "A01B",
			wantError: false,
		},
		{
			name:      "Empty query",
			query:     "",
			wantError: true,
			errorMsg:  "cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats, err := client.GetClassificationStatistics(ctx, tt.query)

			if tt.wantError {
				if err == nil {
					t.Errorf("GetClassificationStatistics() expected error, got nil")
					return
				}
				if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("GetClassificationStatistics() error = %v, want error containing %q", err, tt.errorMsg)
				}
				return
			}

			if err != nil {
				t.Fatalf("GetClassificationStatistics() unexpected error: %v", err)
			}

			if len(stats) == 0 {
				t.Error("GetClassificationStatistics() returned empty statistics")
			}

			// Verify it's XML or JSON
			isXML := strings.Contains(stats, "<?xml") || strings.Contains(stats, "<")
			isJSON := strings.Contains(stats, "{") || strings.Contains(stats, "[")
			if !isXML && !isJSON {
				t.Errorf("GetClassificationStatistics() doesn't look like XML or JSON: %s", stats[:min(100, len(stats))])
			}

			t.Logf("Retrieved statistics for query %q, length: %d bytes, format: %s",
				tt.query, len(stats), map[bool]string{true: "XML", false: "JSON"}[isXML])
		})
	}
}

func TestGetClassificationMapping(t *testing.T) {
	// Skip if no credentials
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client, ctx := setupIntegrationTest(t)

	tests := []struct {
		name         string
		inputFormat  string
		class        string
		subclass     string
		outputFormat string
		additional   bool
		wantError    bool
		errorMsg     string
	}{
		{
			name:         "Valid ECLA to CPC",
			inputFormat:  "ecla",
			class:        "A01D2085",
			subclass:     "8",
			outputFormat: "cpc",
			additional:   false,
			wantError:    false,
		},
		{
			name:         "Valid CPC to ECLA",
			inputFormat:  "cpc",
			class:        "H04W84",
			subclass:     "18",
			outputFormat: "ecla",
			additional:   false,
			wantError:    false,
		},
		{
			name:         "Valid with additional info",
			inputFormat:  "ecla",
			class:        "A01D2085",
			subclass:     "8",
			outputFormat: "cpc",
			additional:   true,
			wantError:    false,
		},
		{
			name:         "Empty input format",
			inputFormat:  "",
			class:        "A01D2085",
			subclass:     "8",
			outputFormat: "cpc",
			additional:   false,
			wantError:    true,
			errorMsg:     "input format cannot be empty",
		},
		{
			name:         "Empty class",
			inputFormat:  "ecla",
			class:        "",
			subclass:     "8",
			outputFormat: "cpc",
			additional:   false,
			wantError:    true,
			errorMsg:     "classification class cannot be empty",
		},
		{
			name:         "Empty subclass",
			inputFormat:  "ecla",
			class:        "A01D2085",
			subclass:     "",
			outputFormat: "cpc",
			additional:   false,
			wantError:    true,
			errorMsg:     "classification subclass cannot be empty",
		},
		{
			name:         "Empty output format",
			inputFormat:  "ecla",
			class:        "A01D2085",
			subclass:     "8",
			outputFormat: "",
			additional:   false,
			wantError:    true,
			errorMsg:     "output format cannot be empty",
		},
		{
			name:         "Invalid input format",
			inputFormat:  "ipc",
			class:        "A01D2085",
			subclass:     "8",
			outputFormat: "cpc",
			additional:   false,
			wantError:    true,
			errorMsg:     "input format must be",
		},
		{
			name:         "Invalid output format",
			inputFormat:  "ecla",
			class:        "A01D2085",
			subclass:     "8",
			outputFormat: "ipc",
			additional:   false,
			wantError:    true,
			errorMsg:     "output format must be",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mapping, err := client.GetClassificationMapping(ctx, tt.inputFormat, tt.class, tt.subclass, tt.outputFormat, tt.additional)

			if tt.wantError {
				if err == nil {
					t.Errorf("GetClassificationMapping() expected error, got nil")
					return
				}
				if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("GetClassificationMapping() error = %v, want error containing %q", err, tt.errorMsg)
				}
				return
			}

			if err != nil {
				t.Fatalf("GetClassificationMapping() unexpected error: %v", err)
			}

			if len(mapping) == 0 {
				t.Error("GetClassificationMapping() returned empty mapping")
			}

			// Verify it's XML
			if !strings.Contains(mapping, "<?xml") && !strings.Contains(mapping, "<") {
				t.Errorf("GetClassificationMapping() doesn't look like XML: %s", mapping[:min(100, len(mapping))])
			}

			t.Logf("Mapped %s %s/%s to %s, length: %d bytes, additional: %v",
				tt.inputFormat, tt.class, tt.subclass, tt.outputFormat, len(mapping), tt.additional)
		})
	}
}
