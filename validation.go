package epo_ops

import (
	"regexp"
	"strings"
)

// Regular expressions for patent number format validation
var (
	// Docdb format: CC.number.KC (e.g., EP.1000000.B1)
	// Country code (2 letters), dot, number (digits), dot, kind code (letter + optional digit)
	docdbPattern = regexp.MustCompile(`^[A-Z]{2}\.\d+\.[A-Z]\d?$`)

	// Epodoc format: CCnumber or CCnumberKC (e.g., EP1000000 or EP1000000B1)
	// Country code (2 letters), number (digits), optional kind code (letter + optional digit)
	epodocPattern = regexp.MustCompile(`^[A-Z]{2}\d+([A-Z]\d?)?$`)

	// Date format: YYYYMMDD (e.g., 20231015)
	datePattern = regexp.MustCompile(`^\d{8}$`)
)

// ValidateDocdbFormat validates the docdb format: CC.number.KC
//
// Examples of valid docdb format:
//   - EP.1000000.B1
//   - US.5551212.A
//   - WO.2023123456.A1
//
// Format rules:
//   - Two-letter country code in uppercase
//   - Dot separator
//   - Numeric document number
//   - Dot separator
//   - Kind code: letter + optional digit
func ValidateDocdbFormat(number string) error {
	if number == "" {
		return &ValidationError{
			Field:   "number",
			Format:  "docdb",
			Value:   number,
			Message: "number cannot be empty",
		}
	}

	if !docdbPattern.MatchString(number) {
		return &ValidationError{
			Field:   "number",
			Format:  "docdb",
			Value:   number,
			Message: "must match pattern: CC.number.KC (e.g., EP.1000000.B1)",
		}
	}

	return nil
}

// ValidateEpodocFormat validates the epodoc format: CCnumber[KC]
//
// Examples of valid epodoc format:
//   - EP1000000B1
//   - EP1000000 (kind code optional)
//   - US5551212A
//   - WO2023123456A1
//
// Format rules:
//   - Two-letter country code in uppercase
//   - Numeric document number
//   - Optional kind code: letter + optional digit
func ValidateEpodocFormat(number string) error {
	if number == "" {
		return &ValidationError{
			Field:   "number",
			Format:  "epodoc",
			Value:   number,
			Message: "number cannot be empty",
		}
	}

	if !epodocPattern.MatchString(number) {
		return &ValidationError{
			Field:   "number",
			Format:  "epodoc",
			Value:   number,
			Message: "must match pattern: CCnumber[KC] (e.g., EP1000000B1)",
		}
	}

	return nil
}

// ValidateOriginalFormat validates the original format (flexible).
//
// The "original" format is the format as provided by the patent authority,
// which can vary significantly. This validation is permissive and only checks
// that the number is non-empty and within reasonable length limits.
//
// Examples of valid original format:
//   - EP 1000000 B1 (with spaces)
//   - US5,551,212 (with commas)
//   - WO/2023/123456
//   - Any format used by patent authorities
func ValidateOriginalFormat(number string) error {
	if number == "" {
		return &ValidationError{
			Field:   "number",
			Format:  "original",
			Value:   number,
			Message: "number cannot be empty",
		}
	}

	// Original format is very flexible - just ensure basic length limits
	if len(number) > 100 {
		return &ValidationError{
			Field:   "number",
			Format:  "original",
			Value:   number,
			Message: "number exceeds maximum length of 100 characters",
		}
	}

	return nil
}

// ValidateFormat validates a patent number based on the specified format.
//
// Parameters:
//   - format: One of "docdb", "epodoc", or "original" (from constants)
//   - number: The patent number to validate
//
// Returns a ValidationError if the format is invalid or the number doesn't match the format rules.
func ValidateFormat(format, number string) error {
	switch format {
	case FormatDocDB:
		return ValidateDocdbFormat(number)
	case FormatEPODOC:
		return ValidateEpodocFormat(number)
	case FormatOriginal:
		return ValidateOriginalFormat(number)
	default:
		return &ValidationError{
			Field:   "format",
			Value:   format,
			Message: "must be 'docdb', 'epodoc', or 'original'",
		}
	}
}

// ValidateDate validates a date string in YYYYMMDD format.
//
// Examples of valid dates:
//   - 20231015 (October 15, 2023)
//   - 19990101 (January 1, 1999)
//
// Note: This only validates the format, not whether the date is valid
// (e.g., it accepts 20231399 which is not a real date).
// Empty string is accepted (date is optional in many API calls).
func ValidateDate(date string) error {
	if date == "" {
		return nil // Date is optional
	}

	if !datePattern.MatchString(date) {
		return &ValidationError{
			Field:   "date",
			Value:   date,
			Message: "must match YYYYMMDD format (e.g., 20231015)",
		}
	}

	return nil
}

// ValidateRefType validates a reference type parameter.
//
// Valid reference types:
//   - "publication" - Published patent documents
//   - "application" - Patent applications
//   - "priority" - Priority documents
func ValidateRefType(refType string) error {
	switch refType {
	case RefTypePublication, RefTypeApplication, RefTypePriority:
		return nil
	default:
		return &ValidationError{
			Field:   "refType",
			Value:   refType,
			Message: "must be 'publication', 'application', or 'priority'",
		}
	}
}

// NormalizeToDocdb converts a patent number to DOCDB format (CC.number.KC).
//
// This function accepts patent numbers in various formats and normalizes them
// to the DOCDB format required by most EPO OPS API endpoints.
//
// Supported input formats:
//   - EPODOC format: "EP1000000B1", "US5551212A", "WO2023123456A1"
//   - DOCDB format: "EP.1000000.B1" (returns unchanged if valid)
//   - With spaces: "EP 1000000 B1" (spaces removed before processing)
//
// Examples:
//   - "EP2884620A2" → "EP.2884620.A2"
//   - "EP.2884620.A2" → "EP.2884620.A2" (already DOCDB)
//   - "US5551212A" → "US.5551212.A"
//   - "EP 1000000 B1" → "EP.1000000.B1"
//
// Returns an error if:
//   - The input is empty
//   - The input cannot be parsed as a valid patent number
//   - The input is in DOCDB format but invalid
func NormalizeToDocdb(number string) (string, error) {
	if number == "" {
		return "", &ValidationError{
			Field:   "number",
			Value:   number,
			Message: "patent number cannot be empty",
		}
	}

	// Remove common whitespace/formatting characters
	// This allows inputs like "EP 1000000 B1" or "EP-1000000-B1"
	var cleaned strings.Builder
	cleaned.Grow(len(number)) // Pre-allocate capacity
	for i := 0; i < len(number); i++ {
		c := number[i]
		// Skip spaces, tabs, hyphens, and slashes
		if c != ' ' && c != '\t' && c != '-' && c != '/' {
			cleaned.WriteByte(c)
		}
	}

	cleanedStr := cleaned.String()
	if cleanedStr == "" {
		return "", &ValidationError{
			Field:   "number",
			Value:   number,
			Message: "patent number contains only whitespace or separators",
		}
	}

	// Check if it's already in valid DOCDB format (contains dots at expected positions)
	// DOCDB format has a dot at position 2 (after country code)
	if len(cleanedStr) > 4 && cleanedStr[2] == '.' {
		// Validate the DOCDB format
		if err := ValidateDocdbFormat(cleanedStr); err != nil {
			return "", &ValidationError{
				Field:   "number",
				Value:   number,
				Message: "invalid DOCDB format: " + err.Error(),
			}
		}
		// Already valid DOCDB format
		return cleanedStr, nil
	}

	// Try to parse as EPODOC format
	parsed := ParsePatentNumber(cleanedStr)

	// Check if parsing was successful
	if parsed.Country == "" || parsed.Number == "" || parsed.Kind == "" {
		return "", &ValidationError{
			Field:   "number",
			Value:   number,
			Message: "unable to parse patent number (expected formats: EP2884620A2 or EP.2884620.A2)",
		}
	}

	// Convert to DOCDB format
	docdb := parsed.Country + "." + parsed.Number + "." + parsed.Kind

	// Final validation of the generated DOCDB format
	if err := ValidateDocdbFormat(docdb); err != nil {
		return "", &ValidationError{
			Field:   "number",
			Value:   number,
			Message: "converted format is invalid: " + err.Error(),
		}
	}

	return docdb, nil
}
