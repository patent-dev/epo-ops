package epo_ops

import (
	"testing"
)

func TestDerivePatentStatus(t *testing.T) {
	tests := []struct {
		name          string
		events        []LegalEvent
		targetCountry string
		want          string
	}{
		{
			name:   "empty events returns unknown",
			events: nil,
			want:   StatusUnknown,
		},
		{
			name:          "no events for target country returns unknown",
			events:        []LegalEvent{{EventCode: "B1", Country: "EP", Date: "2020-01-01"}},
			targetCountry: "US",
			want:          StatusUnknown,
		},
		{
			name: "grant event means active",
			events: []LegalEvent{
				{EventCode: "AK", Country: "EP", Date: "2018-01-01"},
				{EventCode: "B1", Country: "EP", Date: "2020-06-15"},
			},
			targetCountry: "EP",
			want:          StatusActive,
		},
		{
			name: "PG25 lapse in specific country",
			events: []LegalEvent{
				{EventCode: "B1", Country: "EP", Date: "2020-01-01"},
				{EventCode: "PG25", Country: "DE", Date: "2022-03-01"},
			},
			targetCountry: "DE",
			want:          StatusLapsed,
		},
		{
			name: "PG25 in DE does not affect FR status",
			events: []LegalEvent{
				{EventCode: "B1", Country: "EP", Date: "2020-01-01"},
				{EventCode: "PG25", Country: "DE", Date: "2022-03-01"},
				{EventCode: "AK", Country: "FR", Date: "2018-01-01"},
			},
			targetCountry: "FR",
			want:          StatusActive,
		},
		{
			name: "withdrawn application",
			events: []LegalEvent{
				{EventCode: "17P", Country: "EP", Date: "2018-01-01"},
				{EventCode: "18W", Country: "EP", Date: "2019-06-01"},
			},
			targetCountry: "EP",
			want:          StatusWithdrawn,
		},
		{
			name: "revoked patent",
			events: []LegalEvent{
				{EventCode: "B1", Country: "EP", Date: "2018-01-01"},
				{EventCode: "27W", Country: "EP", Date: "2020-01-01"},
			},
			targetCountry: "EP",
			want:          StatusRevoked,
		},
		{
			name: "expired patent",
			events: []LegalEvent{
				{EventCode: "B1", Country: "EP", Date: "2000-01-01"},
				{EventCode: "MK05", Country: "EP", Date: "2020-01-01"},
			},
			targetCountry: "EP",
			want:          StatusExpired,
		},
		{
			name: "pending application",
			events: []LegalEvent{
				{EventCode: "AK", Country: "EP", Date: "2022-01-01"},
				{EventCode: "17P", Country: "EP", Date: "2022-06-01"},
			},
			targetCountry: "EP",
			want:          StatusPending,
		},
		{
			name: "no target country considers all events",
			events: []LegalEvent{
				{EventCode: "B1", Country: "EP", Date: "2020-01-01"},
				{EventCode: "PG25", Country: "DE", Date: "2022-03-01"},
			},
			targetCountry: "",
			want:          StatusLapsed,
		},
		{
			name: "most recent event wins",
			events: []LegalEvent{
				{EventCode: "18W", Country: "EP", Date: "2019-01-01"},
				{EventCode: "B1", Country: "EP", Date: "2020-01-01"},
			},
			targetCountry: "EP",
			want:          StatusActive,
		},
		{
			name: "uses EventCode over Code when both present",
			events: []LegalEvent{
				{Code: "AK  ", EventCode: "B1", Country: "EP", Date: "2020-01-01"},
			},
			targetCountry: "EP",
			want:          StatusActive,
		},
		{
			name: "falls back to Code when EventCode empty",
			events: []LegalEvent{
				{Code: "B1", EventCode: "", Country: "EP", Date: "2020-01-01"},
			},
			targetCountry: "EP",
			want:          StatusActive,
		},
		{
			name: "unknown codes return unknown",
			events: []LegalEvent{
				{EventCode: "ZZZZZ", Country: "EP", Date: "2020-01-01"},
			},
			targetCountry: "EP",
			want:          StatusUnknown,
		},
		{
			name: "informational codes (empty status) are skipped",
			events: []LegalEvent{
				{EventCode: "REG", Country: "EP", Date: "2022-01-01"},
				{EventCode: "AX", Country: "EP", Date: "2021-01-01"},
			},
			targetCountry: "EP",
			want:          StatusUnknown,
		},
		{
			name: "LAPS code means lapsed",
			events: []LegalEvent{
				{EventCode: "B1", Country: "EP", Date: "2018-01-01"},
				{EventCode: "LAPS", Country: "DE", Date: "2022-01-01"},
			},
			targetCountry: "DE",
			want:          StatusLapsed,
		},
		{
			name: "RER code means lapsed",
			events: []LegalEvent{
				{EventCode: "B1", Country: "EP", Date: "2018-01-01"},
				{EventCode: "RER", Country: "AT", Date: "2022-01-01"},
			},
			targetCountry: "AT",
			want:          StatusLapsed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DerivePatentStatus(tt.events, tt.targetCountry)
			if got != tt.want {
				t.Errorf("DerivePatentStatus() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDeriveOverallStatus(t *testing.T) {
	tests := []struct {
		name   string
		events []LegalEvent
		want   string
	}{
		{
			name:   "empty events returns unknown",
			events: nil,
			want:   StatusUnknown,
		},
		{
			name: "grant means active overall",
			events: []LegalEvent{
				{EventCode: "AK", Country: "EP", Date: "2018-01-01"},
				{EventCode: "B1", Country: "EP", Date: "2020-06-15"},
			},
			want: StatusActive,
		},
		{
			name: "PG25 per-country lapse is ignored for overall status",
			events: []LegalEvent{
				{EventCode: "B1", Country: "EP", Date: "2020-01-01"},
				{EventCode: "PG25", Country: "DE", Date: "2022-03-01"},
			},
			want: StatusActive,
		},
		{
			name: "revocation overrides everything",
			events: []LegalEvent{
				{EventCode: "B1", Country: "EP", Date: "2018-01-01"},
				{EventCode: "PG25", Country: "DE", Date: "2019-01-01"},
				{EventCode: "27W", Country: "EP", Date: "2020-01-01"},
			},
			want: StatusRevoked,
		},
		{
			name: "withdrawal",
			events: []LegalEvent{
				{EventCode: "17P", Country: "EP", Date: "2018-01-01"},
				{EventCode: "18D", Country: "EP", Date: "2019-01-01"},
			},
			want: StatusWithdrawn,
		},
		{
			name: "expiry",
			events: []LegalEvent{
				{EventCode: "B1", Country: "EP", Date: "2000-01-01"},
				{EventCode: "MK05", Country: "EP", Date: "2020-01-01"},
			},
			want: StatusExpired,
		},
		{
			name: "pending when only examination events",
			events: []LegalEvent{
				{EventCode: "PUAI", Country: "EP", Date: "2022-01-01"},
				{EventCode: "17P", Country: "EP", Date: "2022-06-01"},
			},
			want: StatusPending,
		},
		{
			name: "newer grant overrides older LAPS for overall status",
			events: []LegalEvent{
				{EventCode: "LAPS", Country: "DE", Date: "2018-01-01"},
				{EventCode: "B1", Country: "EP", Date: "2020-01-01"},
			},
			want: StatusActive,
		},
		{
			name: "RER per-country lapse is ignored for overall status",
			events: []LegalEvent{
				{EventCode: "B1", Country: "EP", Date: "2020-01-01"},
				{EventCode: "RER", Country: "AT", Date: "2022-03-01"},
			},
			want: StatusActive,
		},
		{
			name: "LAPS per-country lapse is ignored for overall status",
			events: []LegalEvent{
				{EventCode: "B1", Country: "EP", Date: "2020-01-01"},
				{EventCode: "LAPS", Country: "DE", Date: "2022-03-01"},
			},
			want: StatusActive,
		},
		{
			name: "most recent event wins for overall status",
			events: []LegalEvent{
				{EventCode: "18W", Country: "EP", Date: "2019-01-01"},
				{EventCode: "B1", Country: "EP", Date: "2020-01-01"},
			},
			want: StatusActive,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DeriveOverallStatus(tt.events)
			if got != tt.want {
				t.Errorf("DeriveOverallStatus() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestLCodeStatusMappingCompleteness(t *testing.T) {
	// Verify all mapped values are valid status constants
	validStatuses := map[string]bool{
		StatusActive:    true,
		StatusExpired:   true,
		StatusLapsed:    true,
		StatusWithdrawn: true,
		StatusRevoked:   true,
		StatusPending:   true,
		"":              true, // informational codes
	}

	for code, status := range LCodeStatusMapping {
		if !validStatuses[status] {
			t.Errorf("LCodeStatusMapping[%q] = %q, not a valid status", code, status)
		}
	}

	// Verify minimum coverage
	if len(LCodeStatusMapping) < 20 {
		t.Errorf("LCodeStatusMapping has %d entries, want at least 20", len(LCodeStatusMapping))
	}
}
