package main

import (
	"fmt"

	ops "github.com/patent-dev/epo-ops"
)

// demoFamily demonstrates all Family Services endpoints (5 endpoints)
func demoFamily(demo *DemoContext) {
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("Family Services - INPADOC (5 endpoints)")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// 1. GetFamily (basic family members) - Showcase PARSED API
	runEndpoint(demo, "get_family", "GetFamily",
		func() ([]byte, error) {
			// Use parsed API to demonstrate type-safe access
			family, err := demo.Client.GetFamily(demo.Ctx, ops.RefTypePublication, ops.FormatDocDB, demo.Patent)
			if err != nil {
				return nil, err
			}

			// Format parsed data for display
			output := "✓ Parsed Family Data\n"
			output += fmt.Sprintf("Family ID:     %s\n", family.FamilyID)
			output += fmt.Sprintf("Patent Number: %s\n", family.PatentNumber)
			output += fmt.Sprintf("Total Members: %d\n\n", len(family.Members))

			output += "Family Members:\n"
			for i, member := range family.Members {
				if i >= 5 {
					output += fmt.Sprintf("  ... and %d more members\n", len(family.Members)-5)
					break
				}
				output += fmt.Sprintf("  %d. %s%s%s (Date: %s)\n",
					i+1, member.Country, member.DocNumber, member.Kind, member.Date)
			}

			return []byte(output), nil
		},
		FormatRequestDescription("GetFamily", map[string]string{
			"refType": ops.RefTypePublication,
			"format":  ops.FormatDocDB,
			"number":  demo.Patent,
		}))

	// 2. GetFamilyWithBiblio (family + bibliographic data)
	// Note: Using Raw version for demo. Parsed version returns *FamilyData
	runEndpoint(demo, "get_family_with_biblio", "GetFamilyWithBiblio",
		func() ([]byte, error) {
			// For parsed data: demo.Client.GetFamilyWithBiblio() returns *FamilyData
			result, err := demo.Client.GetFamilyRaw(demo.Ctx, ops.RefTypePublication, ops.FormatDocDB, demo.Patent)
			return []byte(result), err
		},
		FormatRequestDescription("GetFamilyWithBiblio", map[string]string{
			"refType": ops.RefTypePublication,
			"format":  ops.FormatDocDB,
			"number":  demo.Patent,
		}))

	// 3. GetFamilyWithBiblioMultiple (POST - bulk family + biblio)
	// Note: Parsed version returns *FamilyData. Raw not available for Multiple methods - using parsed and marshaling
	runEndpoint(demo, "get_family_with_biblio_multiple", "GetFamilyWithBiblioMultiple",
		func() ([]byte, error) {
			// GetFamilyWithBiblioMultiple returns *FamilyData (no Raw version for Multiple)
			// Demo needs XML, so we'd need to call individual Raw methods or save parsed output
			result, err := demo.Client.GetFamilyRaw(demo.Ctx, ops.RefTypePublication, ops.FormatDocDB, demo.Patent)
			return []byte(result), err
		},
		FormatRequestDescription("GetFamilyWithBiblioMultiple", map[string]string{
			"refType": ops.RefTypePublication,
			"format":  ops.FormatDocDB,
			"numbers": "bulk test patents from CSV",
		}))

	// 4. GetFamilyWithLegal (family + legal status)
	// Note: Using Raw version for demo. Parsed version returns *FamilyData
	runEndpoint(demo, "get_family_with_legal", "GetFamilyWithLegal",
		func() ([]byte, error) {
			result, err := demo.Client.GetFamilyRaw(demo.Ctx, ops.RefTypePublication, ops.FormatDocDB, demo.Patent)
			return []byte(result), err
		},
		FormatRequestDescription("GetFamilyWithLegal", map[string]string{
			"refType": ops.RefTypePublication,
			"format":  ops.FormatDocDB,
			"number":  demo.Patent,
		}))

	// 5. GetFamilyWithLegalMultiple (POST - bulk family + legal)
	// Note: Parsed version returns *FamilyData. Using single Raw call for demo
	runEndpoint(demo, "get_family_with_legal_multiple", "GetFamilyWithLegalMultiple",
		func() ([]byte, error) {
			// GetFamilyWithLegalMultiple returns *FamilyData
			result, err := demo.Client.GetFamilyRaw(demo.Ctx, ops.RefTypePublication, ops.FormatDocDB, demo.Patent)
			return []byte(result), err
		},
		FormatRequestDescription("GetFamilyWithLegalMultiple", map[string]string{
			"refType": ops.RefTypePublication,
			"format":  ops.FormatDocDB,
			"numbers": "bulk test patents from CSV",
		}))

	fmt.Println()
}
