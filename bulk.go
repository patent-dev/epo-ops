package epo_ops

import (
	"context"
	"fmt"
)

// splitIntoBatches splits items into batches of specified size for EPO-compliant batch sizes (typically 100).
func splitIntoBatches(items []string, batchSize int) [][]string {
	if len(items) == 0 {
		return [][]string{}
	}

	var batches [][]string
	for i := 0; i < len(items); i += batchSize {
		end := i + batchSize
		if end > len(items) {
			end = len(items)
		}
		batches = append(batches, items[i:end])
	}

	return batches
}

// bulkMultipleFunc is a function type for methods that retrieve data for multiple patents.
type bulkMultipleFunc func(ctx context.Context, refType, format string, numbers []string) (string, error)

// executeBulkRequest is a generic method that handles batching and progress tracking for bulk operations.
// It splits large requests into batches of 100 and processes them sequentially.
func (c *Client) executeBulkRequest(ctx context.Context, refType, format string, numbers []string, opts *BulkOptions, fn bulkMultipleFunc) ([]string, error) {
	if opts == nil {
		opts = &BulkOptions{MaxConcurrent: 1}
	}

	batches := splitIntoBatches(numbers, 100)
	results := make([]string, len(batches))

	for i, batch := range batches {
		if opts.OnProgress != nil {
			opts.OnProgress(i+1, len(batches))
		}

		result, err := fn(ctx, refType, format, batch)
		if err != nil {
			return nil, fmt.Errorf("batch %d/%d failed: %w", i+1, len(batches), err)
		}

		results[i] = result
	}

	return results, nil
}

// GetBibliosBulk retrieves bibliographic data for multiple publications with auto-batching.
// Automatically splits requests into batches of 100 (EPO limit) and processes them sequentially.
func (c *Client) GetBibliosBulk(ctx context.Context, refType, format string, numbers []string, opts *BulkOptions) ([]string, error) {
	return c.executeBulkRequest(ctx, refType, format, numbers, opts, c.GetBiblioMultiple)
}

// GetClaimsBulk retrieves claims for multiple publications with auto-batching.
// Automatically splits requests into batches of 100 (EPO limit) and processes them sequentially.
func (c *Client) GetClaimsBulk(ctx context.Context, refType, format string, numbers []string, opts *BulkOptions) ([]string, error) {
	return c.executeBulkRequest(ctx, refType, format, numbers, opts, c.GetClaimsMultiple)
}

// GetDescriptionsBulk retrieves descriptions for multiple publications with auto-batching.
// Automatically splits requests into batches of 100 (EPO limit) and processes them sequentially.
func (c *Client) GetDescriptionsBulk(ctx context.Context, refType, format string, numbers []string, opts *BulkOptions) ([]string, error) {
	return c.executeBulkRequest(ctx, refType, format, numbers, opts, c.GetDescriptionMultiple)
}

// GetAbstractsBulk retrieves abstracts for multiple publications with auto-batching.
// Automatically splits requests into batches of 100 (EPO limit) and processes them sequentially.
func (c *Client) GetAbstractsBulk(ctx context.Context, refType, format string, numbers []string, opts *BulkOptions) ([]string, error) {
	return c.executeBulkRequest(ctx, refType, format, numbers, opts, c.GetAbstractMultiple)
}

// GetFulltextsBulk retrieves fulltext for multiple publications with auto-batching.
// Automatically splits requests into batches of 100 (EPO limit) and processes them sequentially.
func (c *Client) GetFulltextsBulk(ctx context.Context, refType, format string, numbers []string, opts *BulkOptions) ([]string, error) {
	return c.executeBulkRequest(ctx, refType, format, numbers, opts, c.GetFulltextMultiple)
}
