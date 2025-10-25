# EPO OPS Go Client - Comprehensive Demo

This demo showcases all 46 endpoints of the EPO OPS v3.2 API.

## Requirements

### EPO OPS Credentials

You need EPO OPS API credentials (free registration):

1. Register at: https://developers.epo.org/
2. Create an app and get your Consumer Key and Secret
3. Set environment variables:

```bash
export EPO_OPS_CONSUMER_KEY="your-consumer-key"
export EPO_OPS_CONSUMER_SECRET="your-consumer-secret"
```

## Quick Start

```bash
# Build the demo
go build -o demo

# Run all endpoints with default patent EP.2400812.A1
./demo

# Run specific service  
./demo -service=classification

# Use different patent
./demo -patent=EP2533477A1

# Skip saving examples (faster)
./demo -no-save
```

## Command Line Options

```bash
-key string         EPO OPS consumer key (default: $EPO_OPS_CONSUMER_KEY)
-secret string      EPO OPS consumer secret (default: $EPO_OPS_CONSUMER_SECRET)
-patent string      Patent number in docdb format (default: "EP.2400812.A1")
                    Use dots for docdb: EP.2533477.B1
-service string     Run specific service: published, search, family, legal,
                    register, classification, number, usage, images
-examples string    Directory for saved examples (default: "examples")
-no-save           Skip saving request/response files
```
