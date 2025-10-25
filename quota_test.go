package epo_ops

import (
	"strings"
	"testing"
)

func TestValidateTimeRange(t *testing.T) {
	tests := []struct {
		name      string
		timeRange string
		wantError bool
		errorMsg  string
	}{
		// Valid single dates
		{
			name:      "Valid single date",
			timeRange: "01/01/2024",
			wantError: false,
		},
		{
			name:      "Valid single date - different month",
			timeRange: "15/06/2023",
			wantError: false,
		},
		{
			name:      "Valid single date - year boundary",
			timeRange: "31/12/2022",
			wantError: false,
		},

		// Valid date ranges
		{
			name:      "Valid date range",
			timeRange: "01/01/2024~07/01/2024",
			wantError: false,
		},
		{
			name:      "Valid date range - different months",
			timeRange: "01/01/2024~31/03/2024",
			wantError: false,
		},
		{
			name:      "Valid date range - year boundary",
			timeRange: "25/12/2023~05/01/2024",
			wantError: false,
		},

		// Invalid - empty
		{
			name:      "Empty time range",
			timeRange: "",
			wantError: true,
			errorMsg:  "cannot be empty",
		},

		// Invalid - format errors
		{
			name:      "Invalid separator - slash instead of tilde",
			timeRange: "01/01/2024/07/01/2024",
			wantError: true,
			errorMsg:  "10 characters", // Fails length check (23 chars)
		},
		{
			name:      "Invalid separator - dash instead of slash",
			timeRange: "01-01-2024",
			wantError: true,
			errorMsg:  "date must use slashes",
		},
		{
			name:      "Too short",
			timeRange: "1/1/2024",
			wantError: true,
			errorMsg:  "10 characters",
		},
		{
			name:      "Too long",
			timeRange: "01/01/20245",
			wantError: true,
			errorMsg:  "10 characters",
		},

		// Invalid - day errors
		{
			name:      "Invalid day - zero",
			timeRange: "00/01/2024",
			wantError: true,
			errorMsg:  "day must be between",
		},
		{
			name:      "Invalid day - too high",
			timeRange: "32/01/2024",
			wantError: true,
			errorMsg:  "day must be between",
		},

		// Invalid - month errors
		{
			name:      "Invalid month - zero",
			timeRange: "01/00/2024",
			wantError: true,
			errorMsg:  "month must be between",
		},
		{
			name:      "Invalid month - too high",
			timeRange: "01/13/2024",
			wantError: true,
			errorMsg:  "month must be between",
		},

		// Invalid - year errors
		{
			name:      "Invalid year - 3 digits",
			timeRange: "01/01/202",
			wantError: true,
			errorMsg:  "10 characters",
		},
		{
			name:      "Invalid year - too low",
			timeRange: "01/01/0999",
			wantError: true,
			errorMsg:  "4-digit number",
		},

		// Invalid - date range errors
		{
			name:      "Date range - too many tildes",
			timeRange: "01/01/2024~07/01/2024~15/01/2024",
			wantError: true,
			errorMsg:  "exactly one tilde",
		},
		{
			name:      "Date range - invalid start date",
			timeRange: "32/01/2024~07/01/2024",
			wantError: true,
			errorMsg:  "invalid start date",
		},
		{
			name:      "Date range - invalid end date",
			timeRange: "01/01/2024~32/01/2024",
			wantError: true,
			errorMsg:  "invalid end date",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTimeRange(tt.timeRange)

			if tt.wantError {
				if err == nil {
					t.Errorf("ValidateTimeRange(%q) expected error containing %q, got nil",
						tt.timeRange, tt.errorMsg)
					return
				}
				if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("ValidateTimeRange(%q) error = %v, want error containing %q",
						tt.timeRange, err, tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateTimeRange(%q) unexpected error: %v",
						tt.timeRange, err)
				}
			}
		})
	}
}

func TestParseUsageStats(t *testing.T) {
	tests := []struct {
		name        string
		jsonData    string
		timeRange   string
		wantEntries int
		wantError   bool
	}{
		{
			name: "Valid usage stats with multiple entries",
			jsonData: `{
				"data": [
					{
						"timestamp": 1640995200,
						"total_response_size": 1024000,
						"message_count": 150
					},
					{
						"timestamp": 1640998800,
						"total_response_size": 512000,
						"message_count": 75,
						"service": "biblio"
					}
				]
			}`,
			timeRange:   "01/01/2022",
			wantEntries: 2,
			wantError:   false,
		},
		{
			name: "Empty usage stats",
			jsonData: `{
				"data": []
			}`,
			timeRange:   "01/01/2024",
			wantEntries: 0,
			wantError:   false,
		},
		{
			name:        "Invalid JSON",
			jsonData:    `{"data": [invalid json]}`,
			timeRange:   "01/01/2024",
			wantEntries: 0,
			wantError:   true,
		},
		{
			name:        "Missing data field",
			jsonData:    `{"entries": []}`,
			timeRange:   "01/01/2024",
			wantEntries: 0,
			wantError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats, err := parseUsageStats(tt.jsonData, tt.timeRange)

			if tt.wantError {
				if err == nil {
					t.Errorf("parseUsageStats() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("parseUsageStats() unexpected error: %v", err)
			}

			if stats.TimeRange != tt.timeRange {
				t.Errorf("TimeRange = %q, want %q", stats.TimeRange, tt.timeRange)
			}

			if len(stats.Entries) != tt.wantEntries {
				t.Errorf("Entries count = %d, want %d", len(stats.Entries), tt.wantEntries)
			}

			// Validate first entry if present
			if tt.wantEntries > 0 && len(stats.Entries) > 0 {
				entry := stats.Entries[0]
				if entry.Timestamp <= 0 {
					t.Errorf("First entry Timestamp = %d, want > 0", entry.Timestamp)
				}
				if entry.TotalResponseSize < 0 {
					t.Errorf("First entry TotalResponseSize = %d, want >= 0", entry.TotalResponseSize)
				}
				if entry.MessageCount < 0 {
					t.Errorf("First entry MessageCount = %d, want >= 0", entry.MessageCount)
				}
			}

			// Validate second entry has service field if present
			if tt.wantEntries > 1 && len(stats.Entries) > 1 {
				entry := stats.Entries[1]
				if entry.Service == "" && strings.Contains(tt.jsonData, "service") {
					t.Errorf("Second entry Service is empty, but JSON contains service field")
				}
			}
		})
	}
}

func TestParseUsageStats_DetailedValidation(t *testing.T) {
	jsonData := `{
		"data": [
			{
				"timestamp": 1640995200,
				"total_response_size": 1024000,
				"message_count": 150,
				"service": "biblio"
			}
		]
	}`

	stats, err := parseUsageStats(jsonData, "01/01/2022")
	if err != nil {
		t.Fatalf("parseUsageStats() unexpected error: %v", err)
	}

	if len(stats.Entries) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(stats.Entries))
	}

	entry := stats.Entries[0]

	// Check timestamp
	if entry.Timestamp != 1640995200 {
		t.Errorf("Timestamp = %d, want 1640995200", entry.Timestamp)
	}

	// Check total response size
	if entry.TotalResponseSize != 1024000 {
		t.Errorf("TotalResponseSize = %d, want 1024000", entry.TotalResponseSize)
	}

	// Check message count
	if entry.MessageCount != 150 {
		t.Errorf("MessageCount = %d, want 150", entry.MessageCount)
	}

	// Check service
	if entry.Service != "biblio" {
		t.Errorf("Service = %q, want %q", entry.Service, "biblio")
	}
}
