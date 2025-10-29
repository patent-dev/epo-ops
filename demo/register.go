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

	// 1. GetRegisterBiblioRaw (GET) - Format param is "epodoc" but number must have dots (docdb)
	runEndpoint(demo, "get_register_biblio", "GetRegisterBiblioRaw",
		func() ([]byte, error) {
			result, err := demo.Client.GetRegisterBiblioRaw(demo.Ctx, ops.RefTypePublication, ops.FormatEPODOC, demo.Patent)
			return []byte(result), err
		},
		FormatRequestDescription("GetRegisterBiblioRaw", map[string]string{
			"refType": ops.RefTypePublication,
			"format":  ops.FormatEPODOC,
			"number":  demo.Patent,
		}))

	// 2. GetRegisterBiblioMultipleRaw (POST - bulk register biblio)
	runEndpoint(demo, "get_register_biblio_multiple", "GetRegisterBiblioMultipleRaw",
		func() ([]byte, error) {
			numbers := GetBulkTestPatents(demo.Patent)
			result, err := demo.Client.GetRegisterBiblioMultipleRaw(demo.Ctx, ops.RefTypePublication, ops.FormatEPODOC, numbers)
			return []byte(result), err
		},
		FormatRequestDescription("GetRegisterBiblioMultipleRaw", map[string]string{
			"refType": ops.RefTypePublication,
			"format":  ops.FormatEPODOC,
			"numbers": "bulk test patents from CSV",
		}))

	// 3. GetRegisterEvents (GET) - Uses epodoc format parameter but docdb number (with dots)
	runEndpoint(demo, "get_register_events", "GetRegisterEvents",
		func() ([]byte, error) {
			result, err := demo.Client.GetRegisterEventsRaw(demo.Ctx, ops.RefTypePublication, ops.FormatEPODOC, demo.Patent)
			return []byte(result), err
		},
		FormatRequestDescription("GetRegisterEvents", map[string]string{
			"refType": ops.RefTypePublication,
			"format":  ops.FormatEPODOC,
			"number":  demo.Patent,
		}))

	// 4. GetRegisterEventsMultipleRaw (POST - bulk register events)
	runEndpoint(demo, "get_register_events_multiple", "GetRegisterEventsMultipleRaw",
		func() ([]byte, error) {
			numbers := GetBulkTestPatents(demo.Patent)
			result, err := demo.Client.GetRegisterEventsMultipleRaw(demo.Ctx, ops.RefTypePublication, ops.FormatEPODOC, numbers)
			return []byte(result), err
		},
		FormatRequestDescription("GetRegisterEventsMultipleRaw", map[string]string{
			"refType": ops.RefTypePublication,
			"format":  ops.FormatEPODOC,
			"numbers": "bulk test patents from CSV",
		}))

	// 5. GetRegisterProceduralStepsRaw (GET)
	runEndpoint(demo, "get_register_procedural_steps", "GetRegisterProceduralStepsRaw",
		func() ([]byte, error) {
			result, err := demo.Client.GetRegisterProceduralStepsRaw(demo.Ctx, ops.RefTypePublication, ops.FormatEPODOC, demo.Patent)
			return []byte(result), err
		},
		FormatRequestDescription("GetRegisterProceduralStepsRaw", map[string]string{
			"refType": ops.RefTypePublication,
			"format":  ops.FormatEPODOC,
			"number":  demo.Patent,
		}))

	// 6. GetRegisterProceduralStepsMultipleRaw (POST)
	runEndpoint(demo, "get_register_procedural_steps_multiple", "GetRegisterProceduralStepsMultipleRaw",
		func() ([]byte, error) {
			numbers := GetBulkTestPatents(demo.Patent)
			result, err := demo.Client.GetRegisterProceduralStepsMultipleRaw(demo.Ctx, ops.RefTypePublication, ops.FormatEPODOC, numbers)
			return []byte(result), err
		},
		FormatRequestDescription("GetRegisterProceduralStepsMultipleRaw", map[string]string{
			"refType": ops.RefTypePublication,
			"format":  ops.FormatEPODOC,
			"numbers": "bulk test patents from CSV",
		}))

	// 7. GetRegisterUNIPRaw (GET) - Unified Patent Package
	runEndpoint(demo, "get_register_unip", "GetRegisterUNIPRaw",
		func() ([]byte, error) {
			result, err := demo.Client.GetRegisterUNIPRaw(demo.Ctx, ops.RefTypePublication, ops.FormatEPODOC, demo.Patent)
			return []byte(result), err
		},
		FormatRequestDescription("GetRegisterUNIPRaw", map[string]string{
			"refType": ops.RefTypePublication,
			"format":  ops.FormatEPODOC,
			"number":  demo.Patent,
		}))

	// 8. GetRegisterUNIPMultipleRaw (POST)
	runEndpoint(demo, "get_register_unip_multiple", "GetRegisterUNIPMultipleRaw",
		func() ([]byte, error) {
			numbers := GetBulkTestPatents(demo.Patent)
			result, err := demo.Client.GetRegisterUNIPMultipleRaw(demo.Ctx, ops.RefTypePublication, ops.FormatEPODOC, numbers)
			return []byte(result), err
		},
		FormatRequestDescription("GetRegisterUNIPMultipleRaw", map[string]string{
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
