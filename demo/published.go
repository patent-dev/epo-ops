package main

import (
	"fmt"

	ops "github.com/patent-dev/epo-ops"
)

// demoPublished demonstrates all Published Data Services endpoints (15 endpoints)
func demoPublished(demo *DemoContext) {
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("Published Data Services (15 endpoints)")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// 1. GetBiblio (GET)
	runEndpoint(demo, "get_biblio", "GetBiblio",
		func() ([]byte, error) {
			result, err := demo.Client.GetBiblioRaw(demo.Ctx, ops.RefTypePublication, ops.FormatDocDB, demo.Patent)
			return []byte(result), err
		},
		FormatRequestDescription("GetBiblio", map[string]string{
			"refType": ops.RefTypePublication,
			"format":  ops.FormatDocDB,
			"number":  demo.Patent,
		}))

	// 2. GetBiblioMultiple (POST)
	runEndpoint(demo, "get_biblio_multiple", "GetBiblioMultiple",
		func() ([]byte, error) {
			numbers := GetBulkTestPatents(demo.Patent)
			result, err := demo.Client.GetBiblioMultiple(demo.Ctx, ops.RefTypePublication, ops.FormatDocDB, numbers)
			return []byte(result), err
		},
		FormatRequestDescription("GetBiblioMultiple", map[string]string{
			"refType": ops.RefTypePublication,
			"format":  ops.FormatDocDB,
			"numbers": "bulk test patents from CSV",
		}))

	// 3. GetAbstract (GET)
	runEndpoint(demo, "get_abstract", "GetAbstractRaw",
		func() ([]byte, error) {
			result, err := demo.Client.GetAbstractRaw(demo.Ctx, ops.RefTypePublication, ops.FormatDocDB, demo.Patent)
			return []byte(result), err
		},
		FormatRequestDescription("GetAbstractRaw", map[string]string{
			"refType": ops.RefTypePublication,
			"format":  ops.FormatDocDB,
			"number":  demo.Patent,
		}))

	// 4. GetAbstractMultiple (POST)
	runEndpoint(demo, "get_abstract_multiple", "GetAbstractMultiple",
		func() ([]byte, error) {
			numbers := GetBulkTestPatents(demo.Patent)
			result, err := demo.Client.GetAbstractMultiple(demo.Ctx, ops.RefTypePublication, ops.FormatDocDB, numbers)
			return []byte(result), err
		},
		FormatRequestDescription("GetAbstractMultiple", map[string]string{
			"refType": ops.RefTypePublication,
			"format":  ops.FormatDocDB,
			"numbers": "bulk test patents from CSV",
		}))

	// 5. GetClaims (GET)
	runEndpoint(demo, "get_claims", "GetClaimsRaw",
		func() ([]byte, error) {
			result, err := demo.Client.GetClaimsRaw(demo.Ctx, ops.RefTypePublication, ops.FormatDocDB, demo.Patent)
			return []byte(result), err
		},
		FormatRequestDescription("GetClaimsRaw", map[string]string{
			"refType": ops.RefTypePublication,
			"format":  ops.FormatDocDB,
			"number":  demo.Patent,
		}))

	// 6. GetClaimsMultiple (POST)
	runEndpoint(demo, "get_claims_multiple", "GetClaimsMultiple",
		func() ([]byte, error) {
			numbers := GetBulkTestPatents(demo.Patent)
			result, err := demo.Client.GetClaimsMultiple(demo.Ctx, ops.RefTypePublication, ops.FormatDocDB, numbers)
			return []byte(result), err
		},
		FormatRequestDescription("GetClaimsMultiple", map[string]string{
			"refType": ops.RefTypePublication,
			"format":  ops.FormatDocDB,
			"numbers": "bulk test patents from CSV",
		}))

	// 7. GetDescription (GET)
	runEndpoint(demo, "get_description", "GetDescription",
		func() ([]byte, error) {
			result, err := demo.Client.GetDescription(demo.Ctx, ops.RefTypePublication, ops.FormatDocDB, demo.Patent)
			return []byte(result), err
		},
		FormatRequestDescription("GetDescription", map[string]string{
			"refType": ops.RefTypePublication,
			"format":  ops.FormatDocDB,
			"number":  demo.Patent,
		}))

	// 8. GetDescriptionMultiple (POST)
	runEndpoint(demo, "get_description_multiple", "GetDescriptionMultiple",
		func() ([]byte, error) {
			numbers := GetBulkTestPatents(demo.Patent)
			result, err := demo.Client.GetDescriptionMultiple(demo.Ctx, ops.RefTypePublication, ops.FormatDocDB, numbers)
			return []byte(result), err
		},
		FormatRequestDescription("GetDescriptionMultiple", map[string]string{
			"refType": ops.RefTypePublication,
			"format":  ops.FormatDocDB,
			"numbers": "bulk test patents from CSV",
		}))

	// 9. GetFulltext (GET)
	runEndpoint(demo, "get_fulltext", "GetFulltext",
		func() ([]byte, error) {
			result, err := demo.Client.GetFulltext(demo.Ctx, ops.RefTypePublication, ops.FormatDocDB, demo.Patent)
			return []byte(result), err
		},
		FormatRequestDescription("GetFulltext", map[string]string{
			"refType": ops.RefTypePublication,
			"format":  ops.FormatDocDB,
			"number":  demo.Patent,
		}))

	// 10. GetFulltextMultiple (POST)
	runEndpoint(demo, "get_fulltext_multiple", "GetFulltextMultiple",
		func() ([]byte, error) {
			numbers := GetBulkTestPatents(demo.Patent)
			result, err := demo.Client.GetFulltextMultiple(demo.Ctx, ops.RefTypePublication, ops.FormatDocDB, numbers)
			return []byte(result), err
		},
		FormatRequestDescription("GetFulltextMultiple", map[string]string{
			"refType": ops.RefTypePublication,
			"format":  ops.FormatDocDB,
			"numbers": "bulk test patents from CSV",
		}))

	// 11. GetPublishedEquivalents (GET) - Simple family
	runEndpoint(demo, "get_published_equivalents", "GetPublishedEquivalents",
		func() ([]byte, error) {
			result, err := demo.Client.GetPublishedEquivalents(demo.Ctx, ops.RefTypePublication, ops.FormatDocDB, demo.Patent)
			return []byte(result), err
		},
		FormatRequestDescription("GetPublishedEquivalents", map[string]string{
			"refType": ops.RefTypePublication,
			"format":  ops.FormatDocDB,
			"number":  demo.Patent,
		}))

	// 12. GetPublishedEquivalentsMultiple (POST)
	runEndpoint(demo, "get_published_equivalents_multiple", "GetPublishedEquivalentsMultiple",
		func() ([]byte, error) {
			numbers := GetBulkTestPatents(demo.Patent)
			result, err := demo.Client.GetPublishedEquivalentsMultiple(demo.Ctx, ops.RefTypePublication, ops.FormatDocDB, numbers)
			return []byte(result), err
		},
		FormatRequestDescription("GetPublishedEquivalentsMultiple", map[string]string{
			"refType": ops.RefTypePublication,
			"format":  ops.FormatDocDB,
			"numbers": "bulk test patents from CSV",
		}))

	// 13. GetImageInquiry (GET) - Check which images are available
	runEndpoint(demo, "get_image_inquiry", "GetImageInquiry",
		func() ([]byte, error) {
			inquiry, err := demo.Client.GetImageInquiry(demo.Ctx, ops.RefTypePublication, ops.FormatDocDB, demo.Patent)
			if err != nil {
				return nil, err
			}
			// Format as text
			result := fmt.Sprintf("Image Inquiry for %s\n", demo.Patent)
			result += fmt.Sprintf("Document Instances: %d\n", len(inquiry.DocumentInstances))
			for i, inst := range inquiry.DocumentInstances {
				result += fmt.Sprintf("\nInstance %d:\n", i+1)
				result += fmt.Sprintf("  Type: %s\n", inst.DocType)
				result += fmt.Sprintf("  Description: %s\n", inst.Description)
				result += fmt.Sprintf("  Link: %s\n", inst.Link)
				result += fmt.Sprintf("  Pages: %d\n", inst.NumberOfPages)
				result += fmt.Sprintf("  Formats: %v\n", inst.Formats)
			}
			return []byte(result), nil
		},
		FormatRequestDescription("GetImageInquiry", map[string]string{
			"refType": ops.RefTypePublication,
			"format":  ops.FormatDocDB,
			"number":  demo.Patent,
		}))

	// 14. GetFullCycleMultiple (POST) - Full publication cycle for multiple patents
	runEndpoint(demo, "get_full_cycle_multiple", "GetFullCycleMultiple",
		func() ([]byte, error) {
			numbers := GetBulkTestPatents(demo.Patent)
			result, err := demo.Client.GetFullCycleMultiple(demo.Ctx, ops.RefTypePublication, ops.FormatDocDB, numbers)
			return []byte(result), err
		},
		FormatRequestDescription("GetFullCycleMultiple", map[string]string{
			"refType": ops.RefTypePublication,
			"format":  ops.FormatDocDB,
			"numbers": "bulk test patents from CSV",
		}))

	fmt.Println()
}
