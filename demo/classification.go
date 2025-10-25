package main

import (
	"fmt"
)

// demoClassification demonstrates all Classification Services endpoints (5 endpoints)
func demoClassification(demo *DemoContext) {
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("Classification Services (5 endpoints)")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// 1. GetClassificationSchema (GET)
	runEndpoint(demo, "get_classification_schema", "GetClassificationSchema",
		func() ([]byte, error) {
			result, err := demo.Client.GetClassificationSchema(demo.Ctx, "H04W", false, false)
			return []byte(result), err
		},
		FormatRequestDescription("GetClassificationSchema", map[string]string{
			"class":      "H04W",
			"ancestors":  "false",
			"navigation": "false",
		}))

	// 2. GetClassificationSchemaSubclass (GET)
	runEndpoint(demo, "get_classification_schema_subclass", "GetClassificationSchemaSubclass",
		func() ([]byte, error) {
			result, err := demo.Client.GetClassificationSchemaSubclass(demo.Ctx, "H04W4", "00", false, false)
			return []byte(result), err
		},
		FormatRequestDescription("GetClassificationSchemaSubclass", map[string]string{
			"class":      "H04W4",
			"subclass":   "00",
			"ancestors":  "false",
			"navigation": "false",
		}))

	// 3. GetClassificationSchemaMultiple (POST)
	runEndpoint(demo, "get_classification_schema_multiple", "GetClassificationSchemaMultiple",
		func() ([]byte, error) {
			classes := []string{"H04W", "G06F", "A01B"}
			result, err := demo.Client.GetClassificationSchemaMultiple(demo.Ctx, classes)
			return []byte(result), err
		},
		FormatRequestDescription("GetClassificationSchemaMultiple", map[string]string{
			"classes": "H04W, G06F, A01B",
		}))

	// 4. GetClassificationMedia (GET) - CPC images
	runEndpoint(demo, "get_classification_media", "GetClassificationMedia",
		func() ([]byte, error) {
			return demo.Client.GetClassificationMedia(demo.Ctx, "1000.gif", false)
		},
		FormatRequestDescription("GetClassificationMedia", map[string]string{
			"mediaName":    "1000.gif",
			"asAttachment": "false",
		}))

	// 5. GetClassificationStatistics (GET)
	runEndpoint(demo, "get_classification_statistics", "GetClassificationStatistics",
		func() ([]byte, error) {
			result, err := demo.Client.GetClassificationStatistics(demo.Ctx, "H04W")
			return []byte(result), err
		},
		FormatRequestDescription("GetClassificationStatistics", map[string]string{
			"query": "H04W",
		}))

	// 6. GetClassificationMapping (GET) - ECLA <-> CPC conversion
	runEndpoint(demo, "get_classification_mapping", "GetClassificationMapping",
		func() ([]byte, error) {
			result, err := demo.Client.GetClassificationMapping(demo.Ctx, "cpc", "H04W84", "18", "ecla", false)
			return []byte(result), err
		},
		FormatRequestDescription("GetClassificationMapping", map[string]string{
			"inputFormat":  "cpc",
			"class":        "H04W84",
			"subclass":     "18",
			"outputFormat": "ecla",
			"additional":   "false",
		}))

	fmt.Println()
}
