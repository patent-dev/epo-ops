package epo_ops

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/patent-dev/epo-ops/generated"
)

// Legal and Register Services - Legal status and EPO Register data.
//
// This file contains methods for retrieving legal status events and EPO Register information.
func (c *Client) GetLegal(ctx context.Context, refType, format, number string) (*LegalData, error) {
	xmlData, err := c.GetLegalRaw(ctx, refType, format, number)
	if err != nil {
		return nil, err
	}
	return ParseLegal(xmlData)
}

// GetLegalRaw retrieves legal status data as raw XML.
// For parsed data, use GetLegal() instead.
func (c *Client) GetLegalRaw(ctx context.Context, refType, format, number string) (string, error) {
	if err := ValidateRefType(refType); err != nil {
		return "", err
	}
	if err := ValidateFormat(format, number); err != nil {
		return "", err
	}
	return c.makeRequest(ctx, func() (*http.Response, error) {
		return c.generated.LegalDataRetrievalService(ctx,
			generated.LegalDataRetrievalServiceParamsType(refType),
			generated.LegalDataRetrievalServiceParamsFormat(format),
			number)
	})
}

// GetLegalMultiple retrieves legal status data for multiple patents (bulk operation).
// Uses POST endpoint for efficient batch retrieval of up to 100 patents in one request.
//
// Parameters:
//   - refType: Reference type (e.g., RefTypePublication, RefTypeApplication, RefTypePriority)
//   - format: Number format (e.g., FormatDocDB, FormatEPODOC)
//   - numbers: Slice of patent numbers (max 100)
//
// Returns parsed legal status data for all requested patents.
func (c *Client) GetLegalMultiple(ctx context.Context, refType, format string, numbers []string) (*LegalData, error) {
	if err := ValidateRefType(refType); err != nil {
		return nil, err
	}

	if err := ValidateBulkNumbers(numbers, format); err != nil {
		return nil, err
	}

	// Use generated POST method
	body := formatBulkBody(numbers)
	xmlData, err := c.makeRequest(ctx, func() (*http.Response, error) {
		return c.generated.LegalDataRetrievalServicePOSTWithTextBody(ctx,
			generated.LegalDataRetrievalServicePOSTParamsType(refType),
			generated.LegalDataRetrievalServicePOSTParamsFormat(format),
			body)
	})
	if err != nil {
		return nil, err
	}
	return ParseLegal(xmlData)
}

// GetRegisterBiblio retrieves bibliographic data from the EPO Register.
//
// Parameters:
//   - refType: Reference type (e.g., "publication", "application", "priority")
//   - format: Number format (e.g., "docdb", "epodoc")
//   - number: Patent number (e.g., "EP1000000")
//
// Returns EPO Register bibliographic data as XML.
//
// Note: The EPO Register contains more detailed and up-to-date information
// than the standard bibliographic service.
func (c *Client) GetRegisterBiblioRaw(ctx context.Context, refType, format, number string) (string, error) {
	if err := ValidateRefType(refType); err != nil {
		return "", err
	}
	// Note: Register endpoints accept both docdb format (EP.1000000.B1) and epodoc without kind (EP1000000)
	// even when format parameter is "epodoc", so we skip format validation here
	return c.makeRequest(ctx, func() (*http.Response, error) {
		return c.generated.RegisterRetrievalService(ctx,
			generated.RegisterRetrievalServiceParamsType(refType),
			generated.RegisterRetrievalServiceParamsFormat(format),
			number)
	})
}

// GetRegisterBiblioMultiple retrieves bibliographic data from the EPO Register for multiple patents.
// Uses POST endpoint for efficient batch retrieval of up to 100 patents in one request.
//
// Parameters:
//   - refType: Reference type (e.g., RefTypePublication, RefTypeApplication, RefTypePriority)
//   - format: Number format (e.g., FormatDocDB, FormatEPODOC)
//   - numbers: Slice of patent numbers (max 100)
//
// Returns XML containing EPO Register bibliographic data for all requested patents.
func (c *Client) GetRegisterBiblioMultipleRaw(ctx context.Context, refType, format string, numbers []string) (string, error) {
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

	// Use generated POST method
	body := formatBulkBody(numbers)
	return c.makeRequest(ctx, func() (*http.Response, error) {
		return c.generated.RegisterRetrievalServicePOSTWithTextBody(ctx,
			generated.RegisterRetrievalServicePOSTParamsType(refType),
			generated.RegisterRetrievalServicePOSTParamsFormat(format),
			body)
	})
}

// GetRegisterEvents retrieves procedural events from the EPO Register.
//
// Parameters:
//   - refType: Reference type (e.g., "publication", "application", "priority")
//   - format: Number format (e.g., "docdb", "epodoc")
//   - number: Patent number (e.g., "EP1000000")
//
// Returns EPO Register events as XML, including:
//   - Filing events
//   - Publication events
//   - Examination events
//   - Grant/refusal events
func (c *Client) GetRegisterEventsRaw(ctx context.Context, refType, format, number string) (string, error) {
	if err := ValidateRefType(refType); err != nil {
		return "", err
	}
	// Note: Register endpoints accept both docdb format (EP.1000000.B1) and epodoc without kind (EP1000000)
	// even when format parameter is "epodoc", so we skip format validation here
	return c.makeRequest(ctx, func() (*http.Response, error) {
		return c.generated.RegisterEventsService(ctx,
			generated.RegisterEventsServiceParamsType(refType),
			generated.RegisterEventsServiceParamsFormat(format),
			number)
	})
}

// GetRegisterEventsMultiple retrieves procedural events from the EPO Register for multiple patents.
// Uses POST endpoint for efficient batch retrieval of up to 100 patents in one request.
//
// Parameters:
//   - refType: Reference type (e.g., RefTypePublication, RefTypeApplication, RefTypePriority)
//   - format: Number format (e.g., FormatDocDB, FormatEPODOC)
//   - numbers: Slice of patent numbers (max 100)
//
// Returns XML containing EPO Register events for all requested patents.
func (c *Client) GetRegisterEventsMultipleRaw(ctx context.Context, refType, format string, numbers []string) (string, error) {
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

	// Use generated POST method
	body := formatBulkBody(numbers)
	return c.makeRequest(ctx, func() (*http.Response, error) {
		return c.generated.RegisterEventsServicePOSTWithTextBody(ctx,
			generated.RegisterEventsServicePOSTParamsType(refType),
			generated.RegisterEventsServicePOSTParamsFormat(format),
			body)
	})
}

// GetRegisterProceduralSteps retrieves procedural steps from the EPO Register.
//
// Procedural steps provide detailed information about the procedural history of a patent
// application, including milestones, deadlines, and administrative actions.
//
// Parameters:
//   - refType: Reference type ("publication" or "application")
//   - format: Number format ("epodoc" only)
//   - number: Patent number (e.g., "EP1000000")
//
// Returns XML containing:
//   - Procedural step history
//   - Step dates and descriptions
//   - Administrative actions
//   - Milestone information
//
// Example:
//
//	// Get procedural steps for a publication
//	steps, err := client.GetRegisterProceduralSteps(ctx, "publication", "epodoc", "EP1000000")
//	if err != nil {
//	    log.Fatal(err)
//	}
func (c *Client) GetRegisterProceduralStepsRaw(ctx context.Context, refType, format, number string) (string, error) {
	if err := ValidateRefType(refType); err != nil {
		return "", err
	}
	// Note: Register endpoints accept both docdb format (EP.1000000.B1) and epodoc without kind (EP1000000)
	// even when format parameter is "epodoc", so we skip format validation here

	// Convert to enum types
	var typeEnum generated.RegisterProceduralStepsServiceParamsType
	if refType == RefTypePublication {
		typeEnum = generated.RegisterProceduralStepsServiceParamsTypePublication
	} else {
		typeEnum = generated.RegisterProceduralStepsServiceParamsTypeApplication
	}

	return c.makeRequest(ctx, func() (*http.Response, error) {
		return c.generated.RegisterProceduralStepsService(ctx,
			typeEnum,
			generated.RegisterProceduralStepsServiceParamsFormatEpodoc,
			number)
	})
}

// GetRegisterProceduralStepsMultiple retrieves procedural steps for multiple patent numbers.
//
// This method uses the POST endpoint to retrieve procedural step data for multiple
// patent numbers in a single request.
//
// Parameters:
//   - refType: Reference type ("publication" or "application")
//   - format: Number format ("epodoc" only)
//   - numbers: Slice of patent numbers (max 100)
//
// Returns XML containing procedural steps for all requested patents.
//
// Example:
//
//	numbers := []string{"EP1000000", "EP1000001", "EP1000002"}
//	steps, err := client.GetRegisterProceduralStepsMultiple(ctx, "publication", "epodoc", numbers)
func (c *Client) GetRegisterProceduralStepsMultipleRaw(ctx context.Context, refType, format string, numbers []string) (string, error) {
	if len(numbers) == 0 {
		return "", &ConfigError{Message: "numbers list cannot be empty"}
	}
	if len(numbers) > 100 {
		return "", &ConfigError{Message: "maximum 100 numbers allowed per request"}
	}
	if err := ValidateRefType(refType); err != nil {
		return "", err
	}

	// Convert to enum types
	var typeEnum generated.RegisterProceduralStepsServicePOSTParamsType
	if refType == RefTypePublication {
		typeEnum = generated.RegisterProceduralStepsServicePOSTParamsTypePublication
	} else {
		typeEnum = generated.RegisterProceduralStepsServicePOSTParamsTypeApplication
	}

	// Build request body (newline-separated number list)
	body := strings.Join(numbers, "\n")

	return c.makeRequest(ctx, func() (*http.Response, error) {
		return c.generated.RegisterProceduralStepsServicePOSTWithTextBody(ctx,
			typeEnum,
			generated.RegisterProceduralStepsServicePOSTParamsFormatEpodoc,
			generated.RegisterProceduralStepsServicePOSTTextRequestBody(body))
	})
}

// GetRegisterUNIP retrieves unitary patent package (UPP) information from the EPO Register.
//
// Parameters:
//   - refType: Reference type (RefTypePublication or RefTypeApplication)
//   - format: Number format (must be "epodoc")
//   - number: Patent number in specified format
//
// Returns XML or JSON with unitary patent information including:
//   - Unified Patent Court opt-out status
//   - Unitary effect registration
//   - Participating member states
//
// Example:
//
//	unip, err := client.GetRegisterUNIP(ctx, epo_ops.RefTypePublication, "epodoc", "EP3000000")
func (c *Client) GetRegisterUNIPRaw(ctx context.Context, refType, format, number string) (string, error) {
	// Validate reference type
	if err := ValidateRefType(refType); err != nil {
		return "", err
	}

	// Note: Register endpoints accept both docdb format (EP.1000000.B1) and epodoc without kind (EP1000000)
	// even when format parameter is "epodoc", so we skip format validation here

	// Convert refType string to generated enum
	var typeEnum generated.RegisterUNIPServiceParamsType
	if refType == RefTypePublication {
		typeEnum = generated.RegisterUNIPServiceParamsTypePublication
	} else {
		typeEnum = generated.RegisterUNIPServiceParamsTypeApplication
	}

	return c.makeRequest(ctx, func() (*http.Response, error) {
		return c.generated.RegisterUNIPService(ctx,
			typeEnum,
			generated.RegisterUNIPServiceParamsFormatEpodoc,
			number)
	})
}

// GetRegisterUNIPMultiple retrieves unitary patent package (UPP) information for multiple patents.
//
// Parameters:
//   - refType: Reference type (RefTypePublication or RefTypeApplication)
//   - format: Number format (must be "epodoc")
//   - numbers: List of patent numbers (max 100)
//
// Returns XML or JSON with unitary patent information for all requested patents.
//
// Example:
//
//	numbers := []string{"EP3000000", "EP3000001"}
//	unip, err := client.GetRegisterUNIPMultiple(ctx, epo_ops.RefTypePublication, "epodoc", numbers)
func (c *Client) GetRegisterUNIPMultipleRaw(ctx context.Context, refType, format string, numbers []string) (string, error) {
	// Validate reference type
	if err := ValidateRefType(refType); err != nil {
		return "", err
	}

	// Validate format
	if format != "epodoc" {
		return "", &ConfigError{Message: "format must be 'epodoc'"}
	}

	// Validate numbers list
	if len(numbers) == 0 {
		return "", &ConfigError{Message: "numbers list cannot be empty"}
	}

	if len(numbers) > 100 {
		return "", &ConfigError{Message: "maximum 100 numbers allowed per request"}
	}

	// Convert refType string to generated enum
	var typeEnum generated.RegisterUNIPServicePOSTParamsType
	if refType == RefTypePublication {
		typeEnum = generated.RegisterUNIPServicePOSTParamsTypePublication
	} else {
		typeEnum = generated.RegisterUNIPServicePOSTParamsTypeApplication
	}

	// Build request body (newline-separated number list)
	body := strings.Join(numbers, "\n")

	return c.makeRequest(ctx, func() (*http.Response, error) {
		return c.generated.RegisterUNIPServicePOSTWithTextBody(ctx,
			typeEnum,
			generated.RegisterUNIPServicePOSTParamsFormatEpodoc,
			generated.RegisterUNIPServicePOSTTextRequestBody(body))
	})
}

// SearchRegister searches the EPO Register for patents matching a query.
//
// Parameters:
//   - query: Search query (e.g., "ti=plastic", "applicant=google")
//   - rangeSpec: Optional range specification (e.g., "1-25", "26-50"). Empty string uses default (1-25).
//
// The query parameter supports various search fields including:
//   - ti (title), ab (abstract), ta (title and abstract)
//   - applicant, inventor
//   - pn (publication number), an (application number)
//   - pd (publication date), ad (application date)
//   - cl (classification)
//
// Returns XML or JSON with matching register entries including bibliographic data.
//
// Example:
//
//	results, err := client.SearchRegister(ctx, "ti=battery AND applicant=tesla", "1-100")
func (c *Client) SearchRegister(ctx context.Context, query, rangeSpec string) (string, error) {
	if query == "" {
		return "", &ConfigError{Message: "search query cannot be empty"}
	}

	params := &generated.RegisterSearchServiceWithoutConstituentsParams{
		Q: query,
	}

	if rangeSpec != "" {
		params.Range = &rangeSpec
	}

	return c.makeRequest(ctx, func() (*http.Response, error) {
		return c.generated.RegisterSearchServiceWithoutConstituents(ctx, params)
	})
}

// SearchRegisterWithConstituent searches the EPO Register and returns specific constituent data.
//
// Parameters:
//   - constituent: The type of data to return ("biblio", "events", "procedural-steps", "upp")
//   - query: Search query (e.g., "ti=plastic", "applicant=google")
//   - rangeSpec: Optional range specification (e.g., "1-25"). Empty string uses default (1-25).
//
// Constituents:
//   - "biblio": Bibliographic data
//   - "events": Legal events
//   - "procedural-steps": Procedural step information
//   - "upp": Unified Patent Package data
//
// Returns XML or JSON with matching register entries for the specified constituent.
//
// Example:
//
//	// Search for patents and get legal events
//	events, err := client.SearchRegisterWithConstituent(ctx, "events", "applicant=tesla", "1-50")
func (c *Client) SearchRegisterWithConstituent(ctx context.Context, constituent, query, rangeSpec string) (string, error) {
	if constituent == "" {
		return "", &ConfigError{Message: "constituent cannot be empty"}
	}

	if query == "" {
		return "", &ConfigError{Message: "search query cannot be empty"}
	}

	// Validate and convert constituent to enum
	validConstituents := map[string]generated.RegisterSearchServiceWithVariableConstituentsParamsConstituent{
		"biblio":           generated.RegisterSearchServiceWithVariableConstituentsParamsConstituentBiblio,
		"events":           generated.RegisterSearchServiceWithVariableConstituentsParamsConstituentEvents,
		"procedural-steps": generated.RegisterSearchServiceWithVariableConstituentsParamsConstituentProceduralSteps,
		"upp":              generated.RegisterSearchServiceWithVariableConstituentsParamsConstituentUpp,
	}

	constituentEnum, ok := validConstituents[constituent]
	if !ok {
		return "", &ConfigError{Message: "constituent must be one of: biblio, events, procedural-steps, upp"}
	}

	params := &generated.RegisterSearchServiceWithVariableConstituentsParams{
		Q: query,
	}

	if rangeSpec != "" {
		params.Range = &rangeSpec
	}

	return c.makeRequest(ctx, func() (*http.Response, error) {
		return c.generated.RegisterSearchServiceWithVariableConstituents(ctx, constituentEnum, params)
	})
}
