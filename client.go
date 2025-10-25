// Package epo_ops provides a Go client for the European Patent Office's Open Patent Services (OPS) API v3.2.
//
// This library provides an idiomatic Go interface to interact with the EPO's Open Patent Services,
// allowing you to retrieve patent bibliographic data, claims, descriptions, search for patents,
// get patent family information, download images, and more.
//
// Example usage:
//
//	config := &ops.Config{
//	    ConsumerKey:    "your-consumer-key",
//	    ConsumerSecret: "your-consumer-secret",
//	}
//
//	client, err := ops.NewClient(config)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	ctx := context.Background()
//	biblio, err := client.GetBiblio(ctx, "publication", "docdb", "EP1000000")
//	if err != nil {
//	    log.Fatal(err)
//	}
package epo_ops

//go:generate oapi-codegen -package generated -generate types openapi.yaml -o generated/types_gen.go
//go:generate oapi-codegen -package generated -generate client openapi.yaml -o generated/client_gen.go

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/patent-dev/epo-ops/cql"
	"github.com/patent-dev/epo-ops/generated"
)

// Client is the main EPO OPS API client.
type Client struct {
	config        *Config
	httpClient    *http.Client
	authenticator *Authenticator
	generated     *generated.Client
	quota         *quotaTracker
}

// getAcceptHeader returns the appropriate Accept header value based on the endpoint type.
// The EPO OPS API requires different Accept headers for different service endpoints.
func getAcceptHeader(endpoint string) string {
	switch endpoint {
	case EndpointBiblio, EndpointAbstract:
		return "application/exchange+xml"
	case EndpointFulltext, EndpointClaims, EndpointDescription:
		return "application/fulltext+xml"
	case EndpointFamily, EndpointLegal, EndpointSearch:
		return "application/ops+xml"
	case EndpointRegister:
		return "application/register+xml"
	case EndpointImages:
		return "application/tiff"
	default:
		return "application/xml"
	}
}

// getEndpointFromPath extracts the endpoint type from the URL path.
// This is used to determine the appropriate Accept header.
func getEndpointFromPath(path string) string {
	if strings.Contains(path, "/published-data/publication/") {
		// Parse the constituent (biblio, abstract, claims, description, fulltext)
		parts := strings.Split(path, "/")
		for i, part := range parts {
			if part == "publication" && i+3 < len(parts) {
				constituent := parts[i+3]
				switch constituent {
				case "biblio":
					return EndpointBiblio
				case "abstract":
					return EndpointAbstract
				case "claims":
					return EndpointClaims
				case "description":
					return EndpointDescription
				case "fulltext":
					return EndpointFulltext
				}
			}
		}
	}
	if strings.Contains(path, "/family/") {
		return EndpointFamily
	}
	if strings.Contains(path, "/legal") {
		return EndpointLegal
	}
	if strings.Contains(path, "/register") {
		return EndpointRegister
	}
	if strings.Contains(path, "/published-data/search") {
		return EndpointSearch
	}
	if strings.Contains(path, "/published-data/images") {
		return EndpointImages
	}
	return ""
}

// authTransport wraps an http.RoundTripper to add OAuth2 Bearer token to requests.
type authTransport struct {
	base          http.RoundTripper
	authenticator *Authenticator
}

func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Get valid token
	token, err := t.authenticator.GetToken(req.Context())
	if err != nil {
		return nil, err
	}

	// Clone request to avoid modifying original
	req2 := req.Clone(req.Context())
	req2.Header.Set("Authorization", "Bearer "+token)

	// Set Accept header based on endpoint type
	endpoint := getEndpointFromPath(req.URL.Path)
	if endpoint != "" {
		acceptHeader := getAcceptHeader(endpoint)
		req2.Header.Set("Accept", acceptHeader)
	}

	// Perform request
	return t.base.RoundTrip(req2)
}

// NewClient creates a new EPO OPS API client.
func NewClient(config *Config) (*Client, error) {
	if config == nil {
		config = DefaultConfig()
	}

	// Validate required fields
	if config.ConsumerKey == "" {
		return nil, &ConfigError{Message: "ConsumerKey is required"}
	}
	if config.ConsumerSecret == "" {
		return nil, &ConfigError{Message: "ConsumerSecret is required"}
	}

	// Set defaults if not provided
	if config.BaseURL == "" {
		config.BaseURL = "https://ops.epo.org/3.2/rest-services"
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}
	if config.RetryDelay == 0 {
		config.RetryDelay = 1 * time.Second
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	// Create base HTTP client
	baseClient := &http.Client{
		Timeout: config.Timeout,
	}

	// Create authenticator
	authenticator := NewAuthenticator(config.ConsumerKey, config.ConsumerSecret, baseClient)

	// Override auth URL if specified in config (mainly for testing)
	if config.AuthURL != "" {
		authenticator.authURL = config.AuthURL
	}

	// Create HTTP client with auth transport
	httpClient := &http.Client{
		Timeout: config.Timeout,
		Transport: &authTransport{
			base:          http.DefaultTransport,
			authenticator: authenticator,
		},
	}

	// Create generated client
	genClient, err := generated.NewClient(config.BaseURL, generated.WithHTTPClient(httpClient))
	if err != nil {
		return nil, err
	}

	return &Client{
		config:        config,
		httpClient:    httpClient,
		authenticator: authenticator,
		generated:     genClient,
		quota:         &quotaTracker{},
	}, nil
}

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
func (c *Client) GetDescription(ctx context.Context, refType, format, number string) (string, error) {
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
// Returns the full text as an XML string.
func (c *Client) GetFulltext(ctx context.Context, refType, format, number string) (string, error) {
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

// GetLastQuota returns the last quota information from API responses.
// Returns nil if no API calls have been made yet.
//
// Quota tracking helps monitor fair use limits (4GB/week for non-paying users).
// The returned QuotaInfo includes:
//   - Status: "green" (<50%), "yellow" (50-75%), "red" (>75%), "black" (blocked)
//   - Individual: Quota for individual users
//   - Registered: Quota for registered/paying users
//   - Images: Separate quota for image downloads
func (c *Client) GetLastQuota() *QuotaInfo {
	return c.quota.Get()
}

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
func (c *Client) Search(ctx context.Context, query string, rangeStr string) (string, error) {
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
// Returns the search results as XML with the requested constituent data.
func (c *Client) SearchWithConstituent(ctx context.Context, constituent, query string, rangeStr string) (string, error) {
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

	params := &generated.PublishedDataKeywordsSearchWithVariableConstituentsParams{
		Q:     query,
		Range: &rangeStr,
	}

	return c.makeRequest(ctx, func() (*http.Response, error) {
		return c.generated.PublishedDataKeywordsSearchWithVariableConstituents(ctx,
			generated.PublishedDataKeywordsSearchWithVariableConstituentsParamsConstituent(constituent),
			params)
	})
}

// GetFamily retrieves the INPADOC patent family for a given patent.
//
// Parameters:
//   - refType: Reference type (e.g., "publication", "application", "priority")
//   - format: Number format (e.g., "docdb", "epodoc")
//   - number: Patent number (e.g., "EP1000000")
//
// Returns the family data as XML containing all family members.
//
// INPADOC (International Patent Documentation) family includes all patents
// related through priority claims.
func (c *Client) GetFamily(ctx context.Context, refType, format, number string) (string, error) {
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
// Returns the family data with biblio as XML.
func (c *Client) GetFamilyWithBiblio(ctx context.Context, refType, format, number string) (string, error) {
	if err := ValidateRefType(refType); err != nil {
		return "", err
	}
	if err := ValidateFormat(format, number); err != nil {
		return "", err
	}
	return c.makeRequest(ctx, func() (*http.Response, error) {
		return c.generated.INPADOCFamilyRetrievalServiceWithBiblio(ctx,
			generated.INPADOCFamilyRetrievalServiceWithBiblioParamsType(refType),
			generated.INPADOCFamilyRetrievalServiceWithBiblioParamsFormat(format),
			number)
	})
}

// GetFamilyWithLegal retrieves the INPADOC patent family with legal status data.
//
// Parameters:
//   - refType: Reference type (e.g., "publication", "application", "priority")
//   - format: Number format (e.g., "docdb", "epodoc")
//   - number: Patent number (e.g., "EP1000000")
//
// Returns the family data with legal status as XML.
func (c *Client) GetFamilyWithLegal(ctx context.Context, refType, format, number string) (string, error) {
	if err := ValidateRefType(refType); err != nil {
		return "", err
	}
	if err := ValidateFormat(format, number); err != nil {
		return "", err
	}
	return c.makeRequest(ctx, func() (*http.Response, error) {
		return c.generated.INPADOCFamilyRetrievalServiceWithLegal(ctx,
			generated.INPADOCFamilyRetrievalServiceWithLegalParamsType(refType),
			generated.INPADOCFamilyRetrievalServiceWithLegalParamsFormat(format),
			number)
	})
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
// Returns XML containing family data with biblio for all requested patents.
func (c *Client) GetFamilyWithBiblioMultiple(ctx context.Context, refType, format string, numbers []string) (string, error) {
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
		return c.generated.INPADOCFamilyRetrievalServiceWithBiblioPOSTWithTextBody(ctx,
			generated.INPADOCFamilyRetrievalServiceWithBiblioPOSTParamsType(refType),
			generated.INPADOCFamilyRetrievalServiceWithBiblioPOSTParamsFormat(format),
			body)
	})
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
// Returns XML containing family data with legal status for all requested patents.
func (c *Client) GetFamilyWithLegalMultiple(ctx context.Context, refType, format string, numbers []string) (string, error) {
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
		return c.generated.INPADOCFamilyRetrievalServiceWithLegalPOSTWithTextBody(ctx,
			generated.INPADOCFamilyRetrievalServiceWithLegalPOSTParamsType(refType),
			generated.INPADOCFamilyRetrievalServiceWithLegalPOSTParamsFormat(format),
			body)
	})
}

// GetImage retrieves a patent image (drawing page).
//
// Parameters:
//   - country: Two-letter country code (e.g., "EP", "US", "WO")
//   - number: Patent number without country code (e.g., "2400812")
//   - kind: Kind code (e.g., "A1", "B1")
//   - imageType: Image type - use ImageTypeFullImage constant
//   - page: Page number (1-based, e.g., 1)
//
// Returns the image data as bytes (typically TIFF format).
//
// Example:
//
//	imageData, err := client.GetImage(ctx, "EP", "2400812", "A1", ops.ImageTypeFullImage, 1)
//
// Note: EPO typically returns images in TIFF format. Use tiffutil.TIFFToPNG()
// to convert to PNG format.
func (c *Client) GetImage(ctx context.Context, country, number, kind, imageType string, page int) ([]byte, error) {
	params := &generated.PublishedImagesRetrievalServiceParams{
		Range: page,
	}

	return c.makeBinaryRequest(ctx, func() (*http.Response, error) {
		return c.generated.PublishedImagesRetrievalService(ctx, country, number, kind, imageType, params)
	})
}

// GetImagePOST retrieves a patent image using POST method (keeps document identifier encrypted in body).
// This is identical to GetImage but uses POST instead of GET, keeping the document identifier
// in the encrypted request body rather than the URL. Both methods return one page at a time.
//
// Parameters:
//   - page: Page number to retrieve (1-based, e.g., 1)
//   - identifier: Document identifier in format "CC/NNNNNNNN/KC/TYPE"
//     (e.g., "EP/1000000/A1/fullimage", "EP/2400812/A1/drawing")
//
// Returns the binary image data (TIFF, PDF, or PNG format) for the specified page.
//
// Note: Despite the POST method, this does NOT retrieve multiple pages at once.
// Use the page parameter to iterate through pages one at a time.
//
// Example:
//
//	// Get first page of full document
//	data, err := client.GetImagePOST(ctx, 1, "EP/1000000/A1/fullimage")
func (c *Client) GetImagePOST(ctx context.Context, page int, identifier string) ([]byte, error) {
	if identifier == "" {
		return nil, &ValidationError{
			Field:   "identifier",
			Message: "document identifier required",
		}
	}

	params := &generated.PublishedImagesRetrievalServicePOSTParams{
		Range: page,
	}

	// Use generated POST method with single identifier
	body := identifier
	return c.makeBinaryRequest(ctx, func() (*http.Response, error) {
		return c.generated.PublishedImagesRetrievalServicePOSTWithTextBody(ctx, params, body)
	})
}

// GetImageInquiry retrieves metadata about available images for a patent.
//
// This method queries what images are available without downloading them.
// Use this to discover:
//   - How many pages of drawings exist
//   - What document types are available (drawings, full document, etc.)
//   - What formats are available (PDF, TIFF, etc.)
//
// Parameters:
//   - refType: Reference type ("publication", "application", "priority")
//   - format: Number format ("docdb", "epodoc", "original")
//   - number: Patent number (e.g., "EP.1000000.B1" for docdb format)
//
// Returns an ImageInquiry containing:
//   - DocumentInstances: List of available document types
//   - Each instance includes: Description, NumberOfPages, Formats, Link
//
// Example workflow:
//
//	// First, check what's available
//	inquiry, err := client.GetImageInquiry(ctx, "publication", "docdb", "EP.1000000.B1")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Then download the actual images
//	for _, instance := range inquiry.DocumentInstances {
//	    fmt.Printf("Found %s with %d pages\n", instance.Description, instance.NumberOfPages)
//	    for page := 1; page <= instance.NumberOfPages; page++ {
//	        img, _ := client.GetImage(ctx, "EP", "1000000", "B1", "fullimage", page)
//	        // Process image...
//	    }
//	}
func (c *Client) GetImageInquiry(ctx context.Context, refType, format, number string) (*ImageInquiry, error) {
	if err := ValidateRefType(refType); err != nil {
		return nil, err
	}
	if err := ValidateFormat(format, number); err != nil {
		return nil, err
	}

	xmlData, err := c.makeRequest(ctx, func() (*http.Response, error) {
		return c.generated.PublishedImagesInquiryService(ctx,
			generated.PublishedImagesInquiryServiceParamsType(refType),
			generated.PublishedImagesInquiryServiceParamsFormat(format),
			number)
	})
	if err != nil {
		return nil, err
	}

	return ParseImageInquiry(xmlData)
}

// GetUsageStats retrieves usage statistics from the EPO OPS Data Usage API.
//
// The Data Usage API provides historical usage data for quota monitoring and analysis.
// Usage statistics are updated within 10 minutes of each hour and aligned on midnight
// UTC/GMT boundaries. This API does not count against quotas.
//
// Parameters:
//   - timeRange: Time range in one of two formats:
//   - Single date: "dd/mm/yyyy" (e.g., "01/01/2024")
//   - Date range: "dd/mm/yyyy~dd/mm/yyyy" (e.g., "01/01/2024~07/01/2024")
//
// Returns:
//   - UsageStats containing usage entries with timestamps, response sizes, and message counts
//   - error if the time range format is invalid or the request fails
//
// Example:
//
//	// Get usage for a specific date
//	stats, err := client.GetUsageStats(ctx, "01/01/2024")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Get usage for a date range
//	stats, err := client.GetUsageStats(ctx, "01/01/2024~07/01/2024")
//	for _, entry := range stats.Entries {
//	    fmt.Printf("Time: %d, Size: %d bytes, Messages: %d\n",
//	        entry.Timestamp, entry.TotalResponseSize, entry.MessageCount)
//	}
func (c *Client) GetUsageStats(ctx context.Context, timeRange string) (*UsageStats, error) {
	// Validate time range format
	if err := ValidateTimeRange(timeRange); err != nil {
		return nil, err
	}

	// Use generated client stub (endpoint now included in OpenAPI spec via convert-openapi.sh)
	params := &generated.GetUsageStatisticsParams{
		TimeRange: timeRange,
	}

	// Execute request using generated stub
	jsonData, err := c.makeRequest(ctx, func() (*http.Response, error) {
		return c.generated.GetUsageStatistics(ctx, params)
	})

	if err != nil {
		return nil, err
	}

	// Parse JSON response
	return parseUsageStats(jsonData, timeRange)
}

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
func (c *Client) GetClassificationSchema(ctx context.Context, class string, ancestors, navigation bool) (string, error) {
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
func (c *Client) GetClassificationSchemaSubclass(ctx context.Context, class, subclass string, ancestors, navigation bool) (string, error) {
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
func (c *Client) GetClassificationSchemaMultiple(ctx context.Context, classes []string) (string, error) {
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
func (c *Client) GetClassificationStatistics(ctx context.Context, query string) (string, error) {
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
func (c *Client) GetClassificationMapping(ctx context.Context, inputFormat, class, subclass, outputFormat string, additional bool) (string, error) {
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
// Returns legal status data as XML, including:
//   - Legal events (grants, oppositions, revocations, etc.)
//   - Status in different jurisdictions
//   - Procedural information
func (c *Client) GetLegal(ctx context.Context, refType, format, number string) (string, error) {
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
// Returns XML containing legal status data for all requested patents.
func (c *Client) GetLegalMultiple(ctx context.Context, refType, format string, numbers []string) (string, error) {
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
		return c.generated.LegalDataRetrievalServicePOSTWithTextBody(ctx,
			generated.LegalDataRetrievalServicePOSTParamsType(refType),
			generated.LegalDataRetrievalServicePOSTParamsFormat(format),
			body)
	})
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
func (c *Client) GetRegisterBiblio(ctx context.Context, refType, format, number string) (string, error) {
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
func (c *Client) GetRegisterBiblioMultiple(ctx context.Context, refType, format string, numbers []string) (string, error) {
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
func (c *Client) GetRegisterEvents(ctx context.Context, refType, format, number string) (string, error) {
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
func (c *Client) GetRegisterEventsMultiple(ctx context.Context, refType, format string, numbers []string) (string, error) {
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
func (c *Client) GetRegisterProceduralSteps(ctx context.Context, refType, format, number string) (string, error) {
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
func (c *Client) GetRegisterProceduralStepsMultiple(ctx context.Context, refType, format string, numbers []string) (string, error) {
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
func (c *Client) GetRegisterUNIP(ctx context.Context, refType, format, number string) (string, error) {
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
func (c *Client) GetRegisterUNIPMultiple(ctx context.Context, refType, format string, numbers []string) (string, error) {
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

// ConvertPatentNumber converts a patent number from one format to another.
//
// Parameters:
//   - refType: Reference type (e.g., "publication", "application", "priority")
//   - inputFormat: Input format ("original", "epodoc", "docdb")
//   - number: Patent number in input format
//   - outputFormat: Output format ("original", "epodoc", "docdb")
//
// Returns the patent number in the requested output format as XML.
//
// Format examples:
//   - original: US.(05/948,554).19781004
//   - epodoc: US19780948554
//   - docdb: US 19780948554
func (c *Client) ConvertPatentNumber(ctx context.Context, refType, inputFormat, number, outputFormat string) (string, error) {
	if err := ValidateRefType(refType); err != nil {
		return "", err
	}
	if err := ValidateFormat(inputFormat, number); err != nil {
		return "", err
	}
	// Validate output format (just check it's valid, not number specific)
	if outputFormat != FormatDocDB && outputFormat != FormatEPODOC && outputFormat != FormatOriginal {
		return "", &ValidationError{
			Field:   "outputFormat",
			Value:   outputFormat,
			Message: "must be 'docdb', 'epodoc', or 'original'",
		}
	}
	return c.makeRequest(ctx, func() (*http.Response, error) {
		return c.generated.NumberService(ctx,
			generated.NumberServiceParamsType(refType),
			generated.NumberServiceParamsInputFormat(inputFormat),
			number,
			generated.NumberServiceParamsOutputFormat(outputFormat))
	})
}

// ConvertPatentNumberMultiple converts multiple patent numbers from one format to another.
// Uses POST endpoint for efficient batch conversion of up to 100 patents in one request.
//
// Parameters:
//   - refType: Reference type (e.g., RefTypePublication, RefTypeApplication, RefTypePriority)
//   - inputFormat: Input format ("original", "epodoc", "docdb")
//   - numbers: Slice of patent numbers in input format (max 100)
//   - outputFormat: Output format ("original", "epodoc", "docdb")
//
// Returns XML containing converted patent numbers for all requested patents.
func (c *Client) ConvertPatentNumberMultiple(ctx context.Context, refType, inputFormat string, numbers []string, outputFormat string) (string, error) {
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

	// Validate output format
	if outputFormat != FormatDocDB && outputFormat != FormatEPODOC && outputFormat != FormatOriginal {
		return "", &ValidationError{
			Field:   "outputFormat",
			Value:   outputFormat,
			Message: "must be 'docdb', 'epodoc', or 'original'",
		}
	}

	// Validate each patent number
	for i, number := range numbers {
		if err := ValidateFormat(inputFormat, number); err != nil {
			return "", fmt.Errorf("numbers[%d]: %w", i, err)
		}
	}

	// Use generated POST method
	body := formatBulkBody(numbers)
	return c.makeRequest(ctx, func() (*http.Response, error) {
		return c.generated.NumberServicePOSTWithTextBody(ctx,
			generated.NumberServicePOSTParamsType(refType),
			generated.NumberServicePOSTParamsInputFormat(inputFormat),
			generated.NumberServicePOSTParamsOutputFormat(outputFormat),
			body)
	})
}

// formatBulkBody formats patent numbers as newline-delimited text for POST body.
// This format is required by EPO OPS bulk retrieval endpoints.
func formatBulkBody(numbers []string) string {
	return strings.Join(numbers, "\n")
}

// GetBiblioMultiple retrieves bibliographic data for multiple patents (bulk operation).
// Uses POST endpoint for efficient batch retrieval of up to 100 patents in one request.
//
// Parameters:
//   - refType: Reference type (e.g., RefTypePublication, RefTypeApplication, RefTypePriority)
//   - format: Number format (e.g., FormatDocDB, FormatEPODOC)
//   - numbers: Slice of patent numbers (max 100)
//
// Returns XML containing bibliographic data for all requested patents.
//
// Example:
//
//	numbers := []string{"EP.1000000.B1", "EP.1000001.A1", "US.5551212.A"}
//	xml, err := client.GetBiblioMultiple(ctx, ops.RefTypePublication, ops.FormatDocDB, numbers)
func (c *Client) GetBiblioMultiple(ctx context.Context, refType, format string, numbers []string) (string, error) {
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
func (c *Client) GetClaimsMultiple(ctx context.Context, refType, format string, numbers []string) (string, error) {
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
		return c.generated.PublishedClaimsRetrievalServicePOSTWithTextBody(ctx,
			generated.PublishedClaimsRetrievalServicePOSTParamsType(refType),
			generated.PublishedClaimsRetrievalServicePOSTParamsFormat(format),
			body)
	})
}

// GetDescriptionMultiple retrieves descriptions for multiple patents (bulk operation).
// Uses POST endpoint for efficient batch retrieval of up to 100 patents in one request.
//
// Parameters:
//   - refType: Reference type (e.g., RefTypePublication, RefTypeApplication, RefTypePriority)
//   - format: Number format (e.g., FormatDocDB, FormatEPODOC)
//   - numbers: Slice of patent numbers (max 100)
//
// Returns XML containing descriptions for all requested patents.
func (c *Client) GetDescriptionMultiple(ctx context.Context, refType, format string, numbers []string) (string, error) {
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
		return c.generated.PublishedDataDescriptionRetrievalServicePOSTWithTextBody(ctx,
			generated.PublishedDataDescriptionRetrievalServicePOSTParamsType(refType),
			generated.PublishedDataDescriptionRetrievalServicePOSTParamsFormat(format),
			body)
	})
}

// GetAbstractMultiple retrieves abstracts for multiple patents (bulk operation).
// Uses POST endpoint for efficient batch retrieval of up to 100 patents in one request.
//
// Parameters:
//   - refType: Reference type (e.g., RefTypePublication, RefTypeApplication, RefTypePriority)
//   - format: Number format (e.g., FormatDocDB, FormatEPODOC)
//   - numbers: Slice of patent numbers (max 100)
//
// Returns XML containing abstracts for all requested patents.
func (c *Client) GetAbstractMultiple(ctx context.Context, refType, format string, numbers []string) (string, error) {
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
		return c.generated.PublishedDataAbstractServicePOSTWithTextBody(ctx,
			generated.PublishedDataAbstractServicePOSTParamsType(refType),
			generated.PublishedDataAbstractServicePOSTParamsFormat(format),
			body)
	})
}

// GetFulltextMultiple retrieves full text for multiple patents (bulk operation).
// Uses POST endpoint for efficient batch retrieval of up to 100 patents in one request.
//
// Parameters:
//   - refType: Reference type (e.g., RefTypePublication, RefTypeApplication, RefTypePriority)
//   - format: Number format (e.g., FormatDocDB, FormatEPODOC)
//   - numbers: Slice of patent numbers (max 100)
//
// Returns XML containing full text (biblio, abstract, description, claims) for all requested patents.
func (c *Client) GetFulltextMultiple(ctx context.Context, refType, format string, numbers []string) (string, error) {
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
		return c.generated.PublishedDataFulltextInquiryServicePOSTWithTextBody(ctx,
			generated.PublishedDataFulltextInquiryServicePOSTParamsType(refType),
			generated.PublishedDataFulltextInquiryServicePOSTParamsFormat(format),
			body)
	})
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
func (c *Client) GetPublishedEquivalents(ctx context.Context, refType, format, number string) (string, error) {
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
// Returns XML or JSON with equivalent publications for all requested patents.
//
// Example:
//
//	numbers := []string{"EP1000000", "EP1000001"}
//	equivalents, err := client.GetPublishedEquivalentsMultiple(ctx, epo_ops.RefTypePublication, "epodoc", numbers)
func (c *Client) GetPublishedEquivalentsMultiple(ctx context.Context, refType, format string, numbers []string) (string, error) {
	// Validate reference type
	if err := ValidateRefType(refType); err != nil {
		return "", err
	}

	// Validate format
	if format != "epodoc" && format != "docdb" {
		return "", &ConfigError{Message: "format must be 'epodoc' or 'docdb'"}
	}

	// Validate numbers list
	if len(numbers) == 0 {
		return "", &ConfigError{Message: "numbers list cannot be empty"}
	}

	if len(numbers) > 100 {
		return "", &ConfigError{Message: "maximum 100 numbers allowed per request"}
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
		return c.generated.PublishedEquivalentsRetrievalServicePOSTWithTextBody(ctx,
			generated.PublishedEquivalentsRetrievalServicePOSTParamsType(refType),
			generated.PublishedEquivalentsRetrievalServicePOSTParamsFormat(format),
			body)
	})
}

// executeRequest is a common helper that executes an HTTP request with retry logic and 401 handling.
// Returns the response body as bytes.
func (c *Client) executeRequest(ctx context.Context, fn func() (*http.Response, error)) ([]byte, error) {
	var retriedAfter401 atomic.Bool

	// Wrapper that handles 401 token refresh
	requestWithAuth := func() (*http.Response, error) {
		resp, err := fn()

		// Special handling for 401 errors: clear token and retry once
		// Use atomic swap to ensure only one retry happens even with concurrent requests
		if err == nil && resp.StatusCode == http.StatusUnauthorized && !retriedAfter401.Swap(true) {
			_ = resp.Body.Close() // Ignore close error, we're retrying the request

			// Clear cached token to force refresh on next attempt
			c.authenticator.ClearToken()

			// Retry the request immediately (token will be refreshed by authTransport)
			resp, err = fn()
		}

		return resp, err
	}

	// Execute with retry logic
	resp, err := c.retryableRequest(ctx, requestWithAuth)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Parse and store quota information from headers
	quotaInfo := ParseQuotaHeaders(resp.Header)
	c.quota.Update(quotaInfo)

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, c.handleErrorResponse(resp.StatusCode, body)
	}

	return body, nil
}

// makeRequest executes an HTTP request with retry logic and returns the response body as a string.
func (c *Client) makeRequest(ctx context.Context, fn func() (*http.Response, error)) (string, error) {
	body, err := c.executeRequest(ctx, fn)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

// makeBinaryRequest executes an HTTP request with retry logic and returns the response body as bytes.
// This is used for binary data like images.
func (c *Client) makeBinaryRequest(ctx context.Context, fn func() (*http.Response, error)) ([]byte, error) {
	return c.executeRequest(ctx, fn)
}

// handleErrorResponse converts HTTP error responses into appropriate error types.
func (c *Client) handleErrorResponse(statusCode int, body []byte) error {
	// Try to parse structured XML error first
	opsErr, err := parseErrorXML(body, statusCode)
	if err == nil && opsErr != nil {
		// Map specific error codes to appropriate error types
		switch opsErr.Code {
		case "CLIENT.InvalidReference", "SERVER.EntityNotFound", "HTTP.404":
			return &NotFoundError{
				Message: opsErr.Message,
			}
		case "CLIENT.InvalidAccessToken", "CLIENT.MissingAccessToken", "HTTP.401":
			return &AuthError{
				StatusCode: statusCode,
				Message:    opsErr.Message,
			}
		case "SERVER.RateLimitExceeded", "SERVER.QuotaPerWeekExceeded", "HTTP.429", "HTTP.403":
			return &QuotaExceededError{
				Message: opsErr.Message,
			}
		case "HTTP.503":
			return &ServiceUnavailableError{
				StatusCode: statusCode,
				Message:    opsErr.Message,
			}
		default:
			// Return the parsed OPSError for other codes
			return opsErr
		}
	}

	// Fall back to status-code-based error handling if XML parsing fails
	switch statusCode {
	case http.StatusNotFound:
		return &NotFoundError{
			Message: string(body),
		}
	case http.StatusUnauthorized:
		return &AuthError{
			StatusCode: statusCode,
			Message:    string(body),
		}
	case http.StatusTooManyRequests, http.StatusForbidden:
		return &QuotaExceededError{
			Message: string(body),
		}
	case http.StatusServiceUnavailable:
		return &ServiceUnavailableError{
			StatusCode: statusCode,
			Message:    string(body),
		}
	default:
		return fmt.Errorf("HTTP %d: %s", statusCode, string(body))
	}
}
