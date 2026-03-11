package epo_ops

import (
	"sort"
	"strings"
)

// Patent status values derived from INPADOC legal event analysis.
const (
	StatusActive    = "active"
	StatusExpired   = "expired"
	StatusLapsed    = "lapsed"
	StatusWithdrawn = "withdrawn"
	StatusRevoked   = "revoked"
	StatusPending   = "pending"
	StatusUnknown   = "unknown"
)

// perCountryLapseCodes are event codes that represent per-country lapse events.
// These affect only a specific contracting state, not the overall patent status.
// DeriveOverallStatus skips these; use DerivePatentStatus with a targetCountry instead.
var perCountryLapseCodes = map[string]bool{
	"PG25": true, // Lapsed in a contracting state
	"PG2D": true, // Lapsed in a contracting state (new rules)
	"RER":  true, // Ceased as to paragraph 5 lit. 3 (per-country)
	"LAPS": true, // Lapse for non-payment of fees (per-country)
}

// LCodeStatusMapping maps EPO INPADOC legal event codes (L008EP values)
// to their patent status implications. These are the event codes found
// in the ops:legal elements returned by the EPO OPS legal endpoint.
//
// Event codes that are purely informational (no status change) are not
// included. Use DerivePatentStatus() to analyze a sequence of events.
var LCodeStatusMapping = map[string]string{
	// Grant and active status
	"AK":   StatusActive, // Designated contracting states
	"B1":   StatusActive, // Grant of European patent
	"B2":   StatusActive, // Amended patent specification
	"FGA1": StatusActive, // Grant of patent (national)
	"GRAA": StatusActive, // Grant of patent (application granted)
	"GRAP": StatusActive, // Grant of patent (application published)

	// Examination and pending status
	"17P":  StatusPending, // Request for examination filed
	"17Q":  StatusPending, // First examination report
	"PUAI": StatusPending, // Search report published
	"STAA": "",            // Status info - requires L019EP context, skip

	// Lapsed status (per-country)
	"PG25": StatusLapsed, // Lapsed in a contracting state
	"PG2D": StatusLapsed, // Lapsed in a contracting state (new rules)
	"RER":  StatusLapsed, // Ceased as to paragraph 5 lit. 3
	"LAPS": StatusLapsed, // Lapse for non-payment of fees

	// Withdrawn status
	"18D":  StatusWithdrawn, // Application deemed withdrawn
	"18W":  StatusWithdrawn, // Application withdrawn by applicant
	"STAB": StatusWithdrawn, // Information on status - withdrawn

	// Revoked status
	"27W": StatusRevoked, // Patent revoked
	"REV": StatusRevoked, // Patent revoked (opposition)

	// Expired status
	"MK05": StatusExpired, // Expiry of patent term
	"MK06": StatusExpired, // Expiry of patent term with SPC
	"MK07": StatusExpired, // Expiry of supplementary protection certificate

	// Extension and informational (no status change)
	"AX":   "", // Extension/validation states - informational
	"REG":  "", // Reference to national code - informational
	"RAP1": "", // Party data changed - informational
	"RAP3": "", // Party data changed - informational
	"DAX":  "", // Extension states applied - informational
	"RIN1": "", // Inventor changed - informational
}

// DerivePatentStatus analyzes a sequence of legal events and determines the
// patent status for a specific country. It filters events by country,
// sorts by date descending, and returns the status implied by the most recent
// status-changing event.
//
// If targetCountry is empty, all events are considered regardless of country,
// including per-country lapse events (PG25, PG2D, RER, LAPS). For overall
// patent status that excludes per-country lapses, use DeriveOverallStatus instead.
//
// Returns StatusUnknown if no status-changing events are found.
func DerivePatentStatus(events []LegalEvent, targetCountry string) string {
	if len(events) == 0 {
		return StatusUnknown
	}

	// Filter by country if specified
	var filtered []LegalEvent
	for _, e := range events {
		if targetCountry == "" || strings.EqualFold(e.Country, targetCountry) {
			filtered = append(filtered, e)
		}
	}

	if len(filtered) == 0 {
		return StatusUnknown
	}

	// Sort by date descending (most recent first)
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Date > filtered[j].Date
	})

	// Find the most recent status-changing event
	for _, e := range filtered {
		code := strings.TrimSpace(e.EventCode)
		if code == "" {
			code = strings.TrimSpace(e.Code)
		}
		if code == "" {
			continue
		}

		status, exists := LCodeStatusMapping[code]
		if exists && status != "" {
			return status
		}
	}

	return StatusUnknown
}

// DeriveOverallStatus analyzes all events and determines the overall patent
// status across all countries. It sorts events by date descending and returns
// the status implied by the most recent non-per-country status-changing event.
//
// Per-country lapse events (PG25, PG2D, RER, LAPS) are skipped because they
// only affect individual contracting states, not the patent as a whole. Use
// DerivePatentStatus with a specific targetCountry for per-country status.
//
// Returns StatusUnknown if no status-changing events are found.
func DeriveOverallStatus(events []LegalEvent) string {
	if len(events) == 0 {
		return StatusUnknown
	}

	// Sort by date descending (most recent first)
	sorted := make([]LegalEvent, len(events))
	copy(sorted, events)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Date > sorted[j].Date
	})

	// Find the most recent non-per-country status-changing event
	for _, e := range sorted {
		code := strings.TrimSpace(e.EventCode)
		if code == "" {
			code = strings.TrimSpace(e.Code)
		}
		if code == "" {
			continue
		}

		// Skip per-country lapse events for overall status
		if perCountryLapseCodes[code] {
			continue
		}

		status, exists := LCodeStatusMapping[code]
		if exists && status != "" {
			return status
		}
	}

	return StatusUnknown
}
