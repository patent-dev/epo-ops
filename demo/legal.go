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

	// 1. GetLegal (legal status events)
	runEndpoint(demo, "get_legal", "GetLegal",
		func() ([]byte, error) {
			result, err := demo.Client.GetLegal(demo.Ctx, ops.RefTypePublication, ops.FormatDocDB, demo.Patent)
			return []byte(result), err
		},
		FormatRequestDescription("GetLegal", map[string]string{
			"refType": ops.RefTypePublication,
			"format":  ops.FormatDocDB,
			"number":  demo.Patent,
		}))

	// 2. GetLegalMultiple (POST - bulk legal status)
	runEndpoint(demo, "get_legal_multiple", "GetLegalMultiple",
		func() ([]byte, error) {
			numbers := GetBulkTestPatents(demo.Patent)
			result, err := demo.Client.GetLegalMultiple(demo.Ctx, ops.RefTypePublication, ops.FormatDocDB, numbers)
			return []byte(result), err
		},
		FormatRequestDescription("GetLegalMultiple", map[string]string{
			"refType": ops.RefTypePublication,
			"format":  ops.FormatDocDB,
			"numbers": "bulk test patents from CSV",
		}))

	fmt.Println()
}
