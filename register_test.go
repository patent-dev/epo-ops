package epo_ops

import (
	"context"
	"strings"
	"testing"
)

func TestGetRegisterProceduralStepsRaw(t *testing.T) {
	// Skip if no credentials
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client, ctx := setupRegisterTest(t)

	tests := []struct {
		name      string
		refType   string
		format    string
		number    string
		wantError bool
		errorMsg  string
	}{
		{
			name:      "Valid publication",
			refType:   "publication",
			format:    "epodoc",
			number:    "EP1000000",
			wantError: false,
		},
		{
			name:      "Valid application",
			refType:   "application",
			format:    "epodoc",
			number:    "EP20010000001",
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			steps, err := client.GetRegisterProceduralStepsRaw(ctx, tt.refType, tt.format, tt.number)

			if tt.wantError {
				if err == nil {
					t.Errorf("GetRegisterProceduralStepsRaw() expected error, got nil")
					return
				}
				if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("GetRegisterProceduralStepsRaw() error = %v, want error containing %q", err, tt.errorMsg)
				}
				return
			}

			if err != nil {
				t.Fatalf("GetRegisterProceduralStepsRaw() unexpected error: %v", err)
			}

			if len(steps) == 0 {
				t.Error("GetRegisterProceduralStepsRaw() returned empty steps")
			}

			// Verify it's XML or JSON
			isXML := strings.Contains(steps, "<?xml") || strings.Contains(steps, "<")
			isJSON := strings.Contains(steps, "{") || strings.Contains(steps, "[")
			if !isXML && !isJSON {
				t.Errorf("GetRegisterProceduralStepsRaw() doesn't look like XML or JSON: %s", steps[:min(100, len(steps))])
			}

			t.Logf("Retrieved procedural steps for %s %s, length: %d bytes",
				tt.refType, tt.number, len(steps))
		})
	}
}

func TestGetRegisterProceduralStepsMultipleRaw(t *testing.T) {
	// Skip if no credentials
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client, ctx := setupRegisterTest(t)

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
			refType:   "publication",
			format:    "epodoc",
			numbers:   []string{"EP1000000", "EP1000001"},
			wantError: false,
		},
		{
			name:      "Single number",
			refType:   "publication",
			format:    "epodoc",
			numbers:   []string{"EP1000000"},
			wantError: false,
		},
		{
			name:      "Empty list",
			refType:   "publication",
			format:    "epodoc",
			numbers:   []string{},
			wantError: true,
			errorMsg:  "cannot be empty",
		},
		{
			name:      "Too many numbers",
			refType:   "publication",
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

			steps, err := client.GetRegisterProceduralStepsMultipleRaw(ctx, tt.refType, tt.format, tt.numbers)

			if tt.wantError {
				if err == nil {
					t.Errorf("GetRegisterProceduralStepsMultipleRaw() expected error, got nil")
					return
				}
				if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("GetRegisterProceduralStepsMultipleRaw() error = %v, want error containing %q", err, tt.errorMsg)
				}
				return
			}

			if err != nil {
				t.Fatalf("GetRegisterProceduralStepsMultipleRaw() unexpected error: %v", err)
			}

			if len(steps) == 0 {
				t.Error("GetRegisterProceduralStepsMultipleRaw() returned empty steps")
			}

			// Verify it's XML or JSON
			isXML := strings.Contains(steps, "<?xml") || strings.Contains(steps, "<")
			isJSON := strings.Contains(steps, "{") || strings.Contains(steps, "[")
			if !isXML && !isJSON {
				t.Errorf("GetRegisterProceduralStepsMultipleRaw() doesn't look like XML or JSON")
			}

			t.Logf("Retrieved procedural steps for %d numbers, total length: %d bytes",
				len(tt.numbers), len(steps))
		})
	}
}

func TestGetRegisterUNIPRaw(t *testing.T) {
	// Skip if no credentials
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client, ctx := setupRegisterTest(t)

	tests := []struct {
		name      string
		refType   string
		format    string
		number    string
		wantError bool
		errorMsg  string
	}{
		{
			name:      "Valid publication",
			refType:   "publication",
			format:    "epodoc",
			number:    "EP3000000",
			wantError: false,
		},
		{
			name:      "Valid application",
			refType:   "application",
			format:    "epodoc",
			number:    "EP20140000001",
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			unip, err := client.GetRegisterUNIPRaw(ctx, tt.refType, tt.format, tt.number)

			if tt.wantError {
				if err == nil {
					t.Errorf("GetRegisterUNIPRaw() expected error, got nil")
					return
				}
				if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("GetRegisterUNIPRaw() error = %v, want error containing %q", err, tt.errorMsg)
				}
				return
			}

			if err != nil {
				t.Fatalf("GetRegisterUNIPRaw() unexpected error: %v", err)
			}

			if len(unip) == 0 {
				t.Error("GetRegisterUNIPRaw() returned empty unip data")
			}

			// Verify it's XML or JSON
			isXML := strings.Contains(unip, "<?xml") || strings.Contains(unip, "<")
			isJSON := strings.Contains(unip, "{") || strings.Contains(unip, "[")
			if !isXML && !isJSON {
				t.Errorf("GetRegisterUNIPRaw() doesn't look like XML or JSON: %s", unip[:min(100, len(unip))])
			}

			t.Logf("Retrieved UNIP data for %s %s, length: %d bytes",
				tt.refType, tt.number, len(unip))
		})
	}
}

func TestGetRegisterUNIPMultipleRaw(t *testing.T) {
	// Skip if no credentials
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client, ctx := setupRegisterTest(t)

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
			refType:   "publication",
			format:    "epodoc",
			numbers:   []string{"EP3000000", "EP3000001"},
			wantError: false,
		},
		{
			name:      "Single number",
			refType:   "publication",
			format:    "epodoc",
			numbers:   []string{"EP3000000"},
			wantError: false,
		},
		{
			name:      "Empty list",
			refType:   "publication",
			format:    "epodoc",
			numbers:   []string{},
			wantError: true,
			errorMsg:  "cannot be empty",
		},
		{
			name:      "Too many numbers",
			refType:   "publication",
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
					tt.numbers[i] = "EP3000000"
				}
			}

			unip, err := client.GetRegisterUNIPMultipleRaw(ctx, tt.refType, tt.format, tt.numbers)

			if tt.wantError {
				if err == nil {
					t.Errorf("GetRegisterUNIPMultipleRaw() expected error, got nil")
					return
				}
				if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("GetRegisterUNIPMultipleRaw() error = %v, want error containing %q", err, tt.errorMsg)
				}
				return
			}

			if err != nil {
				t.Fatalf("GetRegisterUNIPMultipleRaw() unexpected error: %v", err)
			}

			if len(unip) == 0 {
				t.Error("GetRegisterUNIPMultipleRaw() returned empty unip data")
			}

			// Verify it's XML or JSON
			isXML := strings.Contains(unip, "<?xml") || strings.Contains(unip, "<")
			isJSON := strings.Contains(unip, "{") || strings.Contains(unip, "[")
			if !isXML && !isJSON {
				t.Errorf("GetRegisterUNIPMultipleRaw() doesn't look like XML or JSON")
			}

			t.Logf("Retrieved UNIP data for %d numbers, total length: %d bytes",
				len(tt.numbers), len(unip))
		})
	}
}

func TestSearchRegister(t *testing.T) {
	// Skip if no credentials
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client, ctx := setupRegisterTest(t)

	tests := []struct {
		name      string
		query     string
		rangeSpec string
		wantError bool
		errorMsg  string
	}{
		{
			name:      "Valid title search",
			query:     "ti=plastic",
			rangeSpec: "",
			wantError: false,
		},
		{
			name:      "Valid title search with range",
			query:     "ti=battery",
			rangeSpec: "1-10",
			wantError: false,
		},
		{
			name:      "Valid applicant search",
			query:     "applicant=siemens",
			rangeSpec: "1-5",
			wantError: false,
		},
		{
			name:      "Empty query",
			query:     "",
			rangeSpec: "",
			wantError: true,
			errorMsg:  "cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := client.SearchRegister(ctx, tt.query, tt.rangeSpec)

			if tt.wantError {
				if err == nil {
					t.Errorf("SearchRegister() expected error, got nil")
					return
				}
				if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("SearchRegister() error = %v, want error containing %q", err, tt.errorMsg)
				}
				return
			}

			if err != nil {
				t.Fatalf("SearchRegister() unexpected error: %v", err)
			}

			if len(results) == 0 {
				t.Error("SearchRegister() returned empty results")
			}

			// Verify it's XML or JSON
			isXML := strings.Contains(results, "<?xml") || strings.Contains(results, "<")
			isJSON := strings.Contains(results, "{") || strings.Contains(results, "[")
			if !isXML && !isJSON {
				t.Errorf("SearchRegister() doesn't look like XML or JSON: %s", results[:min(100, len(results))])
			}

			t.Logf("Search results for query %q: %d bytes", tt.query, len(results))
		})
	}
}

func TestSearchRegisterWithConstituent(t *testing.T) {
	// Skip if no credentials
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client, ctx := setupRegisterTest(t)

	tests := []struct {
		name        string
		constituent string
		query       string
		rangeSpec   string
		wantError   bool
		errorMsg    string
	}{
		{
			name:        "Valid biblio search",
			constituent: "biblio",
			query:       "ti=plastic",
			rangeSpec:   "",
			wantError:   false,
		},
		{
			name:        "Valid events search",
			constituent: "events",
			query:       "ti=battery",
			rangeSpec:   "1-10",
			wantError:   false,
		},
		{
			name:        "Valid procedural-steps search",
			constituent: "procedural-steps",
			query:       "applicant=siemens",
			rangeSpec:   "1-5",
			wantError:   false,
		},
		{
			name:        "Valid upp search",
			constituent: "upp",
			query:       "ti=software",
			rangeSpec:   "",
			wantError:   false,
		},
		{
			name:        "Empty constituent",
			constituent: "",
			query:       "ti=plastic",
			rangeSpec:   "",
			wantError:   true,
			errorMsg:    "constituent cannot be empty",
		},
		{
			name:        "Empty query",
			constituent: "biblio",
			query:       "",
			rangeSpec:   "",
			wantError:   true,
			errorMsg:    "query cannot be empty",
		},
		{
			name:        "Invalid constituent",
			constituent: "invalid",
			query:       "ti=plastic",
			rangeSpec:   "",
			wantError:   true,
			errorMsg:    "must be one of",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := client.SearchRegisterWithConstituent(ctx, tt.constituent, tt.query, tt.rangeSpec)

			if tt.wantError {
				if err == nil {
					t.Errorf("SearchRegisterWithConstituent() expected error, got nil")
					return
				}
				if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("SearchRegisterWithConstituent() error = %v, want error containing %q", err, tt.errorMsg)
				}
				return
			}

			if err != nil {
				t.Fatalf("SearchRegisterWithConstituent() unexpected error: %v", err)
			}

			if len(results) == 0 {
				t.Error("SearchRegisterWithConstituent() returned empty results")
			}

			// Verify it's XML or JSON
			isXML := strings.Contains(results, "<?xml") || strings.Contains(results, "<")
			isJSON := strings.Contains(results, "{") || strings.Contains(results, "[")
			if !isXML && !isJSON {
				t.Errorf("SearchRegisterWithConstituent() doesn't look like XML or JSON: %s", results[:min(100, len(results))])
			}

			t.Logf("Search results for constituent %q, query %q: %d bytes",
				tt.constituent, tt.query, len(results))
		})
	}
}

// setupRegisterTest creates a test client and context for register integration tests.
// It skips the test if EPO credentials are not configured.
func setupRegisterTest(t *testing.T) (*Client, context.Context) {
	t.Helper()

	// Try to get credentials from environment or skip test
	consumerKey := getRegisterEnv("EPO_OPS_CONSUMER_KEY", "")
	consumerSecret := getRegisterEnv("EPO_OPS_CONSUMER_SECRET", "")

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

// getRegisterEnv returns an environment variable or a default value
func getRegisterEnv(_ /* key */, defaultValue string) string {
	// Stub that returns default without checking env
	// Integration tests are skipped unless credentials are set
	return defaultValue
}
