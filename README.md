# EPO OPS Go Client

[![CI](https://github.com/patent-dev/epo-ops/actions/workflows/ci.yml/badge.svg)](https://github.com/patent-dev/epo-ops/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/patent-dev/epo-ops.svg)](https://pkg.go.dev/github.com/patent-dev/epo-ops)
[![Go Report Card](https://goreportcard.com/badge/github.com/patent-dev/epo-ops)](https://goreportcard.com/report/github.com/patent-dev/epo-ops)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

A Go client library for the European Patent Office's Open Patent Services (OPS) REST API v3.2, with OAuth2 authentication and typed responses.

## Overview

This library provides an idiomatic Go interface to the EPO's Open Patent Services:

- OAuth2 authentication with automatic token management and refresh on 401
- Patent text retrieval: bibliographic data, claims, description, abstract, and fulltext
- Patent search using CQL (Contextual Query Language), with optional constituents
- INPADOC family retrieval, including biblio and legal variants
- CPC/ECLA classification services (schema, statistics, mapping, media)
- Patent image retrieval with TIFF to PNG conversion
- Legal status retrieval with INPADOC legal event data
- EPO Register access (biblio, events, procedural steps, unitary patent)
- Patent number format conversion and validation
- Quota tracking against the fair use policy
- Typed errors plus automatic retry with exponential backoff

## Installation

```bash
go get github.com/patent-dev/epo-ops
```

## Getting access

The EPO OPS API requires a free developer account and an OAuth2 Consumer Key + Consumer Secret.

1. Register at the [EPO Developer Portal](https://developers.epo.org/user/register), choose the
   **Non-paying** access method, and submit the form. Wait for the confirmation email.

2. Sign in at the [portal](https://developers.epo.org/user/login) and open **My Apps**.

3. Click **Add a new App** to register an application and obtain its **Consumer Key** and
   **Consumer Secret**.

4. Export them for the client / demo:
   ```bash
   export EPO_OPS_CONSUMER_KEY=...
   export EPO_OPS_CONSUMER_SECRET=...
   ```


## Quick start

```go
package main

import (
    "context"
    "fmt"
    "log"

    ops "github.com/patent-dev/epo-ops"
)

func main() {
    client, err := ops.NewClient(&ops.Config{
        ConsumerKey:    "your-consumer-key",
        ConsumerSecret: "your-consumer-secret",
    })
    if err != nil {
        log.Fatal(err)
    }

    ctx := context.Background()

    biblio, err := client.GetBiblio(ctx, "publication", "docdb", "EP.1000000.B1")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Title: %s\n", biblio.Titles["en"])
}
```

## Usage

### Client creation

```go
// Defaults are filled in for any field left zero.
client, err := ops.NewClient(&ops.Config{
    ConsumerKey:    "your-key",
    ConsumerSecret: "your-secret",
})

// Custom configuration.
client, err := ops.NewClient(&ops.Config{
    ConsumerKey:    "your-key",
    ConsumerSecret: "your-secret",
    BaseURL:        "https://ops.epo.org/3.2/rest-services", // default
    MaxRetries:     3,                                       // default
    RetryDelay:     time.Second,                             // default
    Timeout:        30 * time.Second,                        // default
})
```

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `ConsumerKey` | string | *required* | OAuth2 consumer key |
| `ConsumerSecret` | string | *required* | OAuth2 consumer secret |
| `BaseURL` | string | `https://ops.epo.org/3.2/rest-services` | API base URL |
| `MaxRetries` | int | `3` | Maximum retry attempts |
| `RetryDelay` | time.Duration | `1s` | Base delay between retries |
| `Timeout` | time.Duration | `30s` | HTTP client timeout (increase for bulk classification endpoints) |

### Parsed vs raw API

Most methods return parsed Go structs for type-safe access. Each has a `*Raw()` variant that
returns the original XML for custom parsing or storage. Parsed methods internally call their
`*Raw()` counterpart and parse the result, so the two always agree.

```go
// Parsed -> *BiblioData
biblio, err := client.GetBiblio(ctx, "publication", "docdb", "EP.1000000.B1")
fmt.Printf("Title: %s\n", biblio.Titles["en"])

// Raw -> XML string
xmlData, err := client.GetBiblioRaw(ctx, "publication", "docdb", "EP.1000000.B1")
os.WriteFile("biblio.xml", []byte(xmlData), 0644)
```

Common parameters for published-data methods:

- `refType`: `"publication"`, `"application"`, or `"priority"`
- `format`: `"docdb"` or `"epodoc"`
- `number`: docdb format `"EP.1000000.B1"`, epodoc format `"EP1000000"`

### Published data retrieval

```go
// Bibliographic data -> *BiblioData
biblio, err := client.GetBiblio(ctx, "publication", "docdb", "EP.1000000.B1")
fmt.Printf("Publication Date: %s\n", biblio.PublicationDate)
for _, applicant := range biblio.Applicants {
    fmt.Printf("Applicant: %s (%s)\n", applicant.Name, applicant.Country)
}

// Claims -> *ClaimsData
claims, err := client.GetClaims(ctx, "publication", "docdb", "EP.1000000.B1")

// Description -> *DescriptionData
description, err := client.GetDescription(ctx, "publication", "docdb", "EP.1000000.B1")

// Abstract -> *AbstractData
abstract, err := client.GetAbstract(ctx, "publication", "docdb", "EP.1000000.B1")

// Full text (biblio + abstract + description + claims) -> *FulltextData
fulltext, err := client.GetFulltext(ctx, "publication", "docdb", "EP.1000000.B1")

// Published equivalents (simple family) -> *EquivalentsData
equivalents, err := client.GetPublishedEquivalents(ctx, "publication", "docdb", "EP.1000000.B1")
```

### Search

Returns `*SearchResultData` with parsed results.

```go
results, err := client.Search(ctx, "ti=battery", "1-25")
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Total results: %d\n", results.TotalCount)
for _, r := range results.Results {
    fmt.Printf("  %s%s%s - %s\n", r.Country, r.DocNumber, r.Kind, r.Title)
}

// Search with a specific constituent.
results, err = client.SearchWithConstituent(ctx, "biblio", "pa=Siemens", "1-10")

// Raw XML access.
xmlData, err := client.SearchRaw(ctx, "ti=battery", "1-25")
```

CQL examples: `ti=plastic` (title), `pa=Siemens` (applicant), `ic=H04W` (IPC class),
`pd>=2020` (publication date), `ti=plastic and pa=Siemens` (combined), and the
proximity form `ti=green prox/distance<=3 ti=energy`. Range format: `"1-25"`
(default), `"1-100"`, etc.

### Family retrieval

Returns `*FamilyData` with parsed family information.

```go
family, err := client.GetFamily(ctx, "publication", "docdb", "EP.1000000.B1")
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Family ID: %s, members: %d\n", family.FamilyID, family.TotalCount)
for _, member := range family.Members {
    fmt.Printf("Member: %s %s %s (Date: %s)\n",
        member.Country, member.DocNumber, member.Kind, member.Date)
}

// Variants enriching the family with biblio or legal data.
family, err = client.GetFamilyWithBiblio(ctx, "publication", "docdb", "EP.1000000.B1")
family, err = client.GetFamilyWithLegal(ctx, "publication", "docdb", "EP.1000000.B1")

// Raw XML access.
xmlData, err := client.GetFamilyRaw(ctx, "publication", "docdb", "EP.1000000.B1")
```

### Images and TIFF conversion

Patent images from EPO are typically TIFF. The `tiffutil` package converts them to PNG,
handling CCITT Group 3/4 and LZW compression, the CMYK color model, and automatic
landscape-to-portrait rotation.

```go
import (
    ops "github.com/patent-dev/epo-ops"
    "github.com/patent-dev/epo-ops/tiffutil"
)

// Retrieve a patent image (image types: "FullDocument", "Drawing", "FirstPageClipping").
imageData, err := client.GetImage(ctx, "EP", "1000000", "B1", "Drawing", 1)
if err != nil {
    log.Fatal(err)
}

// Convert TIFF -> PNG (with automatic rotation for landscape images).
pngData, err := tiffutil.TIFFToPNG(imageData)
os.WriteFile("patent_drawing.png", pngData, 0644)

// Without rotation, or batch convert multiple pages.
pngData, err = tiffutil.TIFFToPNGNoRotate(imageData)
pngImages, err := tiffutil.BatchTIFFToPNG([][]byte{imageData})
```

### Legal status and register

```go
// Legal status -> *LegalData
legal, err := client.GetLegal(ctx, "publication", "docdb", "EP.1000000.B1")
for _, event := range legal.LegalEvents {
    fmt.Printf("Event: %s (Code: %s, Country: %s)\n", event.Date, event.EventCode, event.Country)
}

// Register events -> *RegisterEventsData
regEvents, err := client.GetRegisterEvents(ctx, "publication", "epodoc", "EP1000000")

// EPO Register bibliographic data (raw XML).
registerBiblio, err := client.GetRegisterBiblioRaw(ctx, "publication", "epodoc", "EP1000000")
```

### Number conversion

```go
converted, err := client.ConvertPatentNumber(ctx, "publication", "docdb", "EP.1000000.B1", "epodoc")
```

Formats: `original` (`US.(05/948,554).19781004`), `epodoc` (`US19780948554`),
`docdb` (`US 19780948554`).

### Quota monitoring

The EPO OPS fair use policy limits non-paying users to 4 GB/week (paying users pay for more);
see the [fair use page](https://www.epo.org/en/service-support/ordering/fair-use). The client
tracks quota usage from API response headers.

```go
client.GetBiblio(ctx, "publication", "docdb", "EP.1000000.B1")

quota := client.GetLastQuota()
if quota != nil {
    fmt.Printf("Status: %s\n", quota.Status) // "green", "yellow", "red", or "black"
    fmt.Printf("Individual: %.2f%%\n", quota.Individual.UsagePercent())
}
```

### Retry logic

The client automatically retries failed requests with exponential backoff:

- **Retryable**: 5xx errors, 408, timeouts, network errors
- **Non-retryable**: 404, 400, authentication errors, quota exceeded
- **Token refresh**: automatic on 401 errors
- **Backoff**: exponential, base delay x (attempt + 1)

Tune it through `MaxRetries` and `RetryDelay` on the `Config`.

## Error handling

The library returns typed errors. Use `errors.As` to inspect them:

```go
biblio, err := client.GetBiblio(ctx, "publication", "docdb", "EP.1000000.B1")
if err != nil {
    var notFound *ops.NotFoundError
    var quota *ops.QuotaExceededError
    switch {
    case errors.As(err, &notFound):
        log.Printf("patent not found: %v", notFound)
    case errors.As(err, &quota):
        log.Printf("fair use quota exceeded: %v", quota)
    default:
        log.Printf("error: %v", err)
    }
}
```

Available error types:

- `AuthError` - authentication failures
- `NotFoundError` - resource not found (404)
- `QuotaExceededError` - fair use quota exceeded (429, 403)
- `ServiceUnavailableError` - temporary service outage (503)
- `AmbiguousPatentError` - multiple kind codes available
- `ValidationError` - invalid input (number, format, date)
- `ConfigError` - configuration issues

## Testing

```bash
make test              # unit tests (mock HTTP server, race)
make test-integration  # integration tests against the real API, needs credentials
make lint
```

Integration tests require credentials in the environment:

```bash
export EPO_OPS_CONSUMER_KEY="your-key"
export EPO_OPS_CONSUMER_SECRET="your-secret"
```

See the [demo/](demo/) directory for a complete example application exercising all features.

## API specification

The typed client in `generated/` is produced by [oapi-codegen](https://github.com/oapi-codegen/oapi-codegen)
from an OpenAPI 3.0 specification kept in [openapi.yaml](openapi.yaml). That spec was:

- Converted from the official EPO OPS Swagger 2.0 specification (`resources/ops.yaml`)
- Fixed up for OpenAPI 3.0 (OAuth2 flow set to `clientCredentials`, invalid parameter formats removed)
- Extended with the Data Usage Statistics endpoint (`/developers/me/stats/usage`), which the official
  spec omits

To regenerate after the spec changes, run `make generate`, which re-applies the conversion fixes (via
the scripts in [scripts/](scripts/)) and then runs the `//go:generate` directives in `client.go`.

## Related projects

Part of the [patent.dev](https://patent.dev) open-source patent data ecosystem:

- [epo-bdds](https://github.com/patent-dev/epo-bdds) - EPO Bulk Data Distribution Service client (DOCDB, INPADOC, EP full text)
- [uspto-odp](https://github.com/patent-dev/uspto-odp) - USPTO Open Data Portal client (patents, PTAB, TSDR, full text)
- [dpma-connect-plus](https://github.com/patent-dev/dpma-connect-plus) - DPMA Connect Plus client (patents, designs, trademarks)

The [bulk-file-loader](https://github.com/patent-dev/bulk-file-loader) uses these libraries for automated patent data downloads.

## License

MIT - Funktionslust GmbH / patent.dev.
