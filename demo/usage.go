package main

import (
	"fmt"
	"time"
)

// demoUsage demonstrates Usage Services endpoint (1 endpoint)
func demoUsage(demo *DemoContext) {
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("Usage Services (1 endpoint)")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// Format date range as dd/mm/yyyy~dd/mm/yyyy (yesterday to today)
	now := time.Now()
	yesterday := now.AddDate(0, 0, -1)
	dateStr := fmt.Sprintf("%02d/%02d/%d~%02d/%02d/%d",
		yesterday.Day(), int(yesterday.Month()), yesterday.Year(),
		now.Day(), int(now.Month()), now.Year())

	// 1. GetUsageStats (GET)
	runEndpoint(demo, "get_usage_stats", "GetUsageStats",
		func() ([]byte, error) {
			stats, err := demo.Client.GetUsageStats(demo.Ctx, dateStr)
			if err != nil {
				return nil, err
			}
			// Return formatted statistics as text
			result := fmt.Sprintf("Usage Statistics for %s\n", stats.TimeRange)
			result += fmt.Sprintf("Total Entries: %d\n", len(stats.Entries))
			if len(stats.Entries) > 0 {
				result += "\nFirst Entry:\n"
				entry := stats.Entries[0]
				result += fmt.Sprintf("  Timestamp: %d\n", entry.Timestamp)
				result += fmt.Sprintf("  Total Response Size: %d\n", entry.TotalResponseSize)
				result += fmt.Sprintf("  Message Count: %d\n", entry.MessageCount)
				if entry.Service != "" {
					result += fmt.Sprintf("  Service: %s\n", entry.Service)
				}
			}
			return []byte(result), nil
		},
		FormatRequestDescription("GetUsageStatistics", map[string]string{
			"timeRange": dateStr,
		}))

	fmt.Println()
}
