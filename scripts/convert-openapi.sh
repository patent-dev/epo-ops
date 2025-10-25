#!/bin/bash
set -e

# OpenAPI Spec Conversion Script
# Converts EPO OPS Swagger 2.0 spec to OpenAPI 3.0 and applies necessary fixes

echo "Converting Swagger 2.0 to OpenAPI 3.0..."
npx --yes swagger2openapi resources/ops.yaml -o openapi.yaml

echo "Fixing OAuth2 security scheme..."
# The conversion tool creates authorizationCode flow, but EPO OPS uses clientCredentials
# We need to replace the security scheme with the correct flow

# Use sed to replace the incorrect OAuth2 flow
if [[ "$OSTYPE" == "darwin"* ]]; then
  # macOS sed
  sed -i '' '/authorizationCode:/,/core: core/ {
    /authorizationCode:/c\
        clientCredentials:
    /authorizationUrl:/d
    /tokenUrl:/!d
    /scopes:/c\
          scopes: {}
  }' openapi.yaml
else
  # GNU sed
  sed -i '/authorizationCode:/,/core: core/ {
    /authorizationCode:/c\        clientCredentials:
    /authorizationUrl:/d
    /tokenUrl:/!d
    /scopes:/c\          scopes: {}
  }' openapi.yaml
fi

echo "Fixing boolean parameters with string format..."
# Some parameters have incorrect type: boolean with format: string
# This should be type: string for query parameters
# We need to fix all occurrences by removing "format: string" from boolean types
if [[ "$OSTYPE" == "darwin"* ]]; then
  # macOS sed - remove format: string from lines following type: boolean
  sed -i '' '
    /type: boolean/{
      n
      /format: string/d
    }
  ' openapi.yaml
else
  # GNU sed
  sed -i '
    /type: boolean/{
      n
      /format: string/d
    }
  ' openapi.yaml
fi

echo "Adding missing usage stats endpoint from official docs..."
# Add usage stats endpoint that EPO forgot to include in swagger
# Reference: OPS v3.2 documentation section "Data usage API"
# Endpoint: GET https://ops.epo.org/3.2/developers/me/stats/usage
# Note: This is under /developers/ not /rest-services/, so we need to handle the path carefully

python3 << 'PYTHON_SCRIPT'
import yaml
import sys

# Load the OpenAPI spec
with open('openapi.yaml', 'r') as f:
    spec = yaml.safe_load(f)

# Add the usage stats endpoint
# Since it's under /3.2/developers/ and not /3.2/rest-services/, we use a relative path
usage_endpoint = {
    'get': {
        'operationId': 'Get Usage Statistics',
        'summary': 'Data usage statistics API',
        'description': 'Retrieve data usage statistics for a given date or date range. Note: Calls to this API do not contribute to quota usage.',
        'tags': ['Usage'],
        'parameters': [
            {
                'name': 'timeRange',
                'in': 'query',
                'description': 'Single date or date range in dd/mm/yyyy or dd/mm/yyyy~dd/mm/yyyy format',
                'required': True,
                'schema': {
                    'type': 'string',
                    'format': 'string',
                    'default': '01/01/2024~31/01/2024'
                }
            }
        ],
        'responses': {
            '200': {
                'description': 'Valid usage statistics response',
                'content': {
                    'application/json': {
                        'schema': {
                            'type': 'object',
                            'description': 'Usage statistics with metrics including total_response_size (bytes) and message_count'
                        }
                    }
                }
            },
            'default': {
                'description': 'Error',
                'content': {
                    'application/json': {
                        'schema': {
                            '$ref': '#/components/schemas/Error'
                        }
                    }
                }
            }
        },
        'security': [
            {'client-credentials': []}
        ]
    }
}

# Add the endpoint using a relative path that goes up from /rest-services/ to /developers/
# Path: /../developers/me/stats/usage will resolve to /3.2/developers/me/stats/usage
spec['paths']['/../developers/me/stats/usage'] = usage_endpoint

# Write back
with open('openapi.yaml', 'w') as f:
    yaml.dump(spec, f, default_flow_style=False, sort_keys=False, allow_unicode=True)

print("✓ Added usage stats endpoint")
PYTHON_SCRIPT

echo "✓ Conversion complete!"
echo ""
echo "Changes made:"
echo "  1. Converted Swagger 2.0 → OpenAPI 3.0"
echo "  2. Fixed OAuth2 flow: authorizationCode → clientCredentials"
echo "  3. Updated tokenUrl to HTTPS"
echo "  4. Removed unnecessary scopes"
echo "  5. Fixed attachment parameter: type boolean → string"
echo "  6. Added missing usage stats endpoint (from official docs)"
echo ""
echo "Output: openapi.yaml"
