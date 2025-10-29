package epo_ops

import (
	"context"
	"fmt"
	"net/http"

	"github.com/patent-dev/epo-ops/generated"
)

// Published Data Service - Bibliographic data, claims, descriptions, abstracts, and fulltext.
//
// This file contains all methods for retrieving published patent data including:
//   - Bibliographic data (GetBiblio)
//   - Claims (GetClaims)
//   - Descriptions (GetDescription)
//   - Abstracts (GetAbstract)
//   - Fulltext (GetFulltext)
//   - Equivalents (GetPublishedEquivalents)
//
// Each method has both a parsed version (returns Go structs) and a Raw version (returns XML string).

// GetBiblio retrieves and parses bibliographic data for a patent.
//
// Parameters:
//   - refType: Reference type (e.g., RefTypePublication, RefTypeApplication, RefTypePriority)
//   - format: Number format (e.g., FormatDocDB, FormatEPODOC)
//   - number: Patent number (e.g., "EP1000000B1")
//
// Returns parsed bibliographic data. For raw XML, use GetBiblioRaw().
func (c *Client) GetBiblio(ctx context.Context, refType, format, number string) (*BiblioData, error) {
	xml, err := c.GetBiblioRaw(ctx, refType, format, number)
	if err != nil {
		return nil, err
	}
	return ParseBiblio(xml)
}

// GetBiblioRaw retrieves bibliographic data for a patent as raw XML.
//
// Parameters:
//   - refType: Reference type (e.g., RefTypePublication, RefTypeApplication, RefTypePriority)
//   - format: Number format (e.g., FormatDocDB, FormatEPODOC)
//   - number: Patent number (e.g., "EP1000000B1")
//
// Returns the bibliographic data as an XML string.
func (c *Client) GetBiblioRaw(ctx context.Context, refType, format, number string) (string, error) {
	if err := ValidateRefType(refType); err != nil {
		return "", err
	}
	if err := ValidateFormat(format, number); err != nil {
		return "", err
	}
	return c.makeRequest(ctx, func() (*http.Response, error) {
		return c.generated.PublishedDataRetrieval(ctx,
			generated.PublishedDataRetrievalParamsType(refType),
			generated.PublishedDataRetrievalParamsFormat(format),
			number)
	})
}

// GetClaims retrieves and parses claims for a patent.
//
// Parameters:
//   - refType: Reference type (e.g., RefTypePublication, RefTypeApplication, RefTypePriority)
//   - format: Number format (e.g., FormatDocDB, FormatEPODOC)
//   - number: Patent number (e.g., "EP1000000B1")
//
// Returns parsed claims data. For raw XML, use GetClaimsRaw().
func (c *Client) GetClaims(ctx context.Context, refType, format, number string) (*ClaimsData, error) {
	xml, err := c.GetClaimsRaw(ctx, refType, format, number)
	if err != nil {
		return nil, err
	}
	return ParseClaims(xml)
}

// GetClaimsRaw retrieves claims for a patent as raw XML.
//
// Parameters:
//   - refType: Reference type (e.g., RefTypePublication, RefTypeApplication, RefTypePriority)
//   - format: Number format (e.g., FormatDocDB, FormatEPODOC)
//   - number: Patent number (e.g., "EP1000000B1")
//
// Returns the claims as an XML string.
func (c *Client) GetClaimsRaw(ctx context.Context, refType, format, number string) (string, error) {
	if err := ValidateRefType(refType); err != nil {
		return "", err
	}
	if err := ValidateFormat(format, number); err != nil {
		return "", err
	}
	return c.makeRequest(ctx, func() (*http.Response, error) {
		return c.generated.PublishedDataClaimsRetrievalService(ctx,
			generated.PublishedDataClaimsRetrievalServiceParamsType(refType),
			generated.PublishedDataClaimsRetrievalServiceParamsFormat(format),
			number)
	})
}

// GetDescription retrieves the description for a patent.
//
// Parameters:
//   - refType: Reference type (e.g., "publication", "application", "priority")
//   - format: Number format (e.g., "docdb", "epodoc")
//   - number: Patent number (e.g., "EP1000000")
//
// Returns the description as an XML string.
func (c *Client) GetDescription(ctx context.Context, refType, format, number string) (*DescriptionData, error) {
	xmlData, err := c.GetDescriptionRaw(ctx, refType, format, number)
	if err != nil {
		return nil, err
	}
	return ParseDescription(xmlData)
}

// GetDescriptionRaw retrieves patent description as raw XML.
// For parsed data, use GetDescription() instead.
func (c *Client) GetDescriptionRaw(ctx context.Context, refType, format, number string) (string, error) {
	if err := ValidateRefType(refType); err != nil {
		return "", err
	}
	if err := ValidateFormat(format, number); err != nil {
		return "", err
	}
	return c.makeRequest(ctx, func() (*http.Response, error) {
		return c.generated.PublishedDataDescriptionRetrievalService(ctx,
			generated.PublishedDataDescriptionRetrievalServiceParamsType(refType),
			generated.PublishedDataDescriptionRetrievalServiceParamsFormat(format),
			number)
	})
}

// GetAbstract retrieves and parses the abstract for a patent.
//
// Parameters:
//   - refType: Reference type (e.g., RefTypePublication, RefTypeApplication, RefTypePriority)
//   - format: Number format (e.g., FormatDocDB, FormatEPODOC)
//   - number: Patent number (e.g., "EP1000000B1")
//
// Returns parsed abstract data. For raw XML, use GetAbstractRaw().
func (c *Client) GetAbstract(ctx context.Context, refType, format, number string) (*AbstractData, error) {
	xml, err := c.GetAbstractRaw(ctx, refType, format, number)
	if err != nil {
		return nil, err
	}
	return ParseAbstract(xml)
}

// GetAbstractRaw retrieves the abstract for a patent as raw XML.
//
// Parameters:
//   - refType: Reference type (e.g., RefTypePublication, RefTypeApplication, RefTypePriority)
//   - format: Number format (e.g., FormatDocDB, FormatEPODOC)
//   - number: Patent number (e.g., "EP1000000B1")
//
// Returns the abstract as an XML string.
func (c *Client) GetAbstractRaw(ctx context.Context, refType, format, number string) (string, error) {
	if err := ValidateRefType(refType); err != nil {
		return "", err
	}
	if err := ValidateFormat(format, number); err != nil {
		return "", err
	}
	return c.makeRequest(ctx, func() (*http.Response, error) {
		return c.generated.PublishedDataAbstractService(ctx,
			generated.PublishedDataAbstractServiceParamsType(refType),
			generated.PublishedDataAbstractServiceParamsFormat(format),
			number)
	})
}

// GetFulltext retrieves the full text (biblio, abstract, description, claims) for a patent.
//
// Parameters:
//   - refType: Reference type (e.g., "publication", "application", "priority")
//   - format: Number format (e.g., "docdb", "epodoc")
//   - number: Patent number (e.g., "EP1000000")
//
// Returns parsed fulltext data including biblio, abstract, description, and claims.
func (c *Client) GetFulltext(ctx context.Context, refType, format, number string) (*FulltextData, error) {
	xmlData, err := c.GetFulltextRaw(ctx, refType, format, number)
	if err != nil {
		return nil, err
	}
	return ParseFulltext(xmlData)
}

// GetFulltextRaw retrieves full text as raw XML.
// For parsed data, use GetFulltext() instead.
func (c *Client) GetFulltextRaw(ctx context.Context, refType, format, number string) (string, error) {
	if err := ValidateRefType(refType); err != nil {
		return "", err
	}
	if err := ValidateFormat(format, number); err != nil {
		return "", err
	}
	return c.makeRequest(ctx, func() (*http.Response, error) {
		return c.generated.PublishedDataFulltextInquiryService(ctx,
			generated.PublishedDataFulltextInquiryServiceParamsType(refType),
			generated.PublishedDataFulltextInquiryServiceParamsFormat(format),
			number)
	})
}

// GetFullCycleMultiple retrieves full cycle data for multiple patents (bulk operation).
// Uses POST endpoint for efficient batch retrieval of up to 100 patents in one request.
//
// Full cycle data includes the complete publication history and bibliographic evolution
// of a patent across different publication stages (e.g., A1, A2, B1, B2).
//
// Parameters:
//   - refType: Reference type (e.g., RefTypePublication, RefTypeApplication, RefTypePriority)
//   - format: Number format (e.g., FormatDocDB, FormatEPODOC)
//   - numbers: Slice of patent numbers (max 100)
//
// Returns XML containing full cycle data for all requested patents.
func (c *Client) GetFullCycleMultiple(ctx context.Context, refType, format string, numbers []string) (string, error) {
	if err := ValidateRefType(refType); err != nil {
		return "", err
	}

	if len(numbers) == 0 {
		return "", &ValidationError{
			Field:   "numbers",
			Message: "at least one patent number required",
		}
	}

	if len(numbers) > 100 {
		return "", &ValidationError{
			Field:   "numbers",
			Value:   fmt.Sprintf("%d", len(numbers)),
			Message: "maximum 100 patent numbers per request",
		}
	}

	// Validate each patent number
	for i, number := range numbers {
		if err := ValidateFormat(format, number); err != nil {
			return "", fmt.Errorf("numbers[%d]: %w", i, err)
		}
	}

	// Use generated POST method
	body := formatBulkBody(numbers)
	return c.makeRequest(ctx, func() (*http.Response, error) {
		return c.generated.PublishedDataFullCycleServicePOSTWithTextBody(ctx,
			generated.PublishedDataFullCycleServicePOSTParamsType(refType),
			generated.PublishedDataFullCycleServicePOSTParamsFormat(format),
			body)
	})
}

// GetBiblioMultiple retrieves bibliographic data for multiple patents (bulk operation).
// Uses POST endpoint for efficient batch retrieval of up to 100 patents in one request.
//
// Parameters:
//   - refType: Reference type (e.g., RefTypePublication, RefTypeApplication, RefTypePriority)
//   - format: Number format (e.g., FormatDocDB, FormatEPODOC)
//   - numbers: Slice of patent numbers (max 100)
//
// Returns XML string containing bibliographic data for all requested patents.
//
// Note: This method returns raw XML instead of parsed structs for historical reasons.
// Some *Multiple() methods return parsed structs (GetDescriptionMultiple, GetFulltextMultiple)
// while others return XML (GetBiblioMultiple, GetClaimsMultiple, GetAbstractMultiple).
// This inconsistency is technical debt to be addressed in a future major version.
// For now, callers can parse the XML using ParseBiblio() if needed.
func (c *Client) GetBiblioMultiple(ctx context.Context, refType, format string, numbers []string) (string, error) {
	if err := ValidateRefType(refType); err != nil {
		return "", err
	}

	if err := ValidateBulkNumbers(numbers, format); err != nil {
		return "", err
	}

	// Use generated POST method
	body := formatBulkBody(numbers)
	return c.makeRequest(ctx, func() (*http.Response, error) {
		return c.generated.PublishedDataRetrievalPOSTWithTextBody(ctx,
			generated.PublishedDataRetrievalPOSTParamsType(refType),
			generated.PublishedDataRetrievalPOSTParamsFormat(format),
			body)
	})
}

// GetClaimsMultiple retrieves claims for multiple patents (bulk operation).
// Uses POST endpoint for efficient batch retrieval of up to 100 patents in one request.
//
// Parameters:
//   - refType: Reference type (e.g., RefTypePublication, RefTypeApplication, RefTypePriority)
//   - format: Number format (e.g., FormatDocDB, FormatEPODOC)
//   - numbers: Slice of patent numbers (max 100)
//
// Returns XML containing claims for all requested patents.
//
// Note: Returns raw XML. See GetBiblioMultiple() documentation for notes on return type inconsistency.
func (c *Client) GetClaimsMultiple(ctx context.Context, refType, format string, numbers []string) (string, error) {
	if err := ValidateRefType(refType); err != nil {
		return "", err
	}

	if err := ValidateBulkNumbers(numbers, format); err != nil {
		return "", err
	}

	// Use generated POST method
	body := formatBulkBody(numbers)
	return c.makeRequest(ctx, func() (*http.Response, error) {
		return c.generated.PublishedClaimsRetrievalServicePOSTWithTextBody(ctx,
			generated.PublishedClaimsRetrievalServicePOSTParamsType(refType),
			generated.PublishedClaimsRetrievalServicePOSTParamsFormat(format),
			body)
	})
}

// GetDescriptionMultiple retrieves descriptions for multiple patents (bulk operation).
//
// Parameters:
//   - refType: Reference type (e.g., RefTypePublication, RefTypeApplication, RefTypePriority)
//   - format: Number format (e.g., FormatDocDB, FormatEPODOC)
//   - numbers: Slice of patent numbers (max 100)
//
// Returns parsed description data including paragraphs for all requested patents.
func (c *Client) GetDescriptionMultiple(ctx context.Context, refType, format string, numbers []string) (*DescriptionData, error) {
	if err := ValidateRefType(refType); err != nil {
		return nil, err
	}

	if err := ValidateBulkNumbers(numbers, format); err != nil {
		return nil, err
	}

	// Use generated POST method
	body := formatBulkBody(numbers)
	xmlData, err := c.makeRequest(ctx, func() (*http.Response, error) {
		return c.generated.PublishedDataDescriptionRetrievalServicePOSTWithTextBody(ctx,
			generated.PublishedDataDescriptionRetrievalServicePOSTParamsType(refType),
			generated.PublishedDataDescriptionRetrievalServicePOSTParamsFormat(format),
			body)
	})
	if err != nil {
		return nil, err
	}
	return ParseDescription(xmlData)
}

// GetAbstractMultiple retrieves abstracts for multiple patents (bulk operation).
// Uses POST endpoint for efficient batch retrieval of up to 100 patents in one request.
//
// Parameters:
//   - refType: Reference type (e.g., RefTypePublication, RefTypeApplication, RefTypePriority)
//   - format: Number format (e.g., FormatDocDB, FormatEPODOC)
//   - numbers: Slice of patent numbers (max 100)
//
// Returns XML string containing abstracts for all requested patents.
//
// Note: Returns raw XML. See GetBiblioMultiple() documentation for notes on return type inconsistency.
func (c *Client) GetAbstractMultiple(ctx context.Context, refType, format string, numbers []string) (string, error) {
	if err := ValidateRefType(refType); err != nil {
		return "", err
	}

	if err := ValidateBulkNumbers(numbers, format); err != nil {
		return "", err
	}

	// Use generated POST method
	body := formatBulkBody(numbers)
	return c.makeRequest(ctx, func() (*http.Response, error) {
		return c.generated.PublishedDataAbstractServicePOSTWithTextBody(ctx,
			generated.PublishedDataAbstractServicePOSTParamsType(refType),
			generated.PublishedDataAbstractServicePOSTParamsFormat(format),
			body)
	})
}

// GetFulltextMultiple retrieves fulltext data for multiple patents (bulk operation).
//
// Parameters:
//   - refType: Reference type (e.g., RefTypePublication, RefTypeApplication, RefTypePriority)
//   - format: Number format (e.g., FormatDocDB, FormatEPODOC)
//   - numbers: Slice of patent numbers (max 100)
//
// Returns parsed fulltext data including biblio, abstract, description, and claims for all requested patents.
func (c *Client) GetFulltextMultiple(ctx context.Context, refType, format string, numbers []string) (*FulltextData, error) {
	if err := ValidateRefType(refType); err != nil {
		return nil, err
	}

	if err := ValidateBulkNumbers(numbers, format); err != nil {
		return nil, err
	}

	// Use generated POST method
	body := formatBulkBody(numbers)
	xmlData, err := c.makeRequest(ctx, func() (*http.Response, error) {
		return c.generated.PublishedDataFulltextInquiryServicePOSTWithTextBody(ctx,
			generated.PublishedDataFulltextInquiryServicePOSTParamsType(refType),
			generated.PublishedDataFulltextInquiryServicePOSTParamsFormat(format),
			body)
	})
	if err != nil {
		return nil, err
	}
	return ParseFulltext(xmlData)
}

// GetPublishedEquivalents retrieves equivalent publications for a patent (simple family).
//
// This returns the "simple family" - equivalent publications of the same invention
// (same priority claim). This is different from the INPADOC family which includes
// extended family members.
//
// Parameters:
//   - refType: Reference type (RefTypePublication, RefTypeApplication, or RefTypePriority)
//   - format: Number format ("epodoc" or "docdb")
//   - number: Patent number in specified format
//
// Returns XML or JSON with equivalent publications.
//
// Example:
//
//	equivalents, err := client.GetPublishedEquivalents(ctx, epo_ops.RefTypePublication, "epodoc", "EP1000000")
func (c *Client) GetPublishedEquivalents(ctx context.Context, refType, format, number string) (*EquivalentsData, error) {
	xmlData, err := c.GetPublishedEquivalentsRaw(ctx, refType, format, number)
	if err != nil {
		return nil, err
	}
	return ParseEquivalents(xmlData)
}

// GetPublishedEquivalentsRaw retrieves equivalent publications as raw XML.
// For parsed data, use GetPublishedEquivalents() instead.
func (c *Client) GetPublishedEquivalentsRaw(ctx context.Context, refType, format, number string) (string, error) {
	// Validate reference type
	if err := ValidateRefType(refType); err != nil {
		return "", err
	}

	// Validate format and number
	if err := ValidateFormat(format, number); err != nil {
		return "", err
	}

	return c.makeRequest(ctx, func() (*http.Response, error) {
		return c.generated.PublishedEquivalentsRetrievalService(ctx,
			generated.PublishedEquivalentsRetrievalServiceParamsType(refType),
			generated.PublishedEquivalentsRetrievalServiceParamsFormat(format),
			number)
	})
}

// GetPublishedEquivalentsMultiple retrieves equivalent publications for multiple patents.
//
// This method uses the POST endpoint to retrieve simple family data for multiple
// patent numbers in a single request.
//
// Parameters:
//   - refType: Reference type (RefTypePublication, RefTypeApplication, or RefTypePriority)
//   - format: Number format ("epodoc" or "docdb")
//   - numbers: Slice of patent numbers (max 100)
//
// Returns parsed equivalents data for all requested patents.
func (c *Client) GetPublishedEquivalentsMultiple(ctx context.Context, refType, format string, numbers []string) (*EquivalentsData, error) {
	if err := ValidateRefType(refType); err != nil {
		return nil, err
	}

	if err := ValidateBulkNumbers(numbers, format); err != nil {
		return nil, err
	}

	// Use generated POST method
	body := formatBulkBody(numbers)
	xmlData, err := c.makeRequest(ctx, func() (*http.Response, error) {
		return c.generated.PublishedEquivalentsRetrievalServicePOSTWithTextBody(ctx,
			generated.PublishedEquivalentsRetrievalServicePOSTParamsType(refType),
			generated.PublishedEquivalentsRetrievalServicePOSTParamsFormat(format),
			body)
	})
	if err != nil {
		return nil, err
	}
	return ParseEquivalents(xmlData)
}
