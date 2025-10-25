package epo_ops

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

// QuotaInfo contains quota information from EPO OPS API responses.
type QuotaInfo struct {
	// Status indicates the overall quota status.
	// Possible values: "green" (<50%), "yellow" (50-75%), "red" (>75%), "black" (blocked)
	Status string

	// Individual quota for individual users (4GB/week for non-paying users)
	Individual QuotaMetric

	// Registered quota for registered users (>4GB/week for paying users)
	Registered QuotaMetric

	// Images quota (separate quota for image downloads)
	Images QuotaMetric

	// Raw header values for debugging
	ThrottlingControl string
	IndividualHeader  string
	RegisteredHeader  string
}

// QuotaMetric represents a quota metric with used and limit values.
type QuotaMetric struct {
	// Used is the amount of quota consumed
	Used int

	// Limit is the maximum quota allowed
	Limit int
}

// UsagePercent calculates the usage percentage (0-100).
func (q *QuotaMetric) UsagePercent() float64 {
	if q.Limit == 0 {
		return 0
	}
	return (float64(q.Used) / float64(q.Limit)) * 100
}

// ParseQuotaHeaders extracts quota information from HTTP response headers.
//
// EPO OPS returns quota information in these headers:
//   - X-Throttling-Control: Overall status (e.g., "green", "yellow", "red", "black")
//   - X-IndividualQuota: Individual quota in format "used=123,quota=456"
//   - X-RegisteredQuota: Registered quota in format "used=123,quota=456"
//   - X-ImagesQuota: Images quota in format "used=123,quota=456" (optional)
func ParseQuotaHeaders(headers http.Header) *QuotaInfo {
	info := &QuotaInfo{
		Status:            headers.Get("X-Throttling-Control"),
		ThrottlingControl: headers.Get("X-Throttling-Control"),
		IndividualHeader:  headers.Get("X-IndividualQuota"),
		RegisteredHeader:  headers.Get("X-RegisteredQuota"),
	}

	// Parse individual quota
	info.Individual = parseQuotaMetric(info.IndividualHeader)

	// Parse registered quota
	info.Registered = parseQuotaMetric(info.RegisteredHeader)

	// Parse images quota (optional)
	imagesHeader := headers.Get("X-ImagesQuota")
	if imagesHeader != "" {
		info.Images = parseQuotaMetric(imagesHeader)
	}

	// If status is not set but we have quota info, calculate it
	if info.Status == "" {
		info.Status = calculateStatus(&info.Individual, &info.Registered)
	}

	return info
}

// parseQuotaMetric parses a quota metric header value in format "used=123,quota=456"
func parseQuotaMetric(header string) QuotaMetric {
	metric := QuotaMetric{}

	if header == "" {
		return metric
	}

	// Parse key=value pairs separated by commas
	parts := strings.Split(header, ",")
	for _, part := range parts {
		kv := strings.Split(strings.TrimSpace(part), "=")
		if len(kv) != 2 {
			continue
		}

		key := strings.TrimSpace(kv[0])
		value := strings.TrimSpace(kv[1])

		switch key {
		case "used":
			if val, err := strconv.Atoi(value); err == nil {
				metric.Used = val
			}
		case "quota":
			if val, err := strconv.Atoi(value); err == nil {
				metric.Limit = val
			}
		}
	}

	return metric
}

// calculateStatus determines the status based on quota usage.
func calculateStatus(individual, registered *QuotaMetric) string {
	// Get the highest usage percentage
	maxPercent := 0.0

	if individual.Limit > 0 {
		percent := individual.UsagePercent()
		if percent > maxPercent {
			maxPercent = percent
		}
	}

	if registered.Limit > 0 {
		percent := registered.UsagePercent()
		if percent > maxPercent {
			maxPercent = percent
		}
	}

	// Determine status based on percentage
	switch {
	case maxPercent >= 100:
		return "black" // Quota exceeded, blocked
	case maxPercent >= 75:
		return "red" // High usage
	case maxPercent >= 50:
		return "yellow" // Medium usage
	default:
		return "green" // Low usage
	}
}

// quotaTracker holds the last quota information from API responses.
type quotaTracker struct {
	mu   sync.RWMutex
	last *QuotaInfo
}

// Update sets the last quota information.
func (qt *quotaTracker) Update(info *QuotaInfo) {
	qt.mu.Lock()
	defer qt.mu.Unlock()
	qt.last = info
}

// Get returns the last quota information (may be nil).
func (qt *quotaTracker) Get() *QuotaInfo {
	qt.mu.RLock()
	defer qt.mu.RUnlock()
	return qt.last
}

// ValidateTimeRange validates a time range string for the Usage Statistics API.
//
// Valid formats:
//   - Single date: "dd/mm/yyyy" (e.g., "01/01/2024")
//   - Date range: "dd/mm/yyyy~dd/mm/yyyy" (e.g., "01/01/2024~07/01/2024")
//
// Returns an error if the format is invalid.
func ValidateTimeRange(timeRange string) error {
	if timeRange == "" {
		return &ConfigError{Message: "time range cannot be empty"}
	}

	// Check if it's a date range (contains ~)
	if strings.Contains(timeRange, "~") {
		parts := strings.Split(timeRange, "~")
		if len(parts) != 2 {
			return &ConfigError{Message: "date range must contain exactly one tilde separator (~)"}
		}

		// Validate both dates
		if err := validateDate(strings.TrimSpace(parts[0])); err != nil {
			return fmt.Errorf("invalid start date: %w", err)
		}
		if err := validateDate(strings.TrimSpace(parts[1])); err != nil {
			return fmt.Errorf("invalid end date: %w", err)
		}

		return nil
	}

	// Single date
	return validateDate(timeRange)
}

// validateDate validates a single date in dd/mm/yyyy format.
func validateDate(date string) error {
	date = strings.TrimSpace(date)

	// Check format: dd/mm/yyyy (10 characters with slashes at positions 2 and 5)
	if len(date) != 10 {
		return &ConfigError{Message: "date must be in dd/mm/yyyy format (10 characters)"}
	}

	if date[2] != '/' || date[5] != '/' {
		return &ConfigError{Message: "date must use slashes (/) as separators"}
	}

	// Extract components
	day := date[0:2]
	month := date[3:5]
	year := date[6:10]

	// Validate day (01-31)
	d, err := strconv.Atoi(day)
	if err != nil || d < 1 || d > 31 {
		return &ConfigError{Message: "day must be between 01 and 31"}
	}

	// Validate month (01-12)
	m, err := strconv.Atoi(month)
	if err != nil || m < 1 || m > 12 {
		return &ConfigError{Message: "month must be between 01 and 12"}
	}

	// Validate year (4 digits)
	y, err := strconv.Atoi(year)
	if err != nil || y < 1000 || y > 9999 {
		return &ConfigError{Message: "year must be a 4-digit number"}
	}

	return nil
}

// usageStatsJSON represents the internal JSON structure for usage statistics.
// This matches the EPO OPS API response format.
type usageStatsJSON struct {
	// Data contains the usage entries
	Data []usageEntryJSON `json:"data"`
}

// usageEntryJSON represents a single usage entry in the JSON response.
type usageEntryJSON struct {
	// Timestamp in Unix time format (seconds since epoch)
	Timestamp int64 `json:"timestamp"`

	// TotalResponseSize in bytes
	TotalResponseSize int64 `json:"total_response_size"`

	// MessageCount is the number of API requests
	MessageCount int `json:"message_count"`

	// Service identifies which OPS service was used (optional)
	Service string `json:"service,omitempty"`
}

// parseUsageStats parses JSON usage statistics data into a UsageStats struct.
func parseUsageStats(jsonData string, timeRange string) (*UsageStats, error) {
	var rawStats usageStatsJSON

	if err := json.Unmarshal([]byte(jsonData), &rawStats); err != nil {
		return nil, fmt.Errorf("failed to parse usage statistics JSON: %w", err)
	}

	stats := &UsageStats{
		TimeRange: timeRange,
		Entries:   make([]UsageEntry, len(rawStats.Data)),
	}

	for i, entry := range rawStats.Data {
		stats.Entries[i] = UsageEntry(entry)
	}

	return stats, nil
}
