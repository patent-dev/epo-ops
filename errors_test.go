package epo_ops

import (
	"errors"
	"net/http"
	"testing"
)

func TestParseErrorXML_DetailedFormat(t *testing.T) {
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<error>
  <code>CLIENT.InvalidReference</code>
  <message>Invalid patent reference format</message>
  <moreInfo>https://ops.epo.org/3.2/rest-services/help</moreInfo>
</error>`

	opsErr, err := parseErrorXML([]byte(xml), http.StatusBadRequest)
	if err != nil {
		t.Fatalf("parseErrorXML failed: %v", err)
	}

	if opsErr.HTTPStatus != http.StatusBadRequest {
		t.Errorf("Expected HTTPStatus %d, got %d", http.StatusBadRequest, opsErr.HTTPStatus)
	}

	if opsErr.Code != "CLIENT.InvalidReference" {
		t.Errorf("Expected code 'CLIENT.InvalidReference', got '%s'", opsErr.Code)
	}

	if opsErr.Message != "Invalid patent reference format" {
		t.Errorf("Expected message 'Invalid patent reference format', got '%s'", opsErr.Message)
	}

	if opsErr.MoreInfo != "https://ops.epo.org/3.2/rest-services/help" {
		t.Errorf("Expected moreInfo URL, got '%s'", opsErr.MoreInfo)
	}

	// Test Error() method includes moreInfo
	errStr := opsErr.Error()
	if errStr != "[400] CLIENT.InvalidReference: Invalid patent reference format (see https://ops.epo.org/3.2/rest-services/help)" {
		t.Errorf("Unexpected Error() output: %s", errStr)
	}
}

func TestParseErrorXML_FaultFormat(t *testing.T) {
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<fault xmlns="http://ops.epo.org">
  <code>404</code>
  <message>Document not found</message>
  <description>No published document found for the specified input</description>
</fault>`

	opsErr, err := parseErrorXML([]byte(xml), http.StatusNotFound)
	if err != nil {
		t.Fatalf("parseErrorXML failed: %v", err)
	}

	if opsErr.HTTPStatus != http.StatusNotFound {
		t.Errorf("Expected HTTPStatus %d, got %d", http.StatusNotFound, opsErr.HTTPStatus)
	}

	if opsErr.Code != "HTTP.404" {
		t.Errorf("Expected code 'HTTP.404', got '%s'", opsErr.Code)
	}

	// Should use description as message
	if opsErr.Message != "No published document found for the specified input" {
		t.Errorf("Expected description as message, got '%s'", opsErr.Message)
	}

	// Test Error() method without moreInfo
	errStr := opsErr.Error()
	if errStr != "[404] HTTP.404: No published document found for the specified input" {
		t.Errorf("Unexpected Error() output: %s", errStr)
	}
}

func TestParseErrorXML_FaultFormat_NoDescription(t *testing.T) {
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<fault xmlns="http://ops.epo.org">
  <code>429</code>
  <message>Fair use limit exceeded</message>
</fault>`

	opsErr, err := parseErrorXML([]byte(xml), http.StatusTooManyRequests)
	if err != nil {
		t.Fatalf("parseErrorXML failed: %v", err)
	}

	if opsErr.Code != "HTTP.429" {
		t.Errorf("Expected code 'HTTP.429', got '%s'", opsErr.Code)
	}

	// Should use message when no description
	if opsErr.Message != "Fair use limit exceeded" {
		t.Errorf("Expected message 'Fair use limit exceeded', got '%s'", opsErr.Message)
	}
}

func TestParseErrorXML_InvalidXML(t *testing.T) {
	xml := `not valid xml`

	opsErr, err := parseErrorXML([]byte(xml), http.StatusBadRequest)
	if err == nil {
		t.Error("Expected error for invalid XML, got nil")
	}

	if opsErr != nil {
		t.Errorf("Expected nil OPSError for invalid XML, got %+v", opsErr)
	}
}

func TestParseErrorXML_EmptyXML(t *testing.T) {
	xml := `<?xml version="1.0" encoding="UTF-8"?><empty/>`

	opsErr, err := parseErrorXML([]byte(xml), http.StatusBadRequest)
	if err == nil {
		t.Error("Expected error for empty XML, got nil")
	}

	if opsErr != nil {
		t.Errorf("Expected nil OPSError for empty XML, got %+v", opsErr)
	}
}

func TestHandleErrorResponse_WithValidErrorXML(t *testing.T) {
	client, _ := NewClient(&Config{
		ConsumerKey:    "test",
		ConsumerSecret: "test",
	})

	xml := `<?xml version="1.0" encoding="UTF-8"?>
<error>
  <code>CLIENT.InvalidReference</code>
  <message>Invalid patent number</message>
</error>`

	err := client.handleErrorResponse(http.StatusBadRequest, []byte(xml))
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	// Should be mapped to NotFoundError based on code
	var notFoundErr *NotFoundError
	if !errors.As(err, &notFoundErr) {
		t.Errorf("Expected NotFoundError, got %T: %v", err, err)
	}

	if notFoundErr.Message != "Invalid patent number" {
		t.Errorf("Expected message 'Invalid patent number', got '%s'", notFoundErr.Message)
	}
}

func TestHandleErrorResponse_WithFaultXML(t *testing.T) {
	client, _ := NewClient(&Config{
		ConsumerKey:    "test",
		ConsumerSecret: "test",
	})

	xml := `<?xml version="1.0" encoding="UTF-8"?>
<fault xmlns="http://ops.epo.org">
  <code>404</code>
  <message>Document not found</message>
  <description>No published document found for the specified input</description>
</fault>`

	err := client.handleErrorResponse(http.StatusNotFound, []byte(xml))
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	// Should be mapped to NotFoundError based on HTTP.404 code
	var notFoundErr *NotFoundError
	if !errors.As(err, &notFoundErr) {
		t.Errorf("Expected NotFoundError, got %T: %v", err, err)
	}

	if notFoundErr.Message != "No published document found for the specified input" {
		t.Errorf("Expected description message, got '%s'", notFoundErr.Message)
	}
}

func TestHandleErrorResponse_FallbackToPlainText(t *testing.T) {
	client, _ := NewClient(&Config{
		ConsumerKey:    "test",
		ConsumerSecret: "test",
	})

	plainText := "Document not found"

	err := client.handleErrorResponse(http.StatusNotFound, []byte(plainText))
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	// Should fall back to NotFoundError with plain text message
	var notFoundErr *NotFoundError
	if !errors.As(err, &notFoundErr) {
		t.Errorf("Expected NotFoundError, got %T: %v", err, err)
	}

	if notFoundErr.Message != plainText {
		t.Errorf("Expected message '%s', got '%s'", plainText, notFoundErr.Message)
	}
}

func TestHandleErrorResponse_AuthErrorMapping(t *testing.T) {
	client, _ := NewClient(&Config{
		ConsumerKey:    "test",
		ConsumerSecret: "test",
	})

	xml := `<?xml version="1.0" encoding="UTF-8"?>
<error>
  <code>CLIENT.InvalidAccessToken</code>
  <message>The access token is invalid</message>
</error>`

	err := client.handleErrorResponse(http.StatusUnauthorized, []byte(xml))
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	var authErr *AuthError
	if !errors.As(err, &authErr) {
		t.Errorf("Expected AuthError, got %T: %v", err, err)
	}

	if authErr.Message != "The access token is invalid" {
		t.Errorf("Expected message 'The access token is invalid', got '%s'", authErr.Message)
	}
}

func TestHandleErrorResponse_QuotaErrorMapping(t *testing.T) {
	client, _ := NewClient(&Config{
		ConsumerKey:    "test",
		ConsumerSecret: "test",
	})

	xml := `<?xml version="1.0" encoding="UTF-8"?>
<error>
  <code>SERVER.RateLimitExceeded</code>
  <message>Rate limit exceeded</message>
</error>`

	err := client.handleErrorResponse(http.StatusTooManyRequests, []byte(xml))
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	var quotaErr *QuotaExceededError
	if !errors.As(err, &quotaErr) {
		t.Errorf("Expected QuotaExceededError, got %T: %v", err, err)
	}

	if quotaErr.Message != "Rate limit exceeded" {
		t.Errorf("Expected message 'Rate limit exceeded', got '%s'", quotaErr.Message)
	}
}

func TestOPSError_ErrorMethod(t *testing.T) {
	tests := []struct {
		name     string
		opsErr   *OPSError
		expected string
	}{
		{
			name: "With moreInfo",
			opsErr: &OPSError{
				HTTPStatus: 400,
				Code:       "CLIENT.InvalidReference",
				Message:    "Invalid input",
				MoreInfo:   "https://example.com/help",
			},
			expected: "[400] CLIENT.InvalidReference: Invalid input (see https://example.com/help)",
		},
		{
			name: "Without moreInfo",
			opsErr: &OPSError{
				HTTPStatus: 404,
				Code:       "HTTP.404",
				Message:    "Not found",
				MoreInfo:   "",
			},
			expected: "[404] HTTP.404: Not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.opsErr.Error()
			if got != tt.expected {
				t.Errorf("Error() = %s, want %s", got, tt.expected)
			}
		})
	}
}
