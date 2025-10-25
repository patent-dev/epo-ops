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

	// Errors contains any validation errors
	Errors []string
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

// checkBracketMatching verifies that parentheses are properly matched.
func (q *CQLQuery) checkBracketMatching() {
	depth := 0
	for _, token := range q.Tokens {
		if token.Type == TokenLParen {
			depth++
		} else if token.Type == TokenRParen {
			depth--
			if depth < 0 {
				q.Errors = append(q.Errors, fmt.Sprintf("unmatched closing parenthesis at position %d", token.Pos))
				return
			}
		}
	}

	if depth > 0 {
		q.Errors = append(q.Errors, fmt.Sprintf("unclosed parentheses: %d opening parenthesis(es) without matching closing", depth))
	}
}

// checkFieldNames validates that all field names are recognized EPO fields.
func (q *CQLQuery) checkFieldNames() {
	for i, token := range q.Tokens {
		// Check if this is a field name (token before '=')
		if i+1 < len(q.Tokens) && q.Tokens[i+1].Type == TokenEquals {
			if !IsValidField(token.Value) {
				q.Errors = append(q.Errors, fmt.Sprintf(
					"invalid field '%s' at position %d (valid fields: %s)",
					token.Value,
					token.Pos,
					strings.Join([]string{"ti", "ab", "pa", "in", "pn", "ic", "cpc", "pd", "ad"}, ", "),
				))
			}
		}
	}
}

// checkQueryStructure validates the overall structure of the query.
func (q *CQLQuery) checkQueryStructure() {
	if len(q.Tokens) == 0 {
		q.Errors = append(q.Errors, "query has no tokens")
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
		q.Errors = append(q.Errors, "query must contain at least one field=value pattern or search term")
	}
}

// Validate checks if the CQL query is valid.
// Returns an error with details if the query is invalid.
func (q *CQLQuery) Validate() error {
	if q.Valid {
		return nil
	}

	if len(q.Errors) == 1 {
		return fmt.Errorf("CQL validation error: %s", q.Errors[0])
	}

	return fmt.Errorf("CQL validation errors: %s", strings.Join(q.Errors, "; "))
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
