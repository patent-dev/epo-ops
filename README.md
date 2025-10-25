# EPO OPS Go Client

A Go client library for the European Patent Office's Open Patent Services (OPS) API v3.2.

## Overview

This library provides an idiomatic Go interface to interact with the EPO's Open Patent Services, allowing you to:

- Retrieve patent bibliographic data, claims, descriptions, and abstracts
- Search for patents using various criteria
- Get patent family information (INPADOC)
- Access CPC/ECLA classification data (schema, statistics, mapping, media)
- Download patent images and convert TIFF to PNG
- Access legal status and register data
- Track API quota usage

## Features

- OAuth2 authentication with automatic token management
- Patent text retrieval (biblio, claims, description, abstract, fulltext)
- Patent search using CQL (Contextual Query Language)
- INPADOC family retrieval
- CPC/ECLA classification services (schema, statistics, mapping, media)
- Patent image retrieval with TIFF to PNG conversion
- Legal status retrieval
- EPO Register access (biblio, events, procedural steps, unitary patent)
- Patent number format conversion
- Comprehensive error handling with custom error types
- Automatic retry logic with exponential backoff
- Quota tracking and fair use monitoring
- Unit and integration tests


## Installation

```bash
go get github.com/patent-dev/epo-ops
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"

    ops "github.com/patent-dev/epo-ops"
)

func main() {
    config := &ops.Config{
        ConsumerKey:    "your-consumer-key",
        ConsumerSecret: "your-consumer-secret",
    }

    client, err := ops.NewClient(config)
    if err != nil {
        log.Fatal(err)
    }

    ctx := context.Background()

    // Retrieve bibliographic data
    biblio, err := client.GetBiblio(ctx, "publication", "docdb", "EP1000000")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println("Biblio:", biblio)

    // Search patents
    results, err := client.Search(ctx, "ti=plastic", "1-5")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println("Search results:", results)

    // Get patent family
    family, err := client.GetFamily(ctx, "publication", "docdb", "EP1000000B1")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println("Family:", family)

    // Get patent image (first page of drawings)
    imageData, err := client.GetImage(ctx, "EP", "1000000", "B1", "Drawing", 1)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Retrieved image: %d bytes\n", len(imageData))
}
```

## Getting Credentials

To use the EPO OPS API, you need to register for API credentials:

1. Visit https://developers.epo.org/
2. Create an account or sign in
3. Register a new application to get your consumer key and secret

## Fair Use Policy & Quota Tracking

The EPO OPS API has usage limits:
- **Non-paying users**: 4 GB/week (free)
- **Paying users**: >4 GB/week (€2,800/year)

See: https://www.epo.org/en/service-support/ordering/fair-use

This client automatically tracks quota usage from API responses:

```go
// Make API calls
client.GetBiblio(ctx, "publication", "docdb", "EP1000000")

// Check quota status
quota := client.GetLastQuota()
if quota != nil {
    fmt.Printf("Status: %s\n", quota.Status) // "green", "yellow", "red", or "black"
    fmt.Printf("Individual: %d/%d (%.2f%%)\n",
        quota.Individual.Used,
        quota.Individual.Limit,
        quota.Individual.UsagePercent())
}
```

## Image Retrieval & TIFF Conversion

Patent images from EPO are typically in TIFF format. This library includes utilities to convert TIFF to PNG:

```go
import (
    ops "github.com/patent-dev/epo-ops"
    "github.com/patent-dev/epo-ops/tiffutil"
)

// Retrieve patent image (TIFF format)
imageData, err := client.GetImage(ctx, "EP", "1000000", "B1", "Drawing", 1)
if err != nil {
    log.Fatal(err)
}

// Convert TIFF to PNG (with automatic rotation for landscape images)
pngData, err := tiffutil.TIFFToPNG(imageData)
if err != nil {
    log.Fatal(err)
}

// Save PNG file
os.WriteFile("patent_drawing.png", pngData, 0644)

// Or convert without rotation
pngData, err := tiffutil.TIFFToPNGNoRotate(imageData)

// Or batch convert multiple pages
pngImages, err := tiffutil.BatchTIFFToPNG([][]byte{imageData1, imageData2, imageData3})
```

The TIFF utilities support:
- CCITT Group 3/4 compression (common in patent drawings)
- LZW compression
- CMYK color model
- Automatic landscape-to-portrait rotation

## API Reference

### Client Creation

```go
// Create client with default configuration
config := &ops.Config{
    ConsumerKey:    "your-key",
    ConsumerSecret: "your-secret",
}
client, err := ops.NewClient(config)

// Create client with custom configuration
config := &ops.Config{
    ConsumerKey:    "your-key",
    ConsumerSecret: "your-secret",
    BaseURL:        "https://ops.epo.org/3.2/rest-services",  // Default
    MaxRetries:     3,                                          // Default
    RetryDelay:     time.Second,                                // Default
    Timeout:        30 * time.Second,                           // Default
}
client, err := ops.NewClient(config)
```

### Published Data Retrieval

```go
// Retrieve bibliographic data
biblio, err := client.GetBiblio(ctx, "publication", "docdb", "EP1000000B1")

// Retrieve claims
claims, err := client.GetClaims(ctx, "publication", "docdb", "EP1000000B1")

// Retrieve description
description, err := client.GetDescription(ctx, "publication", "docdb", "EP1000000B1")

// Retrieve abstract
abstract, err := client.GetAbstract(ctx, "publication", "docdb", "EP1000000B1")

// Retrieve full text (biblio + abstract + description + claims)
fulltext, err := client.GetFulltext(ctx, "publication", "docdb", "EP1000000B1")
```

**Parameters**:
- `refType`: Reference type - `"publication"`, `"application"`, or `"priority"`
- `format`: Number format - `"docdb"` or `"epodoc"`
- `number`: Patent number (e.g., `"EP1000000B1"`)

### Search

```go
// Basic search
results, err := client.Search(ctx, "ti=plastic", "1-25")

// Search with specific constituent
results, err := client.SearchWithConstituent(ctx, "biblio", "pa=Siemens", "1-10")
```

**CQL Query Examples**:
- `ti=plastic` - Title contains "plastic"
- `pa=Siemens` - Applicant is Siemens
- `ti=plastic and pa=Siemens` - Combined search
- `de` - Country code DE

**Range Format**: `"1-25"` (default), `"1-100"`, etc.

### Family Retrieval

```go
// Basic INPADOC family
family, err := client.GetFamily(ctx, "publication", "docdb", "EP1000000B1")

// Family with bibliographic data
family, err := client.GetFamilyWithBiblio(ctx, "publication", "docdb", "EP1000000B1")

// Family with legal status
family, err := client.GetFamilyWithLegal(ctx, "publication", "docdb", "EP1000000B1")
```

### Images

```go
// Retrieve patent image (typically TIFF format)
imageData, err := client.GetImage(ctx, "EP", "1000000", "B1", "Drawing", 1)

// Image types: "FullDocument", "Drawing", "FirstPageClipping"
// Page: 1-based page number
```

### Legal & Register

```go
// Legal status data
legal, err := client.GetLegal(ctx, "publication", "docdb", "EP1000000B1")

// EPO Register bibliographic data
registerBiblio, err := client.GetRegisterBiblio(ctx, "publication", "docdb", "EP1000000B1")

// EPO Register procedural events
events, err := client.GetRegisterEvents(ctx, "publication", "docdb", "EP1000000B1")
```

### Number Conversion

```go
// Convert patent number formats
converted, err := client.ConvertPatentNumber(ctx, "publication", "docdb", "EP1000000B1", "epodoc")
```

**Formats**:
- `original`: `US.(05/948,554).19781004`
- `epodoc`: `US19780948554`
- `docdb`: `US 19780948554`

### Quota Monitoring

```go
// Get last quota information
quota := client.GetLastQuota()
if quota != nil {
    fmt.Printf("Status: %s\n", quota.Status)
    fmt.Printf("Usage: %.2f%%\n", quota.Individual.UsagePercent())
}
```

## Configuration Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `ConsumerKey` | string | *required* | OAuth2 consumer key |
| `ConsumerSecret` | string | *required* | OAuth2 consumer secret |
| `BaseURL` | string | `https://ops.epo.org/3.2/rest-services` | API base URL |
| `MaxRetries` | int | `3` | Maximum retry attempts |
| `RetryDelay` | time.Duration | `1s` | Base delay between retries |
| `Timeout` | time.Duration | `30s` | HTTP client timeout (increase for bulk classification endpoints) |

## Error Handling

The library provides custom error types for different failure scenarios:

```go
biblio, err := client.GetBiblio(ctx, "publication", "docdb", "EP1000000B1")
if err != nil {
    switch e := err.(type) {
    case *ops.AuthError:
        // Authentication failed
        log.Printf("Auth error: %v", e)
    case *ops.NotFoundError:
        // Patent not found (404)
        log.Printf("Patent not found: %v", e)
    case *ops.QuotaExceededError:
        // Fair use limit exceeded
        log.Printf("Quota exceeded: %v", e)
    case *ops.ServiceUnavailableError:
        // Temporary service outage
        log.Printf("Service unavailable: %v", e)
    default:
        // Other errors
        log.Printf("Error: %v", err)
    }
}
```

**Available Error Types**:
- `AuthError` - Authentication failures
- `NotFoundError` - Resource not found (404)
- `QuotaExceededError` - Fair use quota exceeded (429, 403)
- `ServiceUnavailableError` - Temporary service outage (503)
- `AmbiguousPatentError` - Multiple kind codes available
- `ConfigError` - Configuration issues

## Retry Logic

The client automatically retries failed requests with exponential backoff:

- **Retryable**: 5xx errors, 408, timeouts, network errors
- **Non-retryable**: 404, 400, authentication errors, quota exceeded
- **Token refresh**: Automatic on 401 errors
- **Backoff**: Exponential with base delay × (attempt + 1)

Example with custom retry configuration:

```go
config := &ops.Config{
    ConsumerKey:    "your-key",
    ConsumerSecret: "your-secret",
    MaxRetries:     5,                   // Try up to 5 times
    RetryDelay:     2 * time.Second,     // Start with 2s delay
}
client, err := ops.NewClient(config)
```

## Testing

Run unit tests:
```bash
go test -v
```

Run integration tests (requires credentials):
```bash
export EPO_OPS_CONSUMER_KEY="your-key"
export EPO_OPS_CONSUMER_SECRET="your-secret"
go test -tags=integration -v
```

## Demo Application

See the [demo/](demo/) directory for a complete example application demonstrating all features.

## API Specification

This library uses an OpenAPI 3.0 specification that was:
- Converted from the official EPO OPS Swagger 2.0 specification
- Extended to include the Data Usage Statistics endpoint (missing from official spec)
- Enhanced with proper type definitions for generated code

The specification is maintained in [openapi.yaml](openapi.yaml) and used to generate strongly-typed client code.

## Similar Projects

- [epo-bdds](https://github.com/patent-dev/epo-bdds) - EPO Bulk Data Download Service client
- [uspto-odp](https://github.com/patent-dev/uspto-odp) - USPTO Open Data Portal client

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Credits

**Developed by:**
- Wolfgang Stark - [patent.dev](https://patent.dev) - [Funktionslust GmbH](https://funktionslust.digital)