package main

import (
	"fmt"

	ops "github.com/patent-dev/epo-ops"
)

// demoSearch demonstrates all Search Services endpoints (2 endpoints)
func demoSearch(demo *DemoContext) {
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("Search Services (2 endpoints)")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// 1. Search (without constituents)
	runEndpoint(demo, "search", "Search",
		func() ([]byte, error) {
			result, err := demo.Client.Search(demo.Ctx, "ti=battery", "1-5")
			return []byte(result), err
		},
		FormatRequestDescription("Search", map[string]string{
			"query": "ti=battery",
			"range": "1-5",
		}))

	// 2. SearchWithConstituent (with specific data type)
	runEndpoint(demo, "search_with_constituent", "SearchWithConstituent",
		func() ([]byte, error) {
			result, err := demo.Client.SearchWithConstituent(demo.Ctx, ops.ConstituentBiblio, "pa=Siemens", "1-3")
			return []byte(result), err
		},
		FormatRequestDescription("SearchWithConstituent", map[string]string{
			"constituent": ops.ConstituentBiblio,
			"query":       "pa=Siemens",
			"range":       "1-3",
		}))

	fmt.Println()
}
