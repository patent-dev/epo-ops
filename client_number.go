package epo_ops

import (
	"context"
	"fmt"
	"net/http"

	"github.com/patent-dev/epo-ops/generated"
)

// Number Conversion Service - Patent number format conversion.
//
// This file contains methods for converting patent numbers between formats.

// ConvertPatentNumber converts a patent number between formats and returns parsed data.
//
// Parameters:
//   - refType: Reference type (e.g., RefTypePublication, RefTypeApplication, RefTypePriority)
//   - inputFormat: Input format ("original", "epodoc", "docdb")
//   - number: Patent number in input format
//   - outputFormat: Output format ("original", "epodoc", "docdb")
//
// Returns parsed NumberConversionData with Country, DocNumber, Kind, and Date fields.
func (c *Client) ConvertPatentNumber(ctx context.Context, refType, inputFormat, number, outputFormat string) (*NumberConversionData, error) {
	xmlData, err := c.ConvertPatentNumberRaw(ctx, refType, inputFormat, number, outputFormat)
	if err != nil {
		return nil, err
	}
	return ParseNumberConversion(xmlData)
}

func (c *Client) ConvertPatentNumberRaw(ctx context.Context, refType, inputFormat, number, outputFormat string) (string, error) {
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
func (c *Client) ConvertPatentNumberMultipleRaw(ctx context.Context, refType, inputFormat string, numbers []string, outputFormat string) (string, error) {
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
