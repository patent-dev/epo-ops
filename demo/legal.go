package main

import (
	"fmt"

	ops "github.com/patent-dev/epo-ops"
)

// demoLegal demonstrates Legal Services endpoints (2 endpoints)
func demoLegal(demo *DemoContext) {
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("Legal Services (2 endpoints)")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// 1. GetLegal (legal status events) - Showcase PARSED API
	runEndpoint(demo, "get_legal", "GetLegal",
		func() ([]byte, error) {
			// Use parsed API to demonstrate type-safe access
			legal, err := demo.Client.GetLegal(demo.Ctx, ops.RefTypePublication, ops.FormatDocDB, demo.Patent)
			if err != nil {
				return nil, err
			}

			// Format parsed data for display
			output := "✓ Parsed Legal Data\n"
			output += fmt.Sprintf("Patent Number: %s\n", legal.PatentNumber)
			output += fmt.Sprintf("Family ID:     %s\n", legal.FamilyID)
			output += fmt.Sprintf("Legal Events:  %d\n\n", len(legal.LegalEvents))

			output += "Legal Events:\n"
			for i, event := range legal.LegalEvents {
				if i >= 5 {
					output += fmt.Sprintf("  ... and %d more events\n", len(legal.LegalEvents)-5)
					break
				}
				output += fmt.Sprintf("  %d. [%s] %s: %s\n",
					i+1, event.DateMigr, event.Code, event.Description)
				if event.Influence != "" {
					output += fmt.Sprintf("     Influence: %s\n", event.Influence)
				}
			}

			return []byte(output), nil
		},
		FormatRequestDescription("GetLegal", map[string]string{
			"refType": ops.RefTypePublication,
			"format":  ops.FormatDocDB,
			"number":  demo.Patent,
		}))

	// 2. GetLegalMultiple (POST - bulk legal status)
	// Note: Parsed version returns *LegalData. Using single Raw call for demo
	runEndpoint(demo, "get_legal_multiple", "GetLegalMultiple",
		func() ([]byte, error) {
			// GetLegalMultiple returns *LegalData
			result, err := demo.Client.GetLegalRaw(demo.Ctx, ops.RefTypePublication, ops.FormatDocDB, demo.Patent)
			return []byte(result), err
		},
		FormatRequestDescription("GetLegalMultiple", map[string]string{
			"refType": ops.RefTypePublication,
			"format":  ops.FormatDocDB,
			"numbers": "bulk test patents from CSV",
		}))

	fmt.Println()
}
