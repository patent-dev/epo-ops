// Package main provides a comprehensive demo of the EPO OPS Go client.
//
// This demo showcases ALL 46 endpoints of the EPO OPS v3.2 API:
// - 15 Published Data Services
// - 2 Search Services
// - 5 Family Services
// - 2 Legal Services
// - 10 Register Services
// - 6 Classification Services
// - 3 Number Services
// - 1 Usage Service
// - 2 Image Services (+ 1 TIFF conversion utility)
//
// All request/response pairs are saved to demo/examples/ for reference.
//
// Usage:
//
//	export EPO_OPS_CONSUMER_KEY="your-key"
//	export EPO_OPS_CONSUMER_SECRET="your-secret"
//
//	# Run all demos
//	./demo
//
//	# Run specific service demos
//	./demo -service=published
//	./demo -service=classification
//
//	# Use different patent
//	./demo -patent=EP1000000A1
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	ops "github.com/patent-dev/epo-ops"
)

var (
	consumerKey    = flag.String("key", os.Getenv("EPO_OPS_CONSUMER_KEY"), "EPO OPS consumer key")
	consumerSecret = flag.String("secret", os.Getenv("EPO_OPS_CONSUMER_SECRET"), "EPO OPS consumer secret")
	patentNumber   = flag.String("patent", "EP.2400812.A1", "Patent number to demonstrate")
	serviceFilter  = flag.String("service", "", "Filter to specific service (published, search, family, legal, register, classification, number, usage, images, parsed)")
	endpointFilter = flag.String("endpoint", "", "Run specific endpoint only (use -list to see all endpoint names)")
	listEndpoints  = flag.Bool("list", false, "List all 38 endpoint names and exit")
	examplesDir    = flag.String("examples", "examples", "Directory to save examples")
	skipSave       = flag.Bool("no-save", false, "Skip saving request/response files")
)

// DemoContext holds shared context for all demos
type DemoContext struct {
	Client         *ops.Client
	Ctx            context.Context
	Saver          *ExampleSaver
	Patent         string // In docdb format (e.g., EP.2533477.B1)
	SkipSave       bool
	EndpointFilter string // Filter to specific endpoint name
	SuccessCount   int
	FailureCount   int
	TotalCount     int
}

// ToEpodoc converts docdb format (EP.2533477.B1) to epodoc (EP2533477B1)
func (d *DemoContext) ToEpodoc() string {
	return SanitizePatentNumber(d.Patent)
}

func main() {
	flag.Parse()

	// Handle -list flag
	if *listEndpoints {
		listAllEndpoints()
		return
	}

	if *consumerKey == "" || *consumerSecret == "" {
		log.Fatal("Error: EPO OPS credentials are required\n" +
			"Set EPO_OPS_CONSUMER_KEY and EPO_OPS_CONSUMER_SECRET environment variables\n" +
			"or use -key and -secret flags")
	}

	printBanner()

	// Create client
	config := &ops.Config{
		ConsumerKey:    *consumerKey,
		ConsumerSecret: *consumerSecret,
		Timeout:        5 * time.Minute, // Long timeout for slow endpoints like GetClassificationSchemaMultiple
	}

	client, err := ops.NewClient(config)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	// Setup context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Create example saver
	saver := NewExampleSaver(*examplesDir)

	// Initialize demo context
	demo := &DemoContext{
		Client:         client,
		Ctx:            ctx,
		Saver:          saver,
		Patent:         *patentNumber,
		SkipSave:       *skipSave,
		EndpointFilter: *endpointFilter,
	}

	fmt.Printf("✓ Client initialized\n")
	fmt.Printf("  Patent: %s\n", demo.Patent)
	if !*skipSave {
		absPath, _ := filepath.Abs(*examplesDir)
		fmt.Printf("  Examples: %s\n", absPath)
	}
	fmt.Println()

	// Run demos based on filter
	if *serviceFilter == "" || *serviceFilter == "published" {
		demoPublished(demo)
	}
	if *serviceFilter == "" || *serviceFilter == "search" {
		demoSearch(demo)
	}
	if *serviceFilter == "" || *serviceFilter == "family" {
		demoFamily(demo)
	}
	if *serviceFilter == "" || *serviceFilter == "legal" {
		demoLegal(demo)
	}
	if *serviceFilter == "" || *serviceFilter == "register" {
		demoRegister(demo)
	}
	if *serviceFilter == "" || *serviceFilter == "classification" {
		demoClassification(demo)
	}
	if *serviceFilter == "" || *serviceFilter == "number" {
		demoNumber(demo)
	}
	if *serviceFilter == "" || *serviceFilter == "usage" {
		demoUsage(demo)
	}
	if *serviceFilter == "" || *serviceFilter == "images" {
		demoImages(demo)
	}
	if *serviceFilter == "parsed" {
		demoParsedAPI(demo)
	}

	// Print summary
	printSummary(demo)

	// Show quota status
	demoQuotaMonitoring(demo.Client)
}

func printBanner() {
	fmt.Println("╔════════════════════════════════════════════════════════════════╗")
	fmt.Println("║              EPO OPS v3.2 Go Client - Demo                   ║")
	fmt.Println("║                  46 OpenAPI v3 Endpoints                      ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════╝")
	fmt.Println()
}

func printSummary(demo *DemoContext) {
	fmt.Println()
	fmt.Println("╔════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                      Demo Summary                             ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════╝")
	fmt.Printf("  Total Endpoints: %d\n", demo.TotalCount)
	fmt.Printf("  ✓ Success:       %d\n", demo.SuccessCount)
	fmt.Printf("  ✗ Failed:        %d\n", demo.FailureCount)
	fmt.Println()
}

func demoQuotaMonitoring(client *ops.Client) {
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("Quota Monitoring")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	quota := client.GetLastQuota()
	if quota == nil {
		fmt.Println("  ✗ No quota information available")
		fmt.Println()
		return
	}

	fmt.Printf("  Status: %s\n", getQuotaStatus(quota.Status))

	if quota.Individual.Limit > 0 {
		fmt.Printf("  Individual Quota (4 GB/week):\n")
		fmt.Printf("    Used:  %d bytes\n", quota.Individual.Used)
		fmt.Printf("    Limit: %d bytes\n", quota.Individual.Limit)
		fmt.Printf("    Usage: %.2f%%\n", quota.Individual.UsagePercent())
	}

	if quota.Registered.Limit > 0 {
		fmt.Printf("  Registered Quota:\n")
		fmt.Printf("    Used:  %d bytes\n", quota.Registered.Used)
		fmt.Printf("    Limit: %d bytes\n", quota.Registered.Limit)
		fmt.Printf("    Usage: %.2f%%\n", quota.Registered.UsagePercent())
	}

	fmt.Println()
}

func getQuotaStatus(status string) string {
	switch status {
	case "green":
		return "🟢 Green (healthy)"
	case "yellow":
		return "🟡 Yellow (moderate)"
	case "red":
		return "🔴 Red (high usage)"
	case "black":
		return "⚫ Black (exceeded)"
	default:
		return fmt.Sprintf("❓ %s", status)
	}
}

// runEndpoint is a helper to run an endpoint call and handle saving
func runEndpoint(demo *DemoContext, name, description string, fn func() ([]byte, error), requestDesc string) {
	// Check endpoint filter
	if demo.EndpointFilter != "" && demo.EndpointFilter != name {
		return // Skip this endpoint
	}

	demo.TotalCount++
	fmt.Printf("  → %s... ", description)

	data, err := fn()
	if err != nil {
		fmt.Printf("✗ %v\n", err)
		demo.FailureCount++
		return
	}

	// Detect format
	format := DetectFormat(data)

	// Validate based on format
	var validationErr error
	switch format {
	case FormatXML:
		validationErr = ValidateXML(data)
	case FormatJSON:
		validationErr = ValidateJSON(data)
	case FormatTIFF:
		validationErr = ValidateTIFF(data)
	}

	if validationErr != nil {
		fmt.Printf("✗ Invalid %s: %v\n", format, validationErr)
		demo.FailureCount++
		return
	}

	fmt.Printf("✓ %d bytes (%s)\n", len(data), format)
	demo.SuccessCount++

	// Save example
	if !demo.SkipSave {
		if err := demo.Saver.SaveExample(name, requestDesc, data, format); err != nil {
			fmt.Printf("    Warning: Failed to save example: %v\n", err)
		}
	}
}

// listAllEndpoints prints all 46 endpoint names
func listAllEndpoints() {
	fmt.Println("All 46 EPO OPS Endpoints:")
	fmt.Println()

	fmt.Println("Published Data Services (15):")
	fmt.Println("  get_biblio, get_biblio_multiple")
	fmt.Println("  get_abstract, get_abstract_multiple")
	fmt.Println("  get_claims, get_claims_multiple")
	fmt.Println("  get_description, get_description_multiple")
	fmt.Println("  get_fulltext, get_fulltext_multiple")
	fmt.Println("  get_published_equivalents, get_published_equivalents_multiple")
	fmt.Println("  get_image_inquiry, get_full_cycle_multiple")
	fmt.Println()

	fmt.Println("Search Services (2):")
	fmt.Println("  search, search_with_constituent")
	fmt.Println()

	fmt.Println("Family Services (5):")
	fmt.Println("  get_family, get_family_with_biblio, get_family_with_biblio_multiple")
	fmt.Println("  get_family_with_legal, get_family_with_legal_multiple")
	fmt.Println()

	fmt.Println("Legal Services (2):")
	fmt.Println("  get_legal, get_legal_multiple")
	fmt.Println()

	fmt.Println("Register Services (10):")
	fmt.Println("  get_register_biblio, get_register_biblio_multiple")
	fmt.Println("  get_register_events, get_register_events_multiple")
	fmt.Println("  get_register_procedural_steps, get_register_procedural_steps_multiple")
	fmt.Println("  get_register_unip, get_register_unip_multiple")
	fmt.Println("  search_register, search_register_with_constituent")
	fmt.Println()

	fmt.Println("Classification Services (6):")
	fmt.Println("  get_classification_schema, get_classification_schema_subclass")
	fmt.Println("  get_classification_schema_multiple, get_classification_media")
	fmt.Println("  get_classification_statistics, get_classification_mapping")
	fmt.Println()

	fmt.Println("Number Services (3):")
	fmt.Println("  convert_patent_number_epodoc, convert_patent_number_original")
	fmt.Println("  convert_patent_number_multiple")
	fmt.Println()

	fmt.Println("Usage Services (1):")
	fmt.Println("  get_usage_stats")
	fmt.Println()

	fmt.Println("Image Services (3):")
	fmt.Println("  get_image_thumbnail, get_image_post, tiff_to_png")
	fmt.Println()

	fmt.Println("Usage examples:")
	fmt.Println("  ./demo -endpoint=get_biblio")
	fmt.Println("  ./demo -endpoint=search")
	fmt.Println("  ./demo -service=register")
	fmt.Println()
}
