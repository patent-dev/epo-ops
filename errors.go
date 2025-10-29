package epo_ops

import (
	"encoding/xml"
	"fmt"
)

// AuthError represents an authentication error.
type AuthError struct {
	StatusCode int
	Message    string
}

func (e *AuthError) Error() string {
	if e.StatusCode > 0 {
		return fmt.Sprintf("auth error (status %d): %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("auth error: %s", e.Message)
}

// NotFoundError represents a 404 error (document doesn't exist).
type NotFoundError struct {
	Resource string
	Message  string
}

func (e *NotFoundError) Error() string {
	if e.Resource != "" {
		return fmt.Sprintf("not found: %s (%s)", e.Resource, e.Message)
	}
	return fmt.Sprintf("not found: %s", e.Message)
}

// QuotaExceededError represents a fair use quota limit error.
type QuotaExceededError struct {
	Message string
}

func (e *QuotaExceededError) Error() string {
	return fmt.Sprintf("quota exceeded: %s", e.Message)
}

// AmbiguousPatentError represents a situation where a patent number
// has multiple kind codes available (A1, B1, etc.) and the user must
// choose a specific one.
type AmbiguousPatentError struct {
	Country   string
	Number    string
	KindCodes []string
	Message   string
}

func (e *AmbiguousPatentError) Error() string {
	if len(e.KindCodes) > 0 {
		return fmt.Sprintf("ambiguous patent %s%s, available kinds: %v - %s",
			e.Country, e.Number, e.KindCodes, e.Message)
	}
	return fmt.Sprintf("ambiguous patent %s%s - %s", e.Country, e.Number, e.Message)
}

// ServiceUnavailableError represents a temporary service outage.
type ServiceUnavailableError struct {
	StatusCode int
	Message    string
	RetryAfter string // Optional Retry-After header value
}

func (e *ServiceUnavailableError) Error() string {
	if e.RetryAfter != "" {
		return fmt.Sprintf("service unavailable (status %d): %s, retry after: %s",
			e.StatusCode, e.Message, e.RetryAfter)
	}
	return fmt.Sprintf("service unavailable (status %d): %s", e.StatusCode, e.Message)
}

// OPSError represents a structured error response from EPO OPS API.
// The EPO OPS API returns errors in XML format with a code, message, and optional moreInfo URL.
type OPSError struct {
	HTTPStatus int    // HTTP status code
	Code       string // EPO error code (e.g., "CLIENT.InvalidReference", "SERVER.EntityNotFound")
	Message    string // Human-readable error message
	MoreInfo   string // Optional URL with more information
}

func (e *OPSError) Error() string {
	if e.MoreInfo != "" {
		return fmt.Sprintf("[%d] %s: %s (see %s)", e.HTTPStatus, e.Code, e.Message, e.MoreInfo)
	}
	return fmt.Sprintf("[%d] %s: %s", e.HTTPStatus, e.Code, e.Message)
}

// XMLParseError represents an error during XML parsing.
// This error provides context about what failed during XML unmarshaling
// including the parser name, problematic element, and a sample of the XML.
type XMLParseError struct {
	Parser    string // e.g., "ParseFamily", "ParseLegal"
	Element   string // e.g., "family-member", "legal-event"
	XMLSample string // First 200 chars of problematic XML
	Cause     error  // Underlying error from xml.Unmarshal
}

func (e *XMLParseError) Error() string {
	msg := fmt.Sprintf("XML parsing failed in %s", e.Parser)
	if e.Element != "" {
		msg += fmt.Sprintf(" (element: %s)", e.Element)
	}
	if e.Cause != nil {
		msg += fmt.Sprintf(": %v", e.Cause)
	}
	if e.XMLSample != "" {
		msg += fmt.Sprintf("\nXML sample: %s", e.XMLSample)
	}
	return msg
}

func (e *XMLParseError) Unwrap() error {
	return e.Cause
}

// DataValidationError represents an error when parsed data is incomplete or invalid.
// This occurs when XML unmarshaling succeeds but the resulting data structure
// is missing required fields or contains invalid values.
type DataValidationError struct {
	Parser       string // e.g., "ParseFamily", "ParseLegal"
	MissingField string // Name of the missing or invalid field
	Message      string // Description of what's wrong
}

func (e *DataValidationError) Error() string {
	msg := fmt.Sprintf("data validation failed in %s", e.Parser)
	if e.MissingField != "" {
		msg += fmt.Sprintf(" (field: %s)", e.MissingField)
	}
	if e.Message != "" {
		msg += fmt.Sprintf(": %s", e.Message)
	}
	return msg
}

// parseErrorXML parses EPO OPS error response XML into an OPSError struct.
// EPO error responses can have two formats:
//
// Format 1 (detailed error):
//
//	<error>
//	  <code>CLIENT.InvalidReference</code>
//	  <message>Invalid patent number format</message>
//	  <moreInfo>http://...</moreInfo>
//	</error>
//
// Format 2 (fault):
//
//	<fault xmlns="http://ops.epo.org">
//	  <code>404</code>
//	  <message>Document not found</message>
//	  <description>No published document found...</description>
//	</fault>
func parseErrorXML(body []byte, statusCode int) (*OPSError, error) {
	// Try format 1: <error> with string codes
	var errResp struct {
		XMLName  xml.Name `xml:"error"`
		Code     string   `xml:"code"`
		Message  string   `xml:"message"`
		MoreInfo string   `xml:"moreInfo"`
	}

	if err := xml.Unmarshal(body, &errResp); err == nil && errResp.Code != "" {
		return &OPSError{
			HTTPStatus: statusCode,
			Code:       errResp.Code,
			Message:    errResp.Message,
			MoreInfo:   errResp.MoreInfo,
		}, nil
	}

	// Try format 2: <fault> with numeric codes and description
	var faultResp struct {
		XMLName     xml.Name `xml:"fault"`
		Code        string   `xml:"code"`
		Message     string   `xml:"message"`
		Description string   `xml:"description"`
	}

	if err := xml.Unmarshal(body, &faultResp); err == nil && faultResp.Code != "" {
		// Use description as message if available, otherwise use message
		message := faultResp.Message
		if faultResp.Description != "" {
			message = faultResp.Description
		}

		return &OPSError{
			HTTPStatus: statusCode,
			Code:       "HTTP." + faultResp.Code, // Prefix numeric codes with "HTTP."
			Message:    message,
			MoreInfo:   "",
		}, nil
	}

	// Could not parse as either format
	return nil, fmt.Errorf("unable to parse error XML")
}

// ConfigError represents a configuration error.
type ConfigError struct {
	Message string
}

func (e *ConfigError) Error() string {
	return fmt.Sprintf("config error: %s", e.Message)
}

// ValidationError represents an input validation error.
// This is returned when user-provided input doesn't match the expected format.
type ValidationError struct {
	Field   string // Field name that failed validation (e.g., "number", "format", "date")
	Format  string // Expected format (e.g., "docdb", "epodoc")
	Value   string // The invalid value provided
	Message string // Human-readable error message
}

func (e *ValidationError) Error() string {
	if e.Format != "" {
		return fmt.Sprintf("validation error: %s (%s format): %s - got: %q", e.Field, e.Format, e.Message, e.Value)
	}
	return fmt.Sprintf("validation error: %s: %s - got: %q", e.Field, e.Message, e.Value)
}

// NotImplementedError represents a not-yet-implemented feature.
type NotImplementedError struct {
	Message string
}

func (e *NotImplementedError) Error() string {
	return fmt.Sprintf("not implemented: %s", e.Message)
}
