package cql

import (
	"errors"
	"strings"
	"testing"
)

func TestParseCQL_ValidQueries(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		wantValid  bool
		wantTokens int
	}{
		{
			name:       "Simple field=value",
			query:      "ti=bluetooth",
			wantValid:  true,
			wantTokens: 3, // ti, =, bluetooth
		},
		{
			name:       "Two fields with AND",
			query:      "ti=bluetooth AND pa=ericsson",
			wantValid:  true,
			wantTokens: 7, // ti, =, bluetooth, AND, pa, =, ericsson
		},
		{
			name:       "Complex query with OR and parentheses",
			query:      "(ti=5g OR ab=5g) AND pa=apple",
			wantValid:  true,
			wantTokens: 13,
		},
		{
			name:       "Quoted value",
			query:      "pa=\"Apple Inc\"",
			wantValid:  true,
			wantTokens: 5, // pa, =, ", Apple Inc, "
		},
		{
			name:       "Multiple operators",
			query:      "ti=wireless AND pa=ericsson OR in=smith",
			wantValid:  true,
			wantTokens: 11,
		},
		{
			name:       "Publication number search",
			query:      "pn=EP1000000",
			wantValid:  true,
			wantTokens: 3,
		},
		{
			name:       "Date range",
			query:      "pd>=20200101",
			wantValid:  true,
			wantTokens: 3,
		},
		{
			name:       "IPC classification",
			query:      "ic=H04W",
			wantValid:  true,
			wantTokens: 3,
		},
		{
			name:       "CPC classification",
			query:      "cpc=H04W84/18",
			wantValid:  true,
			wantTokens: 3,
		},
		{
			name:       "Nested parentheses",
			query:      "((ti=5g OR ti=lte) AND (pa=apple OR pa=samsung))",
			wantValid:  true,
			wantTokens: 21, // 2x(, ti, =, 5g, OR, ti, =, lte, ), AND, (, pa, =, apple, OR, pa, =, samsung, ), )
		},
		{
			name:       "Abstract search",
			query:      "ab=antenna",
			wantValid:  true,
			wantTokens: 3,
		},
		{
			name:       "Inventor search",
			query:      "in=smith",
			wantValid:  true,
			wantTokens: 3,
		},
		{
			name:       "Application number",
			query:      "ap=EP2020123456",
			wantValid:  true,
			wantTokens: 3,
		},
		{
			name:       "Priority number",
			query:      "pr=US2019123456",
			wantValid:  true,
			wantTokens: 3,
		},
		{
			name:       "Simple search term",
			query:      "bluetooth",
			wantValid:  true,
			wantTokens: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, err := ParseCQL(tt.query)
			if err != nil {
				t.Fatalf("ParseCQL() error = %v", err)
			}

			if q.Valid != tt.wantValid {
				t.Errorf("ParseCQL() Valid = %v, want %v", q.Valid, tt.wantValid)
				if len(q.Errors) > 0 {
					t.Errorf("Errors: %v", q.Errors)
				}
			}

			if len(q.Tokens) != tt.wantTokens {
				t.Errorf("ParseCQL() got %d tokens, want %d", len(q.Tokens), tt.wantTokens)
				for i, token := range q.Tokens {
					t.Logf("CQLToken %d: %s = %q", i, token.Type, token.Value)
				}
			}

			if q.Raw != tt.query {
				t.Errorf("ParseCQL() Raw = %q, want %q", q.Raw, tt.query)
			}
		})
	}
}

func TestParseCQL_InvalidQueries(t *testing.T) {
	tests := []struct {
		name      string
		query     string
		wantError string
	}{
		{
			name:      "Empty query",
			query:     "",
			wantError: "cannot be empty",
		},
		{
			name:      "Only whitespace",
			query:     "   ",
			wantError: "cannot be empty",
		},
		{
			name:      "Invalid field name",
			query:     "invalidfield=value",
			wantError: "invalid field",
		},
		{
			name:      "Unmatched opening parenthesis",
			query:     "(ti=bluetooth",
			wantError: "unclosed parentheses",
		},
		{
			name:      "Unmatched closing parenthesis",
			query:     "ti=bluetooth)",
			wantError: "unmatched closing parenthesis",
		},
		{
			name:      "Multiple unmatched parentheses",
			query:     "((ti=bluetooth",
			wantError: "unclosed parentheses",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, err := ParseCQL(tt.query)

			if tt.wantError == "cannot be empty" {
				if err == nil {
					t.Errorf("ParseCQL() expected error containing %q, got nil", tt.wantError)
				} else if !strings.Contains(err.Error(), tt.wantError) {
					t.Errorf("ParseCQL() error = %v, want error containing %q", err, tt.wantError)
				}
			} else {
				if err != nil {
					// Error during parsing
					if !strings.Contains(err.Error(), tt.wantError) {
						t.Errorf("ParseCQL() error = %v, want error containing %q", err, tt.wantError)
					}
				} else {
					// Check validation errors
					if q.Valid {
						t.Errorf("ParseCQL() Valid = true, want false")
					}

					valErr := q.Validate()
					if valErr == nil {
						t.Errorf("Validate() expected error, got nil")
					} else if !strings.Contains(valErr.Error(), tt.wantError) {
						t.Errorf("Validate() error = %v, want error containing %q", valErr, tt.wantError)
					}
				}
			}
		})
	}
}

func TestIsValidField(t *testing.T) {
	tests := []struct {
		field string
		want  bool
	}{
		{"ti", true},
		{"ab", true},
		{"pa", true},
		{"in", true},
		{"pn", true},
		{"ap", true},
		{"pr", true},
		{"pd", true},
		{"ic", true},
		{"cpc", true},
		{"prd", true}, // not live-confirmable (API returns 500); retained, see fields.go
		// Fields the live OPS API rejects with CLIENT.InvalidIndex "Invalid index
		// name X" - the validator must treat them as invalid (verified 2026-06-12).
		{"ad", false},
		{"de", false},
		{"ep", false},
		{"pc", false},
		{"ecla", false},
		{"ctc", false},
		{"invalid", false},
		{"title", false},
		{"", false},
		{"TI", false}, // case-sensitive
	}

	for _, tt := range tests {
		t.Run(tt.field, func(t *testing.T) {
			got := IsValidField(tt.field)
			if got != tt.want {
				t.Errorf("IsValidField(%q) = %v, want %v", tt.field, got, tt.want)
			}
		})
	}
}

func TestIsValidOperator(t *testing.T) {
	tests := []struct {
		op   string
		want bool
	}{
		{"AND", true},
		{"OR", true},
		{"NOT", true},
		{"and", true},
		{"or", true},
		{"not", true},
		{"PROX", true},
		{"prox", true},
		{"ADJ", true},
		{"NEAR", true},
		{"WITH", true},
		{"INVALID", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.op, func(t *testing.T) {
			got := IsValidOperator(tt.op)
			if got != tt.want {
				t.Errorf("IsValidOperator(%q) = %v, want %v", tt.op, got, tt.want)
			}
		})
	}
}

func TestProximityModifierAccepted(t *testing.T) {
	// The proximity modifier form prox/distance<=N is valid CQL: verified live
	// 2026-06-12, "ti=green prox/distance<=3 ti=energy" returns results (HTTP 200).
	// The validator must NOT treat "prox/distance" as a field name and reject it.
	q := "ti=green prox/distance<=3 ti=energy"
	parsed, err := ParseCQL(q)
	if err != nil {
		t.Fatalf("ParseCQL(%q): %v", q, err)
	}
	if err := parsed.Validate(); err != nil {
		t.Errorf("Validate(%q) = %v, want nil (proximity modifier wrongly rejected as a field)", q, err)
	}
}

func TestCQLQuery_URLEncode(t *testing.T) {
	tests := []struct {
		name  string
		query string
		want  string
	}{
		{
			name:  "Simple query",
			query: "ti=bluetooth",
			want:  "ti%3Dbluetooth",
		},
		{
			name:  "CQLQuery with spaces",
			query: "ti=bluetooth AND pa=apple",
			want:  "ti%3Dbluetooth+AND+pa%3Dapple",
		},
		{
			name:  "CQLQuery with special characters",
			query: "pa=\"Apple Inc\"",
			want:  "pa%3D%22Apple+Inc%22",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, err := ParseCQL(tt.query)
			if err != nil {
				t.Fatalf("ParseCQL() error = %v", err)
			}

			got := q.URLEncode()
			if got != tt.want {
				t.Errorf("URLEncode() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCQLQuery_HasField(t *testing.T) {
	query := "ti=bluetooth AND pa=ericsson"
	q, err := ParseCQL(query)
	if err != nil {
		t.Fatalf("ParseCQL() error = %v", err)
	}

	tests := []struct {
		field string
		want  bool
	}{
		{"ti", true},
		{"pa", true},
		{"ab", false},
		{"in", false},
	}

	for _, tt := range tests {
		t.Run(tt.field, func(t *testing.T) {
			got := q.HasField(tt.field)
			if got != tt.want {
				t.Errorf("HasField(%q) = %v, want %v", tt.field, got, tt.want)
			}
		})
	}
}

func TestCQLQuery_GetFields(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		wantFields []string
	}{
		{
			name:       "Single field",
			query:      "ti=bluetooth",
			wantFields: []string{"ti"},
		},
		{
			name:       "Multiple fields",
			query:      "ti=bluetooth AND pa=ericsson",
			wantFields: []string{"ti", "pa"},
		},
		{
			name:       "Repeated field",
			query:      "ti=bluetooth OR ti=wireless",
			wantFields: []string{"ti"},
		},
		{
			name:       "Many fields",
			query:      "ti=5g AND ab=antenna AND pa=apple AND in=smith",
			wantFields: []string{"ti", "ab", "pa", "in"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, err := ParseCQL(tt.query)
			if err != nil {
				t.Fatalf("ParseCQL() error = %v", err)
			}

			got := q.GetFields()
			if len(got) != len(tt.wantFields) {
				t.Errorf("GetFields() returned %d fields, want %d", len(got), len(tt.wantFields))
			}

			// Check that all expected fields are present
			fieldMap := make(map[string]bool)
			for _, f := range got {
				fieldMap[f] = true
			}

			for _, want := range tt.wantFields {
				if !fieldMap[want] {
					t.Errorf("GetFields() missing field %q", want)
				}
			}
		})
	}
}

func TestTokenize(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		wantTokens []string
	}{
		{
			name:       "Simple",
			query:      "ti=bluetooth",
			wantTokens: []string{"ti", "=", "bluetooth"},
		},
		{
			name:       "With parentheses",
			query:      "(ti=5g)",
			wantTokens: []string{"(", "ti", "=", "5g", ")"},
		},
		{
			name:       "With quotes",
			query:      "pa=\"Apple Inc\"",
			wantTokens: []string{"pa", "=", "\"", "Apple Inc", "\""},
		},
		{
			name:       "Multiple spaces",
			query:      "ti=bluetooth  AND  pa=ericsson",
			wantTokens: []string{"ti", "=", "bluetooth", "AND", "pa", "=", "ericsson"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens := tokenize(tt.query)

			if len(tokens) != len(tt.wantTokens) {
				t.Errorf("tokenize() got %d tokens, want %d", len(tokens), len(tt.wantTokens))
				for i, token := range tokens {
					t.Logf("CQLToken %d: %q", i, token.Value)
				}
				return
			}

			for i, want := range tt.wantTokens {
				if tokens[i].Value != want {
					t.Errorf("token %d: got %q, want %q", i, tokens[i].Value, want)
				}
			}
		})
	}
}

func TestGetFieldDescription(t *testing.T) {
	tests := []struct {
		field string
		want  string
	}{
		{"ti", "title"},
		{"ab", "abstract"},
		{"pa", "applicant name"},
		{"in", "inventor name"},
		{"pn", "publication number"},
		{"invalid", ""},
	}

	for _, tt := range tests {
		t.Run(tt.field, func(t *testing.T) {
			got := GetFieldDescription(tt.field)
			if got != tt.want {
				t.Errorf("GetFieldDescription(%q) = %q, want %q", tt.field, got, tt.want)
			}
		})
	}
}

func TestGetValidFields(t *testing.T) {
	fields := GetValidFields()
	fieldMap := make(map[string]bool)
	for _, f := range fields {
		fieldMap[f] = true
	}

	// Known-valid fields must be present.
	for _, known := range []string{"ti", "ab", "ta", "txt", "pa", "in", "ia", "pn", "ap", "pr", "num", "pd", "prd", "ic", "cpc", "ct", "rf"} {
		if !fieldMap[known] {
			t.Errorf("GetValidFields() missing valid field %q", known)
		}
	}
	// Fields the live OPS API rejects (CLIENT.InvalidIndex) must NOT be advertised.
	for _, invalid := range []string{"ad", "de", "ep", "pc", "ecla", "ctc"} {
		if fieldMap[invalid] {
			t.Errorf("GetValidFields() advertises API-invalid field %q", invalid)
		}
	}
}

func TestComplexQueries(t *testing.T) {
	tests := []struct {
		name      string
		query     string
		wantValid bool
	}{
		{
			name:      "Complex patent search",
			query:     "(ti=5g OR ab=5g) AND (pa=apple OR pa=samsung) AND pd>=20200101",
			wantValid: true,
		},
		{
			name:      "Classification with wildcards",
			query:     "ic=H04W AND cpc=H04W84/18 AND pa=ericsson",
			wantValid: true,
		},
		{
			name:      "Multi-level nesting",
			query:     "((ti=wireless OR ti=radio) AND (pa=qualcomm OR pa=broadcom)) OR (ic=H04B AND in=smith)",
			wantValid: true,
		},
		{
			name:      "Date range query",
			query:     "pd>=20200101 AND pd<=20201231 AND ic=G06F",
			wantValid: true,
		},
		{
			name:      "Inventor and applicant",
			query:     "(in=smith OR in=jones) AND pa=\"Microsoft Corporation\"",
			wantValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, err := ParseCQL(tt.query)
			if err != nil {
				t.Fatalf("ParseCQL() error = %v", err)
			}

			if q.Valid != tt.wantValid {
				t.Errorf("ParseCQL() Valid = %v, want %v", q.Valid, tt.wantValid)
				if len(q.Errors) > 0 {
					t.Errorf("Validation errors: %v", q.Errors)
				}
			}
		})
	}
}

func TestValidate_TypedFailureKinds(t *testing.T) {
	cases := []struct {
		name     string
		query    string
		wantKind CQLValidationKind
	}{
		{"unknown field", "badfield=value", CQLValidationUnknownField},
		{"unclosed parens", "(ti=bluetooth", CQLValidationUnclosedParens},
		{"unmatched closing paren", "ti=bluetooth)", CQLValidationUnmatchedParens},
		// Multiple tokens with no field=value structure (e.g. a free-text
		// "X AND Y" query, or USPTO-style "field:value field:value").
		{"no field=value pattern", "foo bar baz", CQLValidationNoFieldPattern},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			q, err := ParseCQL(tc.query)
			if err != nil {
				t.Fatalf("ParseCQL returned hard error: %v", err)
			}
			valErr := q.Validate()
			if valErr == nil {
				t.Fatalf("Validate returned nil, expected an error")
			}
			var typed *CQLValidationError
			if !errors.As(valErr, &typed) {
				t.Fatalf("Validate did not return *CQLValidationError; got %T (%v)", valErr, valErr)
			}
			first := typed.First()
			if first == nil {
				t.Fatalf("CQLValidationError.First() returned nil")
			}
			if first.Kind != tc.wantKind {
				t.Errorf("first failure Kind = %q, want %q (message=%q)", first.Kind, tc.wantKind, first.Message)
			}
			// Backward-compat: q.Errors must still be populated in parallel.
			if len(q.Errors) != len(typed.Failures) {
				t.Errorf("len(q.Errors)=%d, len(Failures)=%d; should be equal", len(q.Errors), len(typed.Failures))
			}
		})
	}
}

func TestValidate_UnknownFieldCarriesFieldName(t *testing.T) {
	q, err := ParseCQL("nosuchfield=value")
	if err != nil {
		t.Fatalf("ParseCQL: %v", err)
	}
	valErr := q.Validate()
	var typed *CQLValidationError
	if !errors.As(valErr, &typed) {
		t.Fatalf("not a *CQLValidationError")
	}
	f := typed.First()
	if f == nil || f.Kind != CQLValidationUnknownField {
		t.Fatalf("expected unknown_field, got %+v", f)
	}
	if f.Field != "nosuchfield" {
		t.Errorf("Field = %q, want %q", f.Field, "nosuchfield")
	}
}
