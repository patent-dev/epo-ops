package epo_ops

import "time"

// Reference types for API requests
const (
	RefTypePublication = "publication"
	RefTypeApplication = "application"
	RefTypePriority    = "priority"
)

// Number formats for API requests
const (
	FormatDocDB    = "docdb"
	FormatEPODOC   = "epodoc"
	FormatOriginal = "original"
)

// Image types for GetImage (lowercase per EPO OPS specification)
const (
	ImageTypeThumbnail = "thumbnail" // Drawings only
	ImageTypeFullImage = "fullimage" // Full document with all sections
	ImageTypeFirstPage = "firstpage" // First page clipping (requires kind="PA")
)

// Search constituents
const (
	ConstituentBiblio    = "biblio"
	ConstituentAbstract  = "abstract"
	ConstituentFullCycle = "full-cycle"
)

// Endpoint types for Accept header selection
const (
	EndpointBiblio      = "biblio"
	EndpointFulltext    = "fulltext"
	EndpointClaims      = "claims"
	EndpointDescription = "description"
	EndpointAbstract    = "abstract"
	EndpointFamily      = "family"
	EndpointLegal       = "legal"
	EndpointRegister    = "register"
	EndpointSearch      = "search"
	EndpointImages      = "images"
)

// Config holds configuration for the EPO OPS client.
type Config struct {
	// BaseURL is the base URL for the OPS API.
	// Default: "https://ops.epo.org/3.2/rest-services"
	BaseURL string

	// AuthURL is the OAuth2 token endpoint URL.
	// Default: "https://ops.epo.org/3.2/auth/accesstoken"
	// This field is primarily for testing purposes.
	AuthURL string

	// ConsumerKey is the OAuth2 consumer key (required).
	ConsumerKey string

	// ConsumerSecret is the OAuth2 consumer secret (required).
	ConsumerSecret string

	// MaxRetries is the maximum number of retries for failed requests.
	// Default: 3
	MaxRetries int

	// RetryDelay is the base delay between retries.
	// Default: 1 second
	RetryDelay time.Duration

	// Timeout is the HTTP client timeout.
	// Default: 30 seconds
	Timeout time.Duration
}

// DefaultConfig returns a Config with default values.
func DefaultConfig() *Config {
	return &Config{
		BaseURL:    "https://ops.epo.org/3.2/rest-services",
		MaxRetries: 3,
		RetryDelay: 1 * time.Second,
		Timeout:    30 * time.Second,
	}
}

// BulkOptions holds configuration options for bulk retrieval operations.
type BulkOptions struct {
	// MaxConcurrent is the maximum number of concurrent requests.
	// Default: 1 (sequential processing)
	// Note: Concurrent processing not yet implemented
	MaxConcurrent int

	// OnProgress is called after each batch completes.
	// Parameters: current batch number, total batches
	// Optional: set to nil to disable progress callbacks
	OnProgress func(current, total int)
}

// ImageInquiry represents the response from an image inquiry request.
// It contains information about available images for a patent document.
type ImageInquiry struct {
	// DocumentInstances contains the list of available document types and their metadata
	DocumentInstances []DocumentInstance
}

// DocumentInstance represents a single document type available for a patent.
// For example, a patent might have separate instances for drawings and full document.
type DocumentInstance struct {
	// Description is a human-readable description of the document type
	// Examples: "Drawing", "FullDocument", "FirstPageClipping"
	Description string

	// Link is the URL to retrieve this document instance
	Link string

	// NumberOfPages is the total number of pages in this document instance
	NumberOfPages int

	// Formats lists the available formats for this document instance
	// Examples: ["pdf", "tiff"], ["application/pdf", "image/tiff"]
	Formats []string

	// DocType is the internal document type identifier
	// Examples: "Drawing", "FullDocument"
	DocType string
}

// UsageStats represents usage statistics from the EPO OPS Data Usage API.
//
// The Data Usage API endpoint (/3.2/developers/me/stats/usage) provides
// historical usage data for quota monitoring and analysis.
//
// Usage statistics are updated within 10 minutes of each hour and aligned
// on midnight UTC/GMT boundaries. This API does not count against quotas.
//
// Example usage:
//
//	// Get usage for a specific date
//	stats, err := client.GetUsageStats(ctx, "01/01/2024")
//
//	// Get usage for a date range
//	stats, err := client.GetUsageStats(ctx, "01/01/2024~07/01/2024")
type UsageStats struct {
	// TimeRange is the requested time range (dd/mm/yyyy or dd/mm/yyyy~dd/mm/yyyy)
	TimeRange string

	// Entries contains usage data points, typically one per hour
	Entries []UsageEntry
}

// UsageEntry represents a single usage data point.
//
// Each entry corresponds to an hour of usage, with data updated
// within 10 minutes of the hour boundary.
type UsageEntry struct {
	// Timestamp is the Unix timestamp (seconds since epoch) for this entry
	Timestamp int64

	// TotalResponseSize is the total size of API responses in bytes for this period
	TotalResponseSize int64

	// MessageCount is the number of API requests made during this period
	MessageCount int

	// Service identifies which OPS service was used (optional field)
	Service string
}

// PatentNumber represents the parsed components of a patent number.
type PatentNumber struct {
	// Country is the two-letter country code (e.g., "EP", "US", "DE").
	// Empty if the input doesn't contain a valid country code.
	Country string

	// Number is the numeric/alphanumeric portion of the patent number.
	// Empty if no valid number portion is found.
	Number string

	// Kind is the kind code (e.g., "A1", "B1", "C").
	// Empty if the patent number is invalid.
	Kind string
}

// ValidatePatentNumber performs basic validation on a patent number string.
// Returns an error if the patent number is invalid.
//
// Validation rules:
//   - Must not be empty
//   - Length must be between 4 and 50 characters
//   - Must contain only ASCII letters, digits, and basic punctuation
//
// Note: This is a permissive validation. Invalid patent numbers will be
// rejected by the EPO API with appropriate error messages.
func ValidatePatentNumber(number string) error {
	if number == "" {
		return &ConfigError{Message: "patent number cannot be empty"}
	}
	if len(number) > 50 {
		return &ConfigError{Message: "patent number too long (max 50 characters)"}
	}
	// Allow letters, digits, and common separators (space, dash, dot, slash)
	for i := 0; i < len(number); i++ {
		c := number[i]
		if !((c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') ||
			(c >= '0' && c <= '9') || c == ' ' || c == '-' || c == '.' || c == '/') {
			return &ConfigError{Message: "patent number contains invalid characters"}
		}
	}
	return nil
}

// ParsePatentNumber splits a patent number into its components: country code, number, and kind code.
//
// Patent number format rules:
//   - Minimum length: 4 characters (CCnK format)
//   - First two characters (A-Z, a-z) are the country code (CC)
//   - Followed by the number portion (n) - can start with A-Z, a-z and must contain at least one 0-9
//   - Followed by REQUIRED kind code (K) - 1-2 chars, always starts with A-Z, a-z, always at the end
//
// Examples:
//   - "EP2400812A1" → {Country: "EP", Number: "2400812", Kind: "A1"}
//   - "DE123C" → {Country: "DE", Number: "123", Kind: "C"}
//   - "DE123C1" → {Country: "DE", Number: "123", Kind: "C1"}
//   - "USD123456S1" → {Country: "US", Number: "D123456", Kind: "S1"}
//
// Invalid examples (return empty PatentNumber):
//   - "DE123" → {Country: "", Number: "", Kind: ""} (missing kind code)
//   - "123C" → {Country: "", Number: "", Kind: ""} (no country code)
//   - "D123C" → {Country: "", Number: "", Kind: ""} (country code too short)
//   - "DEC" → {Country: "", Number: "", Kind: ""} (no number portion)
func ParsePatentNumber(number string) PatentNumber {
	result := PatentNumber{}

	// Minimum length: 4 chars (CC + at least 1 digit + K)
	if len(number) < 4 {
		return result
	}

	// First two characters must be letters (country code)
	if !isLetter(number[0]) || !isLetter(number[1]) {
		return result
	}

	result.Country = number[0:2]

	// Find where the number portion ends and kind code begins
	// The kind code is 1-2 chars at the end, starting with a letter
	// We scan from the end to find where the kind code starts

	// Case 1: Last 2 chars are letters → 2-char kind code
	if len(number) >= 5 && isLetter(number[len(number)-2]) && isLetter(number[len(number)-1]) {
		result.Number = number[2 : len(number)-2]
		result.Kind = number[len(number)-2:]
		// Validate number portion has at least one digit
		if !hasDigit(result.Number) {
			return PatentNumber{}
		}
		return result
	}

	// Case 2: Last char is letter, second-to-last is digit → 1-char kind code
	if len(number) >= 4 && isDigit(number[len(number)-2]) && isLetter(number[len(number)-1]) {
		result.Number = number[2 : len(number)-1]
		result.Kind = number[len(number)-1:]
		// Validate number portion has at least one digit
		if !hasDigit(result.Number) {
			return PatentNumber{}
		}
		return result
	}

	// Case 3: Last 2 chars are letter+digit (e.g., "C1") → 2-char kind code
	if len(number) >= 5 && isLetter(number[len(number)-2]) && isDigit(number[len(number)-1]) {
		result.Number = number[2 : len(number)-2]
		result.Kind = number[len(number)-2:]
		// Validate number portion has at least one digit
		if !hasDigit(result.Number) {
			return PatentNumber{}
		}
		return result
	}

	// No valid kind code found - patent number is invalid
	return PatentNumber{}
}

// isLetter checks if a byte is a letter (A-Z or a-z)
func isLetter(b byte) bool {
	return (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z')
}

// isDigit checks if a byte is a digit (0-9)
func isDigit(b byte) bool {
	return b >= '0' && b <= '9'
}

// hasDigit checks if a string contains at least one digit
func hasDigit(s string) bool {
	for i := 0; i < len(s); i++ {
		if isDigit(s[i]) {
			return true
		}
	}
	return false
}
