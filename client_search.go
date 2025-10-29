package epo_ops

import (
	"context"
	"net/http"

	"github.com/patent-dev/epo-ops/cql"
	"github.com/patent-dev/epo-ops/generated"
)

//
// This file contains methods for searching patents using CQL queries.

// Search performs a bibliographic search using CQL (Contextual Query Language).
//
// Parameters:
//   - query: CQL query string (e.g., "ti=plastic", "pa=Siemens and de")
//   - rangeStr: Optional range in format "1-25" (default: "1-25")
//
// Returns the search results as XML containing matching patents.
//
// Example queries:
//   - "ti=plastic" - Title contains "plastic"
//   - "pa=Siemens" - Applicant is Siemens
//   - "de" - Country code DE
//   - "ti=plastic and pa=Siemens" - Combined search
//
// See OPS documentation for full CQL syntax.
func (c *Client) Search(ctx context.Context, query string, rangeStr string) (*SearchResultData, error) {
	xmlData, err := c.SearchRaw(ctx, query, rangeStr)
	if err != nil {
		return nil, err
	}
	return ParseSearch(xmlData)
}

// SearchRaw performs a bibliographic search and returns raw XML.
// For parsed data, use Search() instead.
func (c *Client) SearchRaw(ctx context.Context, query string, rangeStr string) (string, error) {
	// Validate CQL query
	cqlQuery, err := cql.ParseCQL(query)
	if err != nil {
		return "", err
	}
	if err := cqlQuery.Validate(); err != nil {
		return "", err
	}

	if rangeStr == "" {
		rangeStr = "1-25"
	}

	params := &generated.PublishedDataKeywordsSearchWithoutConsituentsParams{
		Q:     query,
		Range: &rangeStr,
	}

	return c.makeRequest(ctx, func() (*http.Response, error) {
		return c.generated.PublishedDataKeywordsSearchWithoutConsituents(ctx, params)
	})
}

// SearchWithConstituent performs a bibliographic search with specific constituent.
//
// Parameters:
//   - constituent: The constituent to retrieve (e.g., "biblio", "abstract", "full-cycle")
//   - query: CQL query string
//   - rangeStr: Optional range in format "1-25"
//
// Returns parsed search results with the requested constituent data.
func (c *Client) SearchWithConstituent(ctx context.Context, constituent, query string, rangeStr string) (*SearchResultData, error) {
	// Validate CQL query
	cqlQuery, err := cql.ParseCQL(query)
	if err != nil {
		return nil, err
	}
	if err := cqlQuery.Validate(); err != nil {
		return nil, err
	}

	if rangeStr == "" {
		rangeStr = "1-25"
	}

	params := &generated.PublishedDataKeywordsSearchWithVariableConstituentsParams{
		Q:     query,
		Range: &rangeStr,
	}

	xmlData, err := c.makeRequest(ctx, func() (*http.Response, error) {
		return c.generated.PublishedDataKeywordsSearchWithVariableConstituents(ctx,
			generated.PublishedDataKeywordsSearchWithVariableConstituentsParamsConstituent(constituent),
			params)
	})
	if err != nil {
		return nil, err
	}
	return ParseSearch(xmlData)
}
