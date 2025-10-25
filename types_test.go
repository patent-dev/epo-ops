package epo_ops

import "testing"

func TestParsePatentNumber(t *testing.T) {
	tests := []struct {
		input    string
		expected PatentNumber
		name     string
	}{
		{
			name:  "EP patent with A1 kind code",
			input: "EP2400812A1",
			expected: PatentNumber{
				Country: "EP",
				Number:  "2400812",
				Kind:    "A1",
			},
		},
		{
			name:  "EP patent with B1 kind code",
			input: "EP1000000B1",
			expected: PatentNumber{
				Country: "EP",
				Number:  "1000000",
				Kind:    "B1",
			},
		},
		{
			name:  "DE patent without kind code - INVALID",
			input: "DE123",
			expected: PatentNumber{
				Country: "",
				Number:  "",
				Kind:    "",
			},
		},
		{
			name:  "DE patent with single char kind code",
			input: "DE123C",
			expected: PatentNumber{
				Country: "DE",
				Number:  "123",
				Kind:    "C",
			},
		},
		{
			name:  "DE patent with two char kind code",
			input: "DE123C1",
			expected: PatentNumber{
				Country: "DE",
				Number:  "123",
				Kind:    "C1",
			},
		},
		{
			name:  "No country code - only number and kind",
			input: "123C",
			expected: PatentNumber{
				Country: "",
				Number:  "",
				Kind:    "",
			},
		},
		{
			name:  "Country code too short",
			input: "D123C",
			expected: PatentNumber{
				Country: "",
				Number:  "",
				Kind:    "",
			},
		},
		{
			name:  "No number portion",
			input: "DEC",
			expected: PatentNumber{
				Country: "",
				Number:  "",
				Kind:    "",
			},
		},
		{
			name:  "US patent with letter in number portion",
			input: "USD123456S1",
			expected: PatentNumber{
				Country: "US",
				Number:  "D123456",
				Kind:    "S1",
			},
		},
		{
			name:  "WO patent",
			input: "WO2020123456A1",
			expected: PatentNumber{
				Country: "WO",
				Number:  "2020123456",
				Kind:    "A1",
			},
		},
		{
			name:  "Too short - just country",
			input: "EP",
			expected: PatentNumber{
				Country: "",
				Number:  "",
				Kind:    "",
			},
		},
		{
			name:  "Lowercase country code",
			input: "ep2400812a1",
			expected: PatentNumber{
				Country: "ep",
				Number:  "2400812",
				Kind:    "a1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParsePatentNumber(tt.input)
			if result.Country != tt.expected.Country {
				t.Errorf("Country: got %q, want %q", result.Country, tt.expected.Country)
			}
			if result.Number != tt.expected.Number {
				t.Errorf("Number: got %q, want %q", result.Number, tt.expected.Number)
			}
			if result.Kind != tt.expected.Kind {
				t.Errorf("Kind: got %q, want %q", result.Kind, tt.expected.Kind)
			}
		})
	}
}
