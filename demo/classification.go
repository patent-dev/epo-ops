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
			result, err := demo.Client.GetClassificationSchemaRaw(demo.Ctx, "H04W", false, false)
			return []byte(result), err
		},
		FormatRequestDescription("GetClassificationSchema", map[string]string{
			"class":      "H04W",
			"ancestors":  "false",
			"navigation": "false",
		}))

	// 2. GetClassificationSchemaSubclassRaw (GET)
	runEndpoint(demo, "get_classification_schema_subclass", "GetClassificationSchemaSubclassRaw",
		func() ([]byte, error) {
			result, err := demo.Client.GetClassificationSchemaSubclassRaw(demo.Ctx, "H04W4", "00", false, false)
			return []byte(result), err
		},
		FormatRequestDescription("GetClassificationSchemaSubclassRaw", map[string]string{
			"class":      "H04W4",
			"subclass":   "00",
			"ancestors":  "false",
			"navigation": "false",
		}))

	// 3. GetClassificationSchemaMultipleRaw (POST)
	runEndpoint(demo, "get_classification_schema_multiple", "GetClassificationSchemaMultipleRaw",
		func() ([]byte, error) {
			classes := []string{"H04W", "G06F", "A01B"}
			result, err := demo.Client.GetClassificationSchemaMultipleRaw(demo.Ctx, classes)
			return []byte(result), err
		},
		FormatRequestDescription("GetClassificationSchemaMultipleRaw", map[string]string{
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

	// 5. GetClassificationStatisticsRaw (GET)
	runEndpoint(demo, "get_classification_statistics", "GetClassificationStatisticsRaw",
		func() ([]byte, error) {
			result, err := demo.Client.GetClassificationStatisticsRaw(demo.Ctx, "H04W")
			return []byte(result), err
		},
		FormatRequestDescription("GetClassificationStatisticsRaw", map[string]string{
			"query": "H04W",
		}))

	// 6. GetClassificationMappingRaw (GET) - ECLA <-> CPC conversion
	runEndpoint(demo, "get_classification_mapping", "GetClassificationMappingRaw",
		func() ([]byte, error) {
			result, err := demo.Client.GetClassificationMappingRaw(demo.Ctx, "cpc", "H04W84", "18", "ecla", false)
			return []byte(result), err
		},
		FormatRequestDescription("GetClassificationMappingRaw", map[string]string{
			"inputFormat":  "cpc",
			"class":        "H04W84",
			"subclass":     "18",
			"outputFormat": "ecla",
			"additional":   "false",
		}))

	fmt.Println()
}
