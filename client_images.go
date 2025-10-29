package epo_ops

import (
	"context"
	"net/http"

	"github.com/patent-dev/epo-ops/generated"
)

// Images Service - Patent image retrieval.
//
// This file contains methods for retrieving patent images (drawings, full documents, etc.)

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
//   - What image formats are available (TIFF, PDF, PNG)
//   - Document types (Drawing, FullDocument, FirstPageClipping)
//
// Parameters:
//   - refType: Reference type (e.g., RefTypePublication, RefTypeApplication, RefTypePriority)
//   - format: Number format (e.g., FormatDocDB, FormatEPODOC)
//   - number: Patent number (e.g., "EP1000000")
//
// Returns an ImageInquiry struct with available image metadata.
//
// Example:
//
//	inquiry, err := client.GetImageInquiry(ctx, ops.RefTypePublication, "docdb", "EP1000000")
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
