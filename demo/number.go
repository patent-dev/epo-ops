package main

import (
	"fmt"

	ops "github.com/patent-dev/epo-ops"
)

// demoNumber demonstrates Number Services endpoints (3 endpoints)
func demoNumber(demo *DemoContext) {
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("Number Services (3 endpoints)")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// 1. ConvertPatentNumber (docdb -> epodoc)
	runEndpoint(demo, "convert_patent_number_epodoc", "ConvertPatentNumber (to epodoc)",
		func() ([]byte, error) {
			result, err := demo.Client.ConvertPatentNumber(demo.Ctx, ops.RefTypePublication, ops.FormatDocDB, demo.Patent, ops.FormatEPODOC)
			return []byte(result), err
		},
		FormatRequestDescription("ConvertPatentNumber", map[string]string{
			"refType":      ops.RefTypePublication,
			"inputFormat":  ops.FormatDocDB,
			"number":       demo.Patent,
			"outputFormat": ops.FormatEPODOC,
		}))

	// Also demonstrate docdb -> original conversion
	runEndpoint(demo, "convert_patent_number_original", "ConvertPatentNumber (to original)",
		func() ([]byte, error) {
			result, err := demo.Client.ConvertPatentNumber(demo.Ctx, ops.RefTypePublication, ops.FormatDocDB, demo.Patent, ops.FormatOriginal)
			return []byte(result), err
		},
		FormatRequestDescription("ConvertPatentNumber", map[string]string{
			"refType":      ops.RefTypePublication,
			"inputFormat":  ops.FormatDocDB,
			"number":       demo.Patent,
			"outputFormat": ops.FormatOriginal,
		}))

	// 2. ConvertPatentNumberMultiple (POST - bulk conversion)
	runEndpoint(demo, "convert_patent_number_multiple", "ConvertPatentNumberMultiple",
		func() ([]byte, error) {
			numbers := GetBulkTestPatents(demo.Patent)
			result, err := demo.Client.ConvertPatentNumberMultiple(demo.Ctx, ops.RefTypePublication, ops.FormatDocDB, numbers, ops.FormatEPODOC)
			return []byte(result), err
		},
		FormatRequestDescription("ConvertPatentNumberMultiple", map[string]string{
			"refType":      ops.RefTypePublication,
			"inputFormat":  ops.FormatDocDB,
			"numbers":      "bulk test patents from CSV",
			"outputFormat": ops.FormatEPODOC,
		}))

	fmt.Println()
}
