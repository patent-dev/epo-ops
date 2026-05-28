package cql

import (
	"fmt"
	"net/url"
	"strings"
	"unicode"
)

// CQLQuery represents a parsed CQL query.
type CQLQuery struct {
	// Raw is the original query string
	Raw string

	// Tokens is the parsed token stream
	Tokens []CQLToken

	// Valid indicates whether the query passed validation
	Valid bool

	// Errors contains any validation errors as plain strings.
	// Retained for backward compatibility; new code should consult
	// Failures instead which preserves the structured failure kind.
	Errors []string

	// Failures lists structured validation failures from the parser.
	// One entry per Errors entry; same order.
	Failures []*CQLValidationFailure
}

// CQLValidationKind classifies a single CQL validation failure so callers
// can route on the failure mode without inspecting prose. Stable across
// minor releases; new constants may be added but existing values do not
// change meaning.
type CQLValidationKind string

const (
	// CQLValidationUnknownField - the query referenced a field name that
	// is not in the EPO field list (ti, ab, pa, in, pn, ic, cpc, pd, ad).
	CQLValidationUnknownField CQLValidationKind = "unknown_field"
	// CQLValidationUnclosedParens - more '(' than ')' in the query.
	CQLValidationUnclosedParens CQLValidationKind = "unclosed_parens"
	// CQLValidationUnmatchedParens - a ')' appeared without a matching '('.
	CQLValidationUnmatchedParens CQLValidationKind = "unmatched_parens"
	// CQLValidationEmptyQuery - the query tokenized to nothing.
	CQLValidationEmptyQuery CQLValidationKind = "empty_query"
	// CQLValidationNoFieldPattern - tokens present but no field=value
	// pattern detected. Often a sign the caller used another provider's
	// grammar (e.g. USPTO field:value syntax) on the EPO endpoint.
	CQLValidationNoFieldPattern CQLValidationKind = "no_field_pattern"
)

// CQLValidationFailure describes a single CQL validation problem.
type CQLValidationFailure struct {
	Kind    CQLValidationKind
	Field   string // offending field name when Kind == CQLValidationUnknownField
	Pos     int    // position in the raw query, when available; 0 otherwise
	Message string // human-readable description, used as Error()
}

func (f *CQLValidationFailure) Error() string {
	return f.Message
}

// CQLValidationError is the error type returned by CQLQuery.Validate when
// one or more validation checks failed. Callers can errors.As into this
// type and inspect Failures to dispatch on the specific Kind(s).
type CQLValidationError struct {
	Failures []*CQLValidationFailure
}

func (e *CQLValidationError) Error() string {
	if len(e.Failures) == 1 {
		return "CQL validation error: " + e.Failures[0].Message
	}
	msgs := make([]string, len(e.Failures))
	for i, f := range e.Failures {
		msgs[i] = f.Message
	}
	return "CQL validation errors: " + strings.Join(msgs, "; ")
}

// First returns the first validation failure, or nil if there are none.
// Useful for picking a single dominant failure kind when classifying.
func (e *CQLValidationError) First() *CQLValidationFailure {
	if len(e.Failures) == 0 {
		return nil
	}
	return e.Failures[0]
}

// CQLToken represents a token in a CQL query.
type CQLToken struct {
	Type  TokenType
	Value string
	Pos   int // Position in original string
}

// TokenType represents the type of a CQL token.
type TokenType int

const (
	TokenField TokenType = iota
	TokenOperator
	TokenValue
	TokenEquals
	TokenLParen
	TokenRParen
	TokenQuote
	TokenWhitespace
	TokenUnknown
)

// String returns a string representation of a token type.
func (t TokenType) String() string {
	switch t {
	case TokenField:
		return "FIELD"
	case TokenOperator:
		return "OPERATOR"
	case TokenValue:
		return "VALUE"
	case TokenEquals:
		return "EQUALS"
	case TokenLParen:
		return "LPAREN"
	case TokenRParen:
		return "RPAREN"
	case TokenQuote:
		return "QUOTE"
	case TokenWhitespace:
		return "WHITESPACE"
	default:
		return "UNKNOWN"
	}
}

// ParseCQL parses a CQL query string and returns a CQLQuery object.
//
// Example queries:
//   - "ti=bluetooth"
//   - "ti=bluetooth AND pa=ericsson"
//   - "(ti=5g OR ab=5g) AND pd>=20200101"
//   - "pa=\"Apple Inc\" AND ic=H04W"
//
// Returns:
//   - *CQLQuery: The parsed query with tokens and validation status
//   - error: An error if the query is completely invalid or empty
func ParseCQL(query string) (*CQLQuery, error) {
	if strings.TrimSpace(query) == "" {
		return nil, fmt.Errorf("CQL query cannot be empty")
	}

	q := &CQLQuery{
		Raw:    query,
		Tokens: tokenize(query),
		Valid:  true,
		Errors: []string{},
	}

	// Validate the parsed query
	q.validate()

	return q, nil
}

// tokenize splits a CQL query into tokens.
// This is a simplified tokenizer that handles basic CQL syntax.
func tokenize(query string) []CQLToken {
	var tokens []CQLToken
	var current strings.Builder
	var inQuotes bool
	var pos int

	for i := 0; i < len(query); i++ {
		ch := rune(query[i])

		switch {
		case ch == '"':
			if current.Len() > 0 {
				tokens = append(tokens, CQLToken{
					Type:  classifyToken(current.String()),
					Value: current.String(),
					Pos:   pos,
				})
				current.Reset()
			}
			tokens = append(tokens, CQLToken{Type: TokenQuote, Value: "\"", Pos: i})
			inQuotes = !inQuotes
			pos = i + 1

		case (ch == '=' || ch == '<' || ch == '>') && !inQuotes:
			if current.Len() > 0 {
				tokens = append(tokens, CQLToken{
					Type:  TokenField,
					Value: current.String(),
					Pos:   pos,
				})
				current.Reset()
			}

			// Check for compound operators (>=, <=, <>)
			if (ch == '>' || ch == '<') && i+1 < len(query) {
				nextCh := rune(query[i+1])
				if nextCh == '=' || (ch == '<' && nextCh == '>') {
					tokens = append(tokens, CQLToken{Type: TokenEquals, Value: string([]rune{ch, nextCh}), Pos: i})
					i++ // Skip the next character
					pos = i + 1
					continue
				}
			}

			tokens = append(tokens, CQLToken{Type: TokenEquals, Value: string(ch), Pos: i})
			pos = i + 1

		case ch == '(' && !inQuotes:
			if current.Len() > 0 {
				tokens = append(tokens, CQLToken{
					Type:  classifyToken(current.String()),
					Value: current.String(),
					Pos:   pos,
				})
				current.Reset()
			}
			tokens = append(tokens, CQLToken{Type: TokenLParen, Value: "(", Pos: i})
			pos = i + 1

		case ch == ')' && !inQuotes:
			if current.Len() > 0 {
				tokens = append(tokens, CQLToken{
					Type:  classifyToken(current.String()),
					Value: current.String(),
					Pos:   pos,
				})
				current.Reset()
			}
			tokens = append(tokens, CQLToken{Type: TokenRParen, Value: ")", Pos: i})
			pos = i + 1

		case unicode.IsSpace(ch) && !inQuotes:
			if current.Len() > 0 {
				tokens = append(tokens, CQLToken{
					Type:  classifyToken(current.String()),
					Value: current.String(),
					Pos:   pos,
				})
				current.Reset()
			}
			pos = i + 1

		default:
			if current.Len() == 0 {
				pos = i
			}
			current.WriteByte(byte(ch))
		}
	}

	// Add final token if any
	if current.Len() > 0 {
		tokens = append(tokens, CQLToken{
			Type:  classifyToken(current.String()),
			Value: current.String(),
			Pos:   pos,
		})
	}

	return tokens
}

// classifyToken determines the type of a token based on its value.
func classifyToken(value string) TokenType {
	// Check if it's an operator
	if IsValidOperator(value) || IsValidOperator(strings.ToUpper(value)) {
		return TokenOperator
	}

	// Everything else is a value (fields are identified contextually)
	return TokenValue
}

// validate performs validation checks on the parsed CQL query.
func (q *CQLQuery) validate() {
	q.checkBracketMatching()
	q.checkFieldNames()
	q.checkQueryStructure()

	if len(q.Errors) > 0 {
		q.Valid = false
	}
}

// addFailure records both the structured failure and a parallel string in
// the legacy Errors slice so existing callers that inspected q.Errors keep
// working.
func (q *CQLQuery) addFailure(f *CQLValidationFailure) {
	q.Failures = append(q.Failures, f)
	q.Errors = append(q.Errors, f.Message)
}

// checkBracketMatching verifies that parentheses are properly matched.
func (q *CQLQuery) checkBracketMatching() {
	depth := 0
	for _, token := range q.Tokens {
		switch token.Type {
		case TokenLParen:
			depth++
		case TokenRParen:
			depth--
			if depth < 0 {
				q.addFailure(&CQLValidationFailure{
					Kind:    CQLValidationUnmatchedParens,
					Pos:     token.Pos,
					Message: fmt.Sprintf("unmatched closing parenthesis at position %d", token.Pos),
				})
				return
			}
		}
	}

	if depth > 0 {
		q.addFailure(&CQLValidationFailure{
			Kind:    CQLValidationUnclosedParens,
			Message: fmt.Sprintf("unclosed parentheses: %d opening parenthesis(es) without matching closing", depth),
		})
	}
}

// checkFieldNames validates that all field names are recognized EPO fields.
func (q *CQLQuery) checkFieldNames() {
	for i, token := range q.Tokens {
		// Check if this is a field name (token before '=')
		if i+1 < len(q.Tokens) && q.Tokens[i+1].Type == TokenEquals {
			if !IsValidField(token.Value) {
				q.addFailure(&CQLValidationFailure{
					Kind:  CQLValidationUnknownField,
					Field: token.Value,
					Pos:   token.Pos,
					Message: fmt.Sprintf(
						"invalid field '%s' at position %d (valid fields: %s)",
						token.Value,
						token.Pos,
						strings.Join([]string{"ti", "ab", "pa", "in", "pn", "ic", "cpc", "pd", "ad"}, ", "),
					),
				})
			}
		}
	}
}

// checkQueryStructure validates the overall structure of the query.
func (q *CQLQuery) checkQueryStructure() {
	if len(q.Tokens) == 0 {
		q.addFailure(&CQLValidationFailure{
			Kind:    CQLValidationEmptyQuery,
			Message: "query has no tokens",
		})
		return
	}

	// Check for field=value pattern (any token followed by = and another token)
	hasValidPattern := false
	for i := 0; i < len(q.Tokens)-2; i++ {
		if q.Tokens[i+1].Type == TokenEquals {
			// Found an equals sign, this is a field=value pattern
			hasValidPattern = true
			break
		}
	}

	if !hasValidPattern {
		// Check if it's just a simple value (which is valid)
		if len(q.Tokens) == 1 && q.Tokens[0].Type == TokenValue {
			hasValidPattern = true
		}
	}

	if !hasValidPattern && len(q.Errors) == 0 {
		q.addFailure(&CQLValidationFailure{
			Kind:    CQLValidationNoFieldPattern,
			Message: "query must contain at least one field=value pattern or search term",
		})
	}
}

// Validate checks if the CQL query is valid.
// Returns *CQLValidationError when one or more checks failed; callers can
// errors.As into it to dispatch on the specific failure Kind. The error
// string format is unchanged from earlier versions.
func (q *CQLQuery) Validate() error {
	if q.Valid {
		return nil
	}
	return &CQLValidationError{Failures: q.Failures}
}

// URLEncode returns the CQL query properly URL-encoded for use in API requests.
func (q *CQLQuery) URLEncode() string {
	return url.QueryEscape(q.Raw)
}

// String returns the raw query string.
func (q *CQLQuery) String() string {
	return q.Raw
}

// TokenCount returns the number of tokens in the parsed query.
func (q *CQLQuery) TokenCount() int {
	return len(q.Tokens)
}

// HasField checks if the query contains a specific field.
func (q *CQLQuery) HasField(field string) bool {
	for i, token := range q.Tokens {
		if token.Value == field && i+1 < len(q.Tokens) && q.Tokens[i+1].Type == TokenEquals {
			return true
		}
	}
	return false
}

// GetFields returns all unique fields used in the query.
func (q *CQLQuery) GetFields() []string {
	fields := make(map[string]bool)
	for i, token := range q.Tokens {
		if i+1 < len(q.Tokens) && q.Tokens[i+1].Type == TokenEquals {
			fields[token.Value] = true
		}
	}

	result := make([]string, 0, len(fields))
	for field := range fields {
		result = append(result, field)
	}
	return result
}
