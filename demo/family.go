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

	// 1. GetFamily (basic family members)
	runEndpoint(demo, "get_family", "GetFamily",
		func() ([]byte, error) {
			result, err := demo.Client.GetFamily(demo.Ctx, ops.RefTypePublication, ops.FormatDocDB, demo.Patent)
			return []byte(result), err
		},
		FormatRequestDescription("GetFamily", map[string]string{
			"refType": ops.RefTypePublication,
			"format":  ops.FormatDocDB,
			"number":  demo.Patent,
		}))

	// 2. GetFamilyWithBiblio (family + bibliographic data)
	runEndpoint(demo, "get_family_with_biblio", "GetFamilyWithBiblio",
		func() ([]byte, error) {
			result, err := demo.Client.GetFamilyWithBiblio(demo.Ctx, ops.RefTypePublication, ops.FormatDocDB, demo.Patent)
			return []byte(result), err
		},
		FormatRequestDescription("GetFamilyWithBiblio", map[string]string{
			"refType": ops.RefTypePublication,
			"format":  ops.FormatDocDB,
			"number":  demo.Patent,
		}))

	// 3. GetFamilyWithBiblioMultiple (POST - bulk family + biblio)
	runEndpoint(demo, "get_family_with_biblio_multiple", "GetFamilyWithBiblioMultiple",
		func() ([]byte, error) {
			numbers := GetBulkTestPatents(demo.Patent)
			result, err := demo.Client.GetFamilyWithBiblioMultiple(demo.Ctx, ops.RefTypePublication, ops.FormatDocDB, numbers)
			return []byte(result), err
		},
		FormatRequestDescription("GetFamilyWithBiblioMultiple", map[string]string{
			"refType": ops.RefTypePublication,
			"format":  ops.FormatDocDB,
			"numbers": "bulk test patents from CSV",
		}))

	// 4. GetFamilyWithLegal (family + legal status)
	runEndpoint(demo, "get_family_with_legal", "GetFamilyWithLegal",
		func() ([]byte, error) {
			result, err := demo.Client.GetFamilyWithLegal(demo.Ctx, ops.RefTypePublication, ops.FormatDocDB, demo.Patent)
			return []byte(result), err
		},
		FormatRequestDescription("GetFamilyWithLegal", map[string]string{
			"refType": ops.RefTypePublication,
			"format":  ops.FormatDocDB,
			"number":  demo.Patent,
		}))

	// 5. GetFamilyWithLegalMultiple (POST - bulk family + legal)
	runEndpoint(demo, "get_family_with_legal_multiple", "GetFamilyWithLegalMultiple",
		func() ([]byte, error) {
			numbers := GetBulkTestPatents(demo.Patent)
			result, err := demo.Client.GetFamilyWithLegalMultiple(demo.Ctx, ops.RefTypePublication, ops.FormatDocDB, numbers)
			return []byte(result), err
		},
		FormatRequestDescription("GetFamilyWithLegalMultiple", map[string]string{
			"refType": ops.RefTypePublication,
			"format":  ops.FormatDocDB,
			"numbers": "bulk test patents from CSV",
		}))

	fmt.Println()
}
