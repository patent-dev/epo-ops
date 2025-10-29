package main

import (
	"fmt"

	ops "github.com/patent-dev/epo-ops"
)

// demoParsedAPI showcases the benefits of the parsed struct API
func demoParsedAPI(demo *DemoContext) {
	fmt.Println("\n📊 Parsed API Examples - Type-Safe Access")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("These examples demonstrate direct struct access without XML parsing")
	fmt.Println()

	// Example 1: Family traversal with type safety
	fmt.Println("🔍 Example 1: Type-Safe Family Analysis")
	fmt.Println("─────────────────────────────────────")
	family, err := demo.Client.GetFamily(demo.Ctx, ops.RefTypePublication, ops.FormatDocDB, demo.Patent)
	if err != nil {
		fmt.Printf("❌ Error: %v\n\n", err)
	} else {
		fmt.Printf("Patent Number: %s\n", family.PatentNumber)
		fmt.Printf("Family ID:     %s\n", family.FamilyID)
		fmt.Printf("Total Members: %d\n", len(family.Members))
		fmt.Printf("Has Legal:     %v\n", family.Legal)
		fmt.Println()

		// Type-safe iteration - no XML parsing needed!
		fmt.Println("Family Members:")
		for i, member := range family.Members {
			if i >= 5 {
				fmt.Printf("  ... and %d more members\n", len(family.Members)-5)
				break
			}
			fmt.Printf("  %d. %s%s%s (%s)\n",
				i+1, member.Country, member.DocNumber, member.Kind, member.Date)

			// Access nested data with type safety
			if member.ApplicationRef.DocNumber != "" {
				fmt.Printf("     Application: %s%s (%s)\n",
					member.ApplicationRef.Country,
					member.ApplicationRef.DocNumber,
					member.ApplicationRef.Date)
			}

			// Show priority claims if available
			if len(member.PriorityClaims) > 0 {
				for _, priority := range member.PriorityClaims {
					fmt.Printf("     Priority: %s%s (%s) Active: %s\n",
						priority.Country,
						priority.DocNumber,
						priority.Date,
						priority.Active)
				}
			}
		}
		fmt.Println()
	}

	// Example 2: Legal event analysis
	fmt.Println("⚖️  Example 2: Legal Event Timeline")
	fmt.Println("─────────────────────────────────────")
	legal, err := demo.Client.GetLegal(demo.Ctx, ops.RefTypePublication, ops.FormatDocDB, demo.Patent)
	if err != nil {
		fmt.Printf("❌ Error: %v\n\n", err)
	} else {
		fmt.Printf("Patent:        %s\n", legal.PatentNumber)
		fmt.Printf("Family ID:     %s\n", legal.FamilyID)
		fmt.Printf("Legal Events:  %d\n", len(legal.LegalEvents))
		fmt.Println()

		// Direct access to legal event data
		fmt.Println("Recent Legal Events:")
		for i, event := range legal.LegalEvents {
			if i >= 5 {
				fmt.Printf("  ... and %d more events\n", len(legal.LegalEvents)-5)
				break
			}
			fmt.Printf("  %d. [%s] %s: %s\n",
				i+1, event.DateMigr, event.Code, event.Description)
			if event.Influence != "" {
				fmt.Printf("     Influence: %s\n", event.Influence)
			}
			// Show custom L*EP fields if available
			if len(event.Fields) > 0 {
				fmt.Printf("     Additional fields: %d\n", len(event.Fields))
			}
		}
		fmt.Println()
	}

	// Example 3: Search with immediate access
	fmt.Println("🔎 Example 3: Search with Structured Results")
	fmt.Println("─────────────────────────────────────")
	results, err := demo.Client.Search(demo.Ctx, "ti=battery", "1-5")
	if err != nil {
		fmt.Printf("❌ Error: %v\n\n", err)
	} else {
		fmt.Printf("Query:         %s\n", results.Query)
		fmt.Printf("Total Results: %d\n", results.TotalCount)
		fmt.Printf("Range:         %d-%d\n", results.RangeBegin, results.RangeEnd)
		fmt.Printf("Returned:      %d patents\n", len(results.Results))
		fmt.Println()

		// Direct iteration over search results
		fmt.Println("Search Results:")
		for i, result := range results.Results {
			fmt.Printf("  %d. %s%s (Family: %s, System: %s)\n",
				i+1,
				result.Country,
				result.DocNumber,
				result.FamilyID,
				result.System)
			if result.Title != "" {
				fmt.Printf("     %s\n", result.Title)
			}
		}
		fmt.Println()
	}

	// Example 4: Description with paragraphs
	fmt.Println("📄 Example 4: Description with Structured Text")
	fmt.Println("─────────────────────────────────────")
	description, err := demo.Client.GetDescription(demo.Ctx, ops.RefTypePublication, ops.FormatDocDB, demo.Patent)
	if err != nil {
		fmt.Printf("❌ Error: %v\n\n", err)
	} else {
		fmt.Printf("Patent:     %s\n", description.PatentNumber)
		fmt.Printf("Country:    %s\n", description.Country)
		fmt.Printf("DocNumber:  %s\n", description.DocNumber)
		fmt.Printf("Language:   %s\n", description.Language)
		fmt.Printf("Paragraphs: %d\n", len(description.Paragraphs))
		fmt.Println()

		// Access structured paragraph data
		fmt.Println("First 3 Paragraphs:")
		for i, para := range description.Paragraphs {
			if i >= 3 {
				fmt.Printf("  ... and %d more paragraphs\n", len(description.Paragraphs)-3)
				break
			}
			text := para.Text
			if len(text) > 100 {
				text = text[:100] + "..."
			}
			fmt.Printf("  [%s] %s\n", para.ID, text)
		}
		fmt.Println()
	}

	// Example 5: Equivalents (simple family)
	fmt.Println("🌐 Example 5: Published Equivalents")
	fmt.Println("─────────────────────────────────────")
	equivalents, err := demo.Client.GetPublishedEquivalents(demo.Ctx, ops.RefTypePublication, ops.FormatDocDB, demo.Patent)
	if err != nil {
		fmt.Printf("❌ Error: %v\n\n", err)
	} else {
		fmt.Printf("Patent:      %s\n", equivalents.PatentNumber)
		fmt.Printf("Equivalents: %d\n", len(equivalents.Equivalents))
		fmt.Println()

		fmt.Println("Equivalent Publications:")
		for i, equiv := range equivalents.Equivalents {
			if i >= 10 {
				fmt.Printf("  ... and %d more equivalents\n", len(equivalents.Equivalents)-10)
				break
			}
			fmt.Printf("  %d. %s%s (Kind: %s)\n",
				i+1, equiv.Country, equiv.DocNumber, equiv.Kind)
		}
		fmt.Println()
	}

	// Summary
	fmt.Println("✅ Benefits of Parsed API:")
	fmt.Println("  • No manual XML parsing required")
	fmt.Println("  • Type-safe field access")
	fmt.Println("  • IDE autocomplete support")
	fmt.Println("  • Compiler catches field name errors")
	fmt.Println("  • Cleaner, more readable code")
	fmt.Println()
	fmt.Println("💡 Tip: Use *Raw() methods if you need XML for debugging or storage")
	fmt.Println()
}
