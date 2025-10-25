package main

import (
	"fmt"

	ops "github.com/patent-dev/epo-ops"
)

// demoRegister demonstrates all Register Services endpoints (10 endpoints)
func demoRegister(demo *DemoContext) {
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("Register Services (10 endpoints)")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// 1. GetRegisterBiblio (GET) - Format param is "epodoc" but number must have dots (docdb)
	runEndpoint(demo, "get_register_biblio", "GetRegisterBiblio",
		func() ([]byte, error) {
			result, err := demo.Client.GetRegisterBiblio(demo.Ctx, ops.RefTypePublication, ops.FormatEPODOC, demo.Patent)
			return []byte(result), err
		},
		FormatRequestDescription("GetRegisterBiblio", map[string]string{
			"refType": ops.RefTypePublication,
			"format":  ops.FormatEPODOC,
			"number":  demo.Patent,
		}))

	// 2. GetRegisterBiblioMultiple (POST - bulk register biblio)
	runEndpoint(demo, "get_register_biblio_multiple", "GetRegisterBiblioMultiple",
		func() ([]byte, error) {
			numbers := GetBulkTestPatents(demo.Patent)
			result, err := demo.Client.GetRegisterBiblioMultiple(demo.Ctx, ops.RefTypePublication, ops.FormatEPODOC, numbers)
			return []byte(result), err
		},
		FormatRequestDescription("GetRegisterBiblioMultiple", map[string]string{
			"refType": ops.RefTypePublication,
			"format":  ops.FormatEPODOC,
			"numbers": "bulk test patents from CSV",
		}))

	// 3. GetRegisterEvents (GET) - Uses epodoc format parameter but docdb number (with dots)
	runEndpoint(demo, "get_register_events", "GetRegisterEvents",
		func() ([]byte, error) {
			result, err := demo.Client.GetRegisterEvents(demo.Ctx, ops.RefTypePublication, ops.FormatEPODOC, demo.Patent)
			return []byte(result), err
		},
		FormatRequestDescription("GetRegisterEvents", map[string]string{
			"refType": ops.RefTypePublication,
			"format":  ops.FormatEPODOC,
			"number":  demo.Patent,
		}))

	// 4. GetRegisterEventsMultiple (POST - bulk register events)
	runEndpoint(demo, "get_register_events_multiple", "GetRegisterEventsMultiple",
		func() ([]byte, error) {
			numbers := GetBulkTestPatents(demo.Patent)
			result, err := demo.Client.GetRegisterEventsMultiple(demo.Ctx, ops.RefTypePublication, ops.FormatEPODOC, numbers)
			return []byte(result), err
		},
		FormatRequestDescription("GetRegisterEventsMultiple", map[string]string{
			"refType": ops.RefTypePublication,
			"format":  ops.FormatEPODOC,
			"numbers": "bulk test patents from CSV",
		}))

	// 5. GetRegisterProceduralSteps (GET)
	runEndpoint(demo, "get_register_procedural_steps", "GetRegisterProceduralSteps",
		func() ([]byte, error) {
			result, err := demo.Client.GetRegisterProceduralSteps(demo.Ctx, ops.RefTypePublication, ops.FormatEPODOC, demo.Patent)
			return []byte(result), err
		},
		FormatRequestDescription("GetRegisterProceduralSteps", map[string]string{
			"refType": ops.RefTypePublication,
			"format":  ops.FormatEPODOC,
			"number":  demo.Patent,
		}))

	// 6. GetRegisterProceduralStepsMultiple (POST)
	runEndpoint(demo, "get_register_procedural_steps_multiple", "GetRegisterProceduralStepsMultiple",
		func() ([]byte, error) {
			numbers := GetBulkTestPatents(demo.Patent)
			result, err := demo.Client.GetRegisterProceduralStepsMultiple(demo.Ctx, ops.RefTypePublication, ops.FormatEPODOC, numbers)
			return []byte(result), err
		},
		FormatRequestDescription("GetRegisterProceduralStepsMultiple", map[string]string{
			"refType": ops.RefTypePublication,
			"format":  ops.FormatEPODOC,
			"numbers": "bulk test patents from CSV",
		}))

	// 7. GetRegisterUNIP (GET) - Unified Patent Package
	runEndpoint(demo, "get_register_unip", "GetRegisterUNIP",
		func() ([]byte, error) {
			result, err := demo.Client.GetRegisterUNIP(demo.Ctx, ops.RefTypePublication, ops.FormatEPODOC, demo.Patent)
			return []byte(result), err
		},
		FormatRequestDescription("GetRegisterUNIP", map[string]string{
			"refType": ops.RefTypePublication,
			"format":  ops.FormatEPODOC,
			"number":  demo.Patent,
		}))

	// 8. GetRegisterUNIPMultiple (POST)
	runEndpoint(demo, "get_register_unip_multiple", "GetRegisterUNIPMultiple",
		func() ([]byte, error) {
			numbers := GetBulkTestPatents(demo.Patent)
			result, err := demo.Client.GetRegisterUNIPMultiple(demo.Ctx, ops.RefTypePublication, ops.FormatEPODOC, numbers)
			return []byte(result), err
		},
		FormatRequestDescription("GetRegisterUNIPMultiple", map[string]string{
			"refType": ops.RefTypePublication,
			"format":  ops.FormatEPODOC,
			"numbers": "bulk test patents from CSV",
		}))

	// 9. SearchRegister (without constituents)
	runEndpoint(demo, "search_register", "SearchRegister",
		func() ([]byte, error) {
			result, err := demo.Client.SearchRegister(demo.Ctx, "ti=battery", "1-5")
			return []byte(result), err
		},
		FormatRequestDescription("SearchRegister", map[string]string{
			"query": "ti=battery",
			"range": "1-5",
		}))

	// 10. SearchRegisterWithConstituent (with specific data type)
	runEndpoint(demo, "search_register_with_constituent", "SearchRegisterWithConstituent",
		func() ([]byte, error) {
			result, err := demo.Client.SearchRegisterWithConstituent(demo.Ctx, "biblio", "ti=solar", "1-3")
			return []byte(result), err
		},
		FormatRequestDescription("SearchRegisterWithConstituent", map[string]string{
			"constituent": "biblio",
			"query":       "ti=solar",
			"range":       "1-3",
		}))

	fmt.Println()
}
