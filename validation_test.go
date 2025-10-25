package epo_ops

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestValidateDocdbFormat(t *testing.T) {
	tests := []struct {
		name      string
		number    string
		wantError bool
	}{
		{"Valid EP patent", "EP.1000000.B1", false},
		{"Valid US patent", "US.5551212.A", false},
		{"Valid WO patent", "WO.2023123456.A1", false},
		{"Valid with single digit kind", "DE.102023001.A", false},
		{"Empty number", "", true},
		{"Missing dots", "EP1000000B1", true},
		{"Missing kind code", "EP.1000000.", true},
		{"Lowercase country", "ep.1000000.B1", true},
		{"Invalid kind code", "EP.1000000.B12", true},
		{"No country code", ".1000000.B1", true},
		{"Letters in number", "EP.100A000.B1", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDocdbFormat(tt.number)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateDocdbFormat(%q) error = %v, wantError %v", tt.number, err, tt.wantError)
			}

			// Check that error is ValidationError type
			if err != nil {
				var valErr *ValidationError
				if !errors.As(err, &valErr) {
					t.Errorf("Expected ValidationError, got %T", err)
				}
				if valErr.Format != "docdb" {
					t.Errorf("Expected format 'docdb', got %q", valErr.Format)
				}
			}
		})
	}
}

func TestValidateEpodocFormat(t *testing.T) {
	tests := []struct {
		name      string
		number    string
		wantError bool
	}{
		{"Valid with kind code", "EP1000000B1", false},
		{"Valid without kind code", "EP1000000", false},
		{"Valid US patent", "US5551212A", false},
		{"Valid WO patent", "WO2023123456A1", false},
		{"Valid single char kind", "DE102023001A", false},
		{"Empty number", "", true},
		{"With dots", "EP.1000000.B1", true},
		{"Lowercase country", "ep1000000B1", true},
		{"No country code", "1000000B1", true},
		{"Letters in number", "EP100A000B1", true},
		{"Invalid kind code", "EP1000000B12", true},
		{"Single letter country", "E1000000B1", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEpodocFormat(tt.number)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateEpodocFormat(%q) error = %v, wantError %v", tt.number, err, tt.wantError)
			}

			// Check that error is ValidationError type
			if err != nil {
				var valErr *ValidationError
				if !errors.As(err, &valErr) {
					t.Errorf("Expected ValidationError, got %T", err)
				}
				if valErr.Format != "epodoc" {
					t.Errorf("Expected format 'epodoc', got %q", valErr.Format)
				}
			}
		})
	}
}

func TestValidateOriginalFormat(t *testing.T) {
	tests := []struct {
		name      string
		number    string
		wantError bool
	}{
		{"Standard format", "EP 1000000 B1", false},
		{"With commas", "US5,551,212", false},
		{"With slashes", "WO/2023/123456", false},
		{"Mixed format", "EP-1000000-B1", false},
		{"Very flexible", "Patent#123-456/789", false},
		{"Empty number", "", true},
		{"Too long", string(make([]byte, 101)), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateOriginalFormat(tt.number)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateOriginalFormat(%q) error = %v, wantError %v", tt.number, err, tt.wantError)
			}

			if err != nil {
				var valErr *ValidationError
				if !errors.As(err, &valErr) {
					t.Errorf("Expected ValidationError, got %T", err)
				}
			}
		})
	}
}

func TestValidateFormat(t *testing.T) {
	tests := []struct {
		name      string
		format    string
		number    string
		wantError bool
	}{
		{"Docdb format valid", "docdb", "EP.1000000.B1", false},
		{"Epodoc format valid", "epodoc", "EP1000000B1", false},
		{"Original format valid", "original", "EP 1000000 B1", false},
		{"Using constant FormatDocDB", FormatDocDB, "EP.1000000.B1", false},
		{"Using constant FormatEPODOC", FormatEPODOC, "EP1000000B1", false},
		{"Using constant FormatOriginal", FormatOriginal, "EP 1000000 B1", false},
		{"Invalid format", "invalid", "EP1000000B1", true},
		{"Wrong format for number", "docdb", "EP1000000B1", true},
		{"Empty number", "docdb", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFormat(tt.format, tt.number)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateFormat(%q, %q) error = %v, wantError %v", tt.format, tt.number, err, tt.wantError)
			}

			if err != nil {
				var valErr *ValidationError
				if !errors.As(err, &valErr) {
					t.Errorf("Expected ValidationError, got %T", err)
				}
			}
		})
	}
}

func TestValidateDate(t *testing.T) {
	tests := []struct {
		name      string
		date      string
		wantError bool
	}{
		{"Valid date", "20231015", false},
		{"Valid date 1999", "19990101", false},
		{"Valid date 2025", "20251231", false},
		{"Empty date (optional)", "", false},
		{"Too short", "2023101", true},
		{"Too long", "202310155", true},
		{"Letters in date", "2023AB15", true},
		{"With dashes", "2023-10-15", true},
		{"With slashes", "2023/10/15", true},
		{"Invalid month (format check only)", "20231399", false}, // Format is valid, semantic check not done
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDate(tt.date)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateDate(%q) error = %v, wantError %v", tt.date, err, tt.wantError)
			}

			if err != nil {
				var valErr *ValidationError
				if !errors.As(err, &valErr) {
					t.Errorf("Expected ValidationError, got %T", err)
				}
				if valErr.Field != "date" {
					t.Errorf("Expected field 'date', got %q", valErr.Field)
				}
			}
		})
	}
}

func TestValidateRefType(t *testing.T) {
	tests := []struct {
		name      string
		refType   string
		wantError bool
	}{
		{"Valid publication", "publication", false},
		{"Valid application", "application", false},
		{"Valid priority", "priority", false},
		{"Using constant RefTypePublication", RefTypePublication, false},
		{"Using constant RefTypeApplication", RefTypeApplication, false},
		{"Using constant RefTypePriority", RefTypePriority, false},
		{"Invalid type", "invalid", true},
		{"Empty string", "", true},
		{"Uppercase", "PUBLICATION", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRefType(tt.refType)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateRefType(%q) error = %v, wantError %v", tt.refType, err, tt.wantError)
			}

			if err != nil {
				var valErr *ValidationError
				if !errors.As(err, &valErr) {
					t.Errorf("Expected ValidationError, got %T", err)
				}
				if valErr.Field != "refType" {
					t.Errorf("Expected field 'refType', got %q", valErr.Field)
				}
			}
		})
	}
}

func TestValidationError_Error(t *testing.T) {
	tests := []struct {
		name     string
		valErr   *ValidationError
		expected string
	}{
		{
			name: "With format",
			valErr: &ValidationError{
				Field:   "number",
				Format:  "docdb",
				Value:   "EP1000000B1",
				Message: "must match pattern",
			},
			expected: `validation error: number (docdb format): must match pattern - got: "EP1000000B1"`,
		},
		{
			name: "Without format",
			valErr: &ValidationError{
				Field:   "refType",
				Value:   "invalid",
				Message: "must be publication, application, or priority",
			},
			expected: `validation error: refType: must be publication, application, or priority - got: "invalid"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.valErr.Error()
			if got != tt.expected {
				t.Errorf("Error() = %q, want %q", got, tt.expected)
			}
		})
	}
}

// Integration test: Verify validation happens before API call
func TestValidation_PreventsBadAPICall(t *testing.T) {
	// This test verifies that validation errors are returned
	// BEFORE making any HTTP request to the API

	config := DefaultConfig()
	config.ConsumerKey = "test"
	config.ConsumerSecret = "test"

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Test with invalid docdb format (should fail validation before API call)
	_, err = client.GetBiblioRaw(context.TODO(), RefTypePublication, FormatDocDB, "EP1000000B1") // Missing dots

	if err == nil {
		t.Fatal("Expected validation error, got nil")
	}

	// Should be a ValidationError, not an API error
	var valErr *ValidationError
	if !errors.As(err, &valErr) {
		t.Errorf("Expected ValidationError before API call, got %T: %v", err, err)
	}

	// Verify it's the format validation that failed
	if valErr != nil && valErr.Format != "docdb" {
		t.Errorf("Expected docdb format validation error, got format: %q", valErr.Format)
	}
}

func TestNormalizeToDocdb(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		want      string
		wantError bool
		errorMsg  string // Partial match for error message
	}{
		// Valid EPODOC format conversions
		{
			name:  "EPODOC EP format",
			input: "EP2884620A2",
			want:  "EP.2884620.A2",
		},
		{
			name:  "EPODOC US format",
			input: "US5551212A",
			want:  "US.5551212.A",
		},
		{
			name:  "EPODOC WO format",
			input: "WO2023123456A1",
			want:  "WO.2023123456.A1",
		},
		{
			name:  "EPODOC DE format single char kind",
			input: "DE123C",
			want:  "DE.123.C",
		},
		{
			name:  "EPODOC EP with B1 kind",
			input: "EP1000000B1",
			want:  "EP.1000000.B1",
		},
		{
			name:  "EPODOC with complex number",
			input: "JP2023567890A",
			want:  "JP.2023567890.A",
		},

		// Already valid DOCDB format (should return unchanged)
		{
			name:  "Already DOCDB EP",
			input: "EP.2884620.A2",
			want:  "EP.2884620.A2",
		},
		{
			name:  "Already DOCDB US",
			input: "US.5551212.A",
			want:  "US.5551212.A",
		},
		{
			name:  "Already DOCDB WO",
			input: "WO.2023123456.A1",
			want:  "WO.2023123456.A1",
		},

		// With whitespace (should be cleaned)
		{
			name:  "With spaces",
			input: "EP 2884620 A2",
			want:  "EP.2884620.A2",
		},
		{
			name:  "With leading/trailing spaces",
			input: "  EP2884620A2  ",
			want:  "EP.2884620.A2",
		},
		{
			name:  "DOCDB with spaces",
			input: "EP .2884620 .A2",
			want:  "EP.2884620.A2",
		},

		// With hyphens (should be removed)
		{
			name:  "With hyphens",
			input: "EP-2884620-A2",
			want:  "EP.2884620.A2",
		},

		// With slashes (should be removed)
		{
			name:  "With slashes",
			input: "EP/2884620/A2",
			want:  "EP.2884620.A2",
		},

		// Mixed separators
		{
			name:  "Mixed separators",
			input: "EP 2884620-A2",
			want:  "EP.2884620.A2",
		},

		// Error cases
		{
			name:      "Empty string",
			input:     "",
			wantError: true,
			errorMsg:  "cannot be empty",
		},
		{
			name:      "Only whitespace",
			input:     "   ",
			wantError: true,
			errorMsg:  "only whitespace",
		},
		{
			name:      "Only separators",
			input:     "---",
			wantError: true,
			errorMsg:  "only whitespace",
		},
		{
			name:      "Missing country code",
			input:     "2884620A2",
			wantError: true,
			errorMsg:  "unable to parse",
		},
		{
			name:      "Single char country",
			input:     "E2884620A2",
			wantError: true,
			errorMsg:  "unable to parse",
		},
		{
			name:      "Missing kind code",
			input:     "EP2884620",
			wantError: true,
			errorMsg:  "unable to parse",
		},
		{
			name:      "Invalid DOCDB format",
			input:     "EP.2884620.A23", // Kind code too long
			wantError: true,
			errorMsg:  "invalid DOCDB format",
		},
		{
			name:      "Lowercase country code",
			input:     "ep2884620A2",
			wantError: true,
			errorMsg:  "converted format is invalid",
		},
		{
			name:      "Letters in number",
			input:     "EP28A4620A2",
			wantError: true,
			errorMsg:  "converted format is invalid",
		},
		{
			name:      "Invalid kind code format",
			input:     "EP2884620A23", // Kind code too long
			wantError: true,
			errorMsg:  "unable to parse",
		},
		{
			name:      "Only country code",
			input:     "EP",
			wantError: true,
			errorMsg:  "unable to parse",
		},
		{
			name:      "Missing number portion",
			input:     "EPA2",
			wantError: true,
			errorMsg:  "unable to parse",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeToDocdb(tt.input)

			// Check error expectation
			if (err != nil) != tt.wantError {
				t.Errorf("NormalizeToDocdb(%q) error = %v, wantError %v", tt.input, err, tt.wantError)
				return
			}

			// If error expected, check error message
			if tt.wantError {
				if err == nil {
					t.Errorf("NormalizeToDocdb(%q) expected error containing %q, got nil", tt.input, tt.errorMsg)
					return
				}
				// Check that error is ValidationError type
				var valErr *ValidationError
				if !errors.As(err, &valErr) {
					t.Errorf("Expected ValidationError, got %T", err)
				}
				// Check error message contains expected substring
				if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("NormalizeToDocdb(%q) error = %q, want error containing %q", tt.input, err.Error(), tt.errorMsg)
				}
				return
			}

			// Check result
			if got != tt.want {
				t.Errorf("NormalizeToDocdb(%q) = %q, want %q", tt.input, got, tt.want)
			}

			// Verify result is valid DOCDB format
			if err := ValidateDocdbFormat(got); err != nil {
				t.Errorf("NormalizeToDocdb(%q) produced invalid DOCDB format %q: %v", tt.input, got, err)
			}
		})
	}
}

// TestNormalizeToDocdb_Idempotent verifies that normalizing an already-normalized number returns the same value
func TestNormalizeToDocdb_Idempotent(t *testing.T) {
	inputs := []string{
		"EP2884620A2",
		"US5551212A",
		"WO2023123456A1",
		"EP 1000000 B1",
		"DE-123-C",
	}

	for _, input := range inputs {
		t.Run(input, func(t *testing.T) {
			// First normalization
			first, err := NormalizeToDocdb(input)
			if err != nil {
				t.Fatalf("First normalization failed: %v", err)
			}

			// Second normalization (should be idempotent)
			second, err := NormalizeToDocdb(first)
			if err != nil {
				t.Fatalf("Second normalization failed: %v", err)
			}

			if first != second {
				t.Errorf("Not idempotent: first=%q, second=%q", first, second)
			}
		})
	}
}

// TestNormalizeToDocdb_EdgeCases tests various edge cases
func TestNormalizeToDocdb_EdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantError bool
	}{
		{"Very long number", "EP123456789012345678901234567890A1", false},
		{"Single digit number", "EP1A", false},
		{"Two digit number", "EP12A", false},
		{"Three letter country (should fail)", "EUR1000000A1", true},
		{"Numeric country code (should fail)", "12345678A1", true},
		{"Special characters in country", "E@1000000A1", true},
		{"Unicode characters", "EP\u20001000000A1", true},
		{"Tab character", "EP\t1000000\tA1", false},    // Should be cleaned like space
		{"Newline character", "EP\n1000000\nA1", true}, // Not cleaned, should fail
		{"Kind code without number", "EPA", true},
		{"Multiple dots already", "EP..1000000..A1", true},
		{"Dot at wrong position", "E.P1000000A1", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NormalizeToDocdb(tt.input)
			if (err != nil) != tt.wantError {
				t.Errorf("NormalizeToDocdb(%q) error = %v, wantError %v", tt.input, err, tt.wantError)
			}
		})
	}
}
