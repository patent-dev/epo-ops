package epo_ops

import (
	"context"
	"net/http"
	"strings"

	"github.com/patent-dev/epo-ops/generated"
)

// Classification Service - CPC and ECLA classification services.
//
// This file contains methods for CPC schema, statistics, mapping, and media.

// GetClassificationSchema retrieves CPC classification schema hierarchy.
//
// This method retrieves the Cooperative Patent Classification (CPC) hierarchy for a given
// classification symbol. The CPC is a patent classification system jointly developed by
// the EPO and USPTO.
//
// Parameters:
//   - class: CPC classification symbol (e.g., "A01B", "H04W84/18")
//   - Section + Class format: "A01" retrieves all subclasses
//   - Class + Subclass format: "A01B" retrieves class hierarchy
//   - Full symbol: "H04W84/18" retrieves specific classification
//   - ancestors: If true, include ancestor classifications in the hierarchy
//   - navigation: If true, include navigation links to related classifications
//
// Returns XML containing:
//   - Classification hierarchy structure
//   - Class titles and descriptions
//   - Parent/child relationships
//   - Related classification links (if navigation=true)
//
// Example:
//
//	// Get full hierarchy for class A01B
//	schema, err := client.GetClassificationSchema(ctx, "A01B", false, false)
//
//	// Get with ancestors and navigation
//	schema, err := client.GetClassificationSchema(ctx, "H04W84/18", true, true)
func (c *Client) GetClassificationSchemaRaw(ctx context.Context, class string, ancestors, navigation bool) (string, error) {
	if class == "" {
		return "", &ConfigError{Message: "classification class cannot be empty"}
	}

	// Build params
	params := &generated.ClassificationSchemaServiceParams{}
	if ancestors {
		ancestorsFlag := generated.ClassificationSchemaServiceParamsAncestors(true)
		params.Ancestors = &ancestorsFlag
	}
	if navigation {
		navFlag := generated.ClassificationSchemaServiceParamsNavigation(true)
		params.Navigation = &navFlag
	}

	return c.makeRequest(ctx, func() (*http.Response, error) {
		return c.generated.ClassificationSchemaService(ctx, class, params)
	})
}

// GetClassificationSchemaSubclass retrieves CPC classification schema for a specific subclass.
//
// This is a more specific version of GetClassificationSchema that retrieves classification
// hierarchy for a specific class/subclass combination.
//
// Parameters:
//   - class: CPC class identifier (e.g., "A01B1")
//   - subclass: CPC subclass identifier (e.g., "00")
//   - ancestors: If true, include ancestor classifications in the hierarchy
//   - navigation: If true, include navigation links to related classifications
//
// Returns XML containing the subclass classification hierarchy.
//
// Example:
//
//	// Get specific subclass hierarchy
//	schema, err := client.GetClassificationSchemaSubclass(ctx, "A01B1", "00", false, false)
func (c *Client) GetClassificationSchemaSubclassRaw(ctx context.Context, class, subclass string, ancestors, navigation bool) (string, error) {
	if class == "" {
		return "", &ConfigError{Message: "classification class cannot be empty"}
	}
	if subclass == "" {
		return "", &ConfigError{Message: "classification subclass cannot be empty"}
	}

	// Build params
	params := &generated.ClassificationSchemaSubclassServiceParams{}
	if ancestors {
		ancestorsFlag := generated.ClassificationSchemaSubclassServiceParamsAncestors(true)
		params.Ancestors = &ancestorsFlag
	}
	if navigation {
		navFlag := generated.ClassificationSchemaSubclassServiceParamsNavigation(true)
		params.Navigation = &navFlag
	}

	return c.makeRequest(ctx, func() (*http.Response, error) {
		return c.generated.ClassificationSchemaSubclassService(ctx, class, subclass, params)
	})
}

// GetClassificationSchemaMultiple retrieves CPC classification schemas for multiple classifications.
//
// This method uses the POST endpoint to retrieve classification data for multiple
// classification symbols in a single request.
//
// Parameters:
//   - classes: Slice of CPC classification symbols (max 100)
//   - Each can be in any supported format: "A01", "A01B", "H04W84/18"
//
// Returns XML containing classification hierarchies for all requested symbols.
//
// Example:
//
//	classes := []string{"A01B", "H04W", "G06F17/30"}
//	schemas, err := client.GetClassificationSchemaMultiple(ctx, classes)
func (c *Client) GetClassificationSchemaMultipleRaw(ctx context.Context, classes []string) (string, error) {
	if len(classes) == 0 {
		return "", &ConfigError{Message: "classes list cannot be empty"}
	}
	if len(classes) > 100 {
		return "", &ConfigError{Message: "maximum 100 classes allowed per request"}
	}

	// Build request body (newline-separated class list)
	body := strings.Join(classes, "\n")

	return c.makeRequest(ctx, func() (*http.Response, error) {
		return c.generated.ClassificationSchemaServicePOSTWithTextBody(ctx,
			generated.ClassificationSchemaServicePOSTTextRequestBody(body))
	})
}

// GetClassificationMedia retrieves media files (images/diagrams) for CPC classifications.
//
// The CPC classification system includes illustrative diagrams and images to help
// understand classification concepts. This method downloads these media files.
//
// Parameters:
//   - mediaName: Name of the media resource (e.g., "1000.gif", "5000.png")
//   - Media names are referenced in classification schema XML
//   - Common formats: GIF, PNG, JPG
//   - asAttachment: If true, sets Content-Disposition to attachment (forces download)
//   - false (default): inline display
//   - true: download as attachment
//
// Returns binary image data that can be saved to a file or displayed.
//
// Example:
//
//	// Download a classification diagram
//	imageData, err := client.GetClassificationMedia(ctx, "1000.gif", false)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Save to file
//	err = os.WriteFile("classification-1000.gif", imageData, 0644)
func (c *Client) GetClassificationMedia(ctx context.Context, mediaName string, asAttachment bool) ([]byte, error) {
	if mediaName == "" {
		return nil, &ConfigError{Message: "media name cannot be empty"}
	}

	// Build params
	var params *generated.ClassificationMediaServiceParams
	if asAttachment {
		attachment := generated.ClassificationMediaServiceParamsAttachmentTrue
		params = &generated.ClassificationMediaServiceParams{
			Attachment: &attachment,
		}
	}

	return c.makeBinaryRequest(ctx, func() (*http.Response, error) {
		return c.generated.ClassificationMediaService(ctx, mediaName, params)
	})
}

// GetClassificationStatistics searches for CPC classification statistics.
//
// This method retrieves statistical information about patent counts across CPC
// classification codes. It allows searching for classification codes and returns
// the number of patents in each classification.
//
// Parameters:
//   - query: Search query for classification codes
//   - Can be a keyword (e.g., "plastic", "wireless")
//   - Can be a classification code (e.g., "H04W", "A01B")
//   - Can use wildcard patterns
//
// Returns XML or JSON containing:
//   - Classification codes matching the query
//   - Patent counts for each classification
//   - Classification titles and descriptions
//
// The response format depends on the Accept header sent by the client.
// By default, XML is returned.
//
// Example:
//
//	// Search for statistics on "wireless" classifications
//	stats, err := client.GetClassificationStatistics(ctx, "wireless")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Search for specific classification
//	stats, err := client.GetClassificationStatistics(ctx, "H04W")
func (c *Client) GetClassificationStatisticsRaw(ctx context.Context, query string) (string, error) {
	if query == "" {
		return "", &ConfigError{Message: "search query cannot be empty"}
	}

	params := &generated.ClassificationStatisticsServiceParams{
		Q: query,
	}

	return c.makeRequest(ctx, func() (*http.Response, error) {
		return c.generated.ClassificationStatisticsService(ctx, params)
	})
}

// GetClassificationMapping converts between CPC and ECLA classification formats.
//
// This method maps classification codes between the Cooperative Patent Classification (CPC)
// and European Classification (ECLA) systems. This is useful when working with patents that
// use different classification systems.
//
// Parameters:
//   - inputFormat: Format of the input classification ("cpc" or "ecla")
//   - class: Classification class code (e.g., "A01D2085")
//   - subclass: Classification subclass code (e.g., "8")
//   - outputFormat: Desired output format ("cpc" or "ecla")
//   - additional: If true, include additional/invention information
//
// Returns XML containing:
//   - Mapped classification codes
//   - Mapping relationships
//   - Classification descriptions
//
// Example:
//
//	// Convert ECLA to CPC
//	mapping, err := client.GetClassificationMapping(ctx, "ecla", "A01D2085", "8", "cpc", false)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Convert CPC to ECLA with additional information
//	mapping, err := client.GetClassificationMapping(ctx, "cpc", "H04W84", "18", "ecla", true)
func (c *Client) GetClassificationMappingRaw(ctx context.Context, inputFormat, class, subclass, outputFormat string, additional bool) (string, error) {
	if inputFormat == "" {
		return "", &ConfigError{Message: "input format cannot be empty"}
	}
	if class == "" {
		return "", &ConfigError{Message: "classification class cannot be empty"}
	}
	if subclass == "" {
		return "", &ConfigError{Message: "classification subclass cannot be empty"}
	}
	if outputFormat == "" {
		return "", &ConfigError{Message: "output format cannot be empty"}
	}

	// Validate format values
	if inputFormat != "cpc" && inputFormat != "ecla" {
		return "", &ConfigError{Message: "input format must be 'cpc' or 'ecla'"}
	}
	if outputFormat != "cpc" && outputFormat != "ecla" {
		return "", &ConfigError{Message: "output format must be 'cpc' or 'ecla'"}
	}

	// Convert format strings to enum types
	var inputFmt generated.ClassificationMappingServiceParamsInputFormat
	if inputFormat == "cpc" {
		inputFmt = generated.ClassificationMappingServiceParamsInputFormatCpc
	} else {
		inputFmt = generated.ClassificationMappingServiceParamsInputFormatEcla
	}

	var outputFmt generated.ClassificationMappingServiceParamsOutputFormat
	if outputFormat == "cpc" {
		outputFmt = generated.ClassificationMappingServiceParamsOutputFormatCpc
	} else {
		outputFmt = generated.ClassificationMappingServiceParamsOutputFormatEcla
	}

	params := &generated.ClassificationMappingServiceParams{
		Additional: additional,
	}

	return c.makeRequest(ctx, func() (*http.Response, error) {
		return c.generated.ClassificationMappingService(ctx, inputFmt, class, subclass, outputFmt, params)
	})
}

// GetLegal retrieves legal status data for a patent.
//
// Parameters:
//   - refType: Reference type (e.g., "publication", "application", "priority")
//   - format: Number format (e.g., "docdb", "epodoc")
//   - number: Patent number (e.g., "EP1000000")
//
// Returns parsed legal status data including:
//   - Legal events (grants, oppositions, revocations, etc.)
//   - Status in different jurisdictions
