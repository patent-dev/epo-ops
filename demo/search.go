package main

import (
	"fmt"

	ops "github.com/patent-dev/epo-ops"
)

// demoSearch demonstrates all Search Services endpoints (2 endpoints)
func demoSearch(demo *DemoContext) {
	fmt.Println("==============================================================")
	fmt.Println("Search Services (2 endpoints)")
	fmt.Println("==============================================================")

	// 1. Search (without constituents) - Showcase PARSED API
	runEndpoint(demo, "search", "Search",
		func() ([]byte, error) {
			// Use parsed API to demonstrate type-safe access
			results, err := demo.Client.Search(demo.Ctx, "ti=battery", "1-5")
			if err != nil {
				return nil, err
			}

			// Format parsed data for display
			output := "OK Parsed Search Results\n"
			output += fmt.Sprintf("Query:         %s\n", results.Query)
			output += fmt.Sprintf("Total Results: %d\n", results.TotalCount)
			output += fmt.Sprintf("Range:         %d-%d\n\n", results.RangeBegin, results.RangeEnd)

			output += "Results:\n"
			for i, result := range results.Results {
				output += fmt.Sprintf("  %d. %s%s (Family: %s)\n",
					i+1, result.Country, result.DocNumber, result.FamilyID)
				if result.Title != "" {
					title := result.Title
					if len(title) > 60 {
						title = title[:60] + "..."
					}
					output += fmt.Sprintf("     %s\n", title)
				}
			}

			return []byte(output), nil
		},
		FormatRequestDescription("Search", map[string]string{
			"query": "ti=battery",
			"range": "1-5",
		}))

	// 2. SearchWithConstituentRaw (every hit with its biblio constituent)
	// Note: the parsed SearchWithConstituent returns *SearchResultData
	runEndpoint(demo, "search_with_constituent", "SearchWithConstituentRaw",
		func() ([]byte, error) {
			result, err := demo.Client.SearchWithConstituentRaw(demo.Ctx, ops.ConstituentBiblio, "ti=battery", "1-3")
			return []byte(result), err
		},
		FormatRequestDescription("SearchWithConstituentRaw", map[string]string{
			"constituent": ops.ConstituentBiblio,
			"query":       "ti=battery",
			"range":       "1-3",
		}))

	fmt.Println()
}
