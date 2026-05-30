package epo_ops

import (
	"errors"
	"fmt"
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

func TestParseEPOJSONErrorBody_UpstreamNotFound(t *testing.T) {
	body := []byte(`<?xml-stylesheet type='text/xsl' href='../../../style/cpc.xsl' ?>{"error":{"message":"Unexpected response from remote server: GET http://classification-web-green-v4.bss-cst.svc.cluster.local/foo resulted HTTP 404 Not Found","details":{"original":{"code":404,"message":"Invalid or unknown class H04W84"}}}}`)

	err, ok := parseEPOJSONErrorBody(body)
	if !ok {
		t.Fatal("expected ok=true for JSON-in-XML envelope")
	}

	var notFound *NotFoundError
	if !errors.As(err, &notFound) {
		t.Fatalf("expected NotFoundError, got %T: %v", err, err)
	}
	if notFound.Message != "Invalid or unknown class H04W84" {
		t.Errorf("expected upstream message, got %q", notFound.Message)
	}
}

func TestParseEPOJSONErrorBody_FallbackToOuterMessage(t *testing.T) {
	body := []byte(`<?xml-stylesheet type='text/xsl' href='x'?>{"error":{"message":"Something broke","code":"SERVER.Generic"}}`)

	err, ok := parseEPOJSONErrorBody(body)
	if !ok {
		t.Fatal("expected ok=true")
	}

	var opsErr *OPSError
	if !errors.As(err, &opsErr) {
		t.Fatalf("expected OPSError fallback, got %T: %v", err, err)
	}
	if opsErr.Message != "Something broke" {
		t.Errorf("expected outer message, got %q", opsErr.Message)
	}
	if opsErr.HTTPStatus != http.StatusOK {
		t.Errorf("expected HTTPStatus 200 for missing upstream code, got %d", opsErr.HTTPStatus)
	}
}

func TestParseEPOJSONErrorBody_MapsUpstreamCodes(t *testing.T) {
	body := func(code int) []byte {
		return []byte(fmt.Sprintf(`<?xml-stylesheet href='x'?>{"error":{"details":{"original":{"code":%d,"message":"upstream"}}}}`, code))
	}

	err, _ := parseEPOJSONErrorBody(body(http.StatusUnauthorized))
	var authErr *AuthError
	if !errors.As(err, &authErr) {
		t.Errorf("401: expected AuthError, got %T", err)
	}

	err, _ = parseEPOJSONErrorBody(body(http.StatusForbidden))
	var quotaErr *QuotaExceededError
	if !errors.As(err, &quotaErr) {
		t.Errorf("403: expected QuotaExceededError, got %T", err)
	}

	err, _ = parseEPOJSONErrorBody(body(http.StatusTooManyRequests))
	if !errors.As(err, &quotaErr) {
		t.Errorf("429: expected QuotaExceededError, got %T", err)
	}

	err, _ = parseEPOJSONErrorBody(body(http.StatusServiceUnavailable))
	var svcErr *ServiceUnavailableError
	if !errors.As(err, &svcErr) {
		t.Errorf("503: expected ServiceUnavailableError, got %T", err)
	}
}

func TestParseEPOJSONErrorBody_RejectsNormalXML(t *testing.T) {
	body := []byte(`<?xml version="1.0" encoding="UTF-8"?><?xml-stylesheet type='text/xsl' href='x'?><ops:world-patent-data xmlns:ops="http://ops.epo.org"><ops:foo/></ops:world-patent-data>`)

	err, ok := parseEPOJSONErrorBody(body)
	if ok {
		t.Errorf("expected ok=false for well-formed XML, got err=%v", err)
	}
}

func TestParseEPOJSONErrorBody_RejectsEmptyAndMalformed(t *testing.T) {
	cases := [][]byte{
		nil,
		[]byte(""),
		[]byte("   "),
		[]byte(`<?xml-stylesheet href='x'?>not json`),
		[]byte(`<?xml-stylesheet href='x'?>{"unrelated":1}`),
		[]byte(`<?xml-stylesheet href='x'`), // unterminated PI
	}
	for i, body := range cases {
		if _, ok := parseEPOJSONErrorBody(body); ok {
			t.Errorf("case %d: expected ok=false for %q", i, body)
		}
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
