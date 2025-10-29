package epo_ops

import (
	"context"
	"net/http"

	"github.com/patent-dev/epo-ops/generated"
)

// Family Service - INPADOC patent family retrieval.
//
// This file contains all methods for retrieving INPADOC (International Patent Documentation)
// family data, which includes all patents related through priority claims.

// GetFamily retrieves the INPADOC patent family for a given patent.
//
// Parameters:
//   - refType: Reference type (e.g., "publication", "application", "priority")
//   - format: Number format (e.g., "docdb", "epodoc")
//   - number: Patent number (e.g., "EP1000000")
//
// Returns parsed family data containing all family members with their bibliographic details.
//
// INPADOC (International Patent Documentation) family includes all patents
// related through priority claims.
func (c *Client) GetFamily(ctx context.Context, refType, format, number string) (*FamilyData, error) {
	xmlData, err := c.GetFamilyRaw(ctx, refType, format, number)
	if err != nil {
		return nil, err
	}
	return ParseFamily(xmlData)
}

// GetFamilyRaw retrieves the INPADOC patent family as raw XML.
// For parsed data, use GetFamily() instead.
func (c *Client) GetFamilyRaw(ctx context.Context, refType, format, number string) (string, error) {
	if err := ValidateRefType(refType); err != nil {
		return "", err
	}
	if err := ValidateFormat(format, number); err != nil {
		return "", err
	}
	return c.makeRequest(ctx, func() (*http.Response, error) {
		return c.generated.INPADOCFamilyRetrievalService(ctx,
			generated.INPADOCFamilyRetrievalServiceParamsType(refType),
			generated.INPADOCFamilyRetrievalServiceParamsFormat(format),
			number)
	})
}

// GetFamilyWithBiblio retrieves the INPADOC patent family with bibliographic data.
//
// Parameters:
//   - refType: Reference type (e.g., "publication", "application", "priority")
//   - format: Number format (e.g., "docdb", "epodoc")
//   - number: Patent number (e.g., "EP1000000")
//
// Returns parsed family data with bibliographic details for all family members.
func (c *Client) GetFamilyWithBiblio(ctx context.Context, refType, format, number string) (*FamilyData, error) {
	if err := ValidateRefType(refType); err != nil {
		return nil, err
	}
	if err := ValidateFormat(format, number); err != nil {
		return nil, err
	}
	xmlData, err := c.makeRequest(ctx, func() (*http.Response, error) {
		return c.generated.INPADOCFamilyRetrievalServiceWithBiblio(ctx,
			generated.INPADOCFamilyRetrievalServiceWithBiblioParamsType(refType),
			generated.INPADOCFamilyRetrievalServiceWithBiblioParamsFormat(format),
			number)
	})
	if err != nil {
		return nil, err
	}
	return ParseFamily(xmlData)
}

// GetFamilyWithLegal retrieves the INPADOC patent family with legal status data.
//
// Parameters:
//   - refType: Reference type (e.g., "publication", "application", "priority")
//   - format: Number format (e.g., "docdb", "epodoc")
//   - number: Patent number (e.g., "EP1000000")
//
// Returns parsed family data with legal status events for all family members.
func (c *Client) GetFamilyWithLegal(ctx context.Context, refType, format, number string) (*FamilyData, error) {
	if err := ValidateRefType(refType); err != nil {
		return nil, err
	}
	if err := ValidateFormat(format, number); err != nil {
		return nil, err
	}
	xmlData, err := c.makeRequest(ctx, func() (*http.Response, error) {
		return c.generated.INPADOCFamilyRetrievalServiceWithLegal(ctx,
			generated.INPADOCFamilyRetrievalServiceWithLegalParamsType(refType),
			generated.INPADOCFamilyRetrievalServiceWithLegalParamsFormat(format),
			number)
	})
	if err != nil {
		return nil, err
	}
	return ParseFamily(xmlData)
}

// GetFamilyWithBiblioMultiple retrieves INPADOC patent family with bibliographic data for multiple patents.
//
// This method uses the POST endpoint to retrieve family data with bibliographic information
// for multiple patent numbers in a single request.
//
// Parameters:
//   - refType: Reference type (e.g., RefTypePublication, RefTypeApplication, RefTypePriority)
//   - format: Number format (e.g., FormatDocDB, FormatEPODOC)
//   - numbers: Slice of patent numbers (max 100)
//
// Returns parsed family data with bibliographic details for all requested patents.
func (c *Client) GetFamilyWithBiblioMultiple(ctx context.Context, refType, format string, numbers []string) (*FamilyData, error) {
	if err := ValidateRefType(refType); err != nil {
		return nil, err
	}

	if err := ValidateBulkNumbers(numbers, format); err != nil {
		return nil, err
	}

	// Use generated POST method
	body := formatBulkBody(numbers)
	xmlData, err := c.makeRequest(ctx, func() (*http.Response, error) {
		return c.generated.INPADOCFamilyRetrievalServiceWithBiblioPOSTWithTextBody(ctx,
			generated.INPADOCFamilyRetrievalServiceWithBiblioPOSTParamsType(refType),
			generated.INPADOCFamilyRetrievalServiceWithBiblioPOSTParamsFormat(format),
			body)
	})
	if err != nil {
		return nil, err
	}
	return ParseFamily(xmlData)
}

// GetFamilyWithLegalMultiple retrieves INPADOC patent family with legal status data for multiple patents.
//
// This method uses the POST endpoint to retrieve family data with legal status
// for multiple patent numbers in a single request.
//
// Parameters:
//   - refType: Reference type (e.g., RefTypePublication, RefTypeApplication, RefTypePriority)
//   - format: Number format (e.g., FormatDocDB, FormatEPODOC)
//   - numbers: Slice of patent numbers (max 100)
//
// Returns parsed family data with legal status events for all requested patents.
func (c *Client) GetFamilyWithLegalMultiple(ctx context.Context, refType, format string, numbers []string) (*FamilyData, error) {
	if err := ValidateRefType(refType); err != nil {
		return nil, err
	}

	if err := ValidateBulkNumbers(numbers, format); err != nil {
		return nil, err
	}

	// Use generated POST method
	body := formatBulkBody(numbers)
	xmlData, err := c.makeRequest(ctx, func() (*http.Response, error) {
		return c.generated.INPADOCFamilyRetrievalServiceWithLegalPOSTWithTextBody(ctx,
			generated.INPADOCFamilyRetrievalServiceWithLegalPOSTParamsType(refType),
			generated.INPADOCFamilyRetrievalServiceWithLegalPOSTParamsFormat(format),
			body)
	})
	if err != nil {
		return nil, err
	}
	return ParseFamily(xmlData)
}
