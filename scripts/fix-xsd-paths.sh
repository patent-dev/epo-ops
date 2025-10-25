#!/bin/bash
set -e

# Fix XSD Schema Paths
#
# Problem: ops.xsd contains broken import paths from EPO's internal build system:
#   - ../../../../wars/levelx/target/webapp/schema/CPCSchema.xsd
#   - ../../../../wars/levelx/target/webapp/schema/CPCDefinitions.xsd
#
# Solution: Replace with local relative paths that match our resources/ directory
#
# Reference: These schemas are needed for:
#   - CPCSchema.xsd: CPC classification export data
#   - CPCDefinitions.xsd: CPC classification definitions
#   - Both are imported by ops.xsd and referenced in OPS API responses

XSD_FILE="resources/ops.xsd"

echo "Fixing XSD import paths in $XSD_FILE..."

if [ ! -f "$XSD_FILE" ]; then
    echo "Error: $XSD_FILE not found"
    exit 1
fi

# Create backup
cp "$XSD_FILE" "$XSD_FILE.backup"
echo "Created backup: $XSD_FILE.backup"

# Fix CPCSchema.xsd path
if [[ "$OSTYPE" == "darwin"* ]]; then
    # macOS sed
    sed -i '' 's|schemaLocation="../../../../wars/levelx/target/webapp/schema/CPCSchema\.xsd"|schemaLocation="CPCSchema.xsd"|g' "$XSD_FILE"
    sed -i '' 's|schemaLocation="../../../../wars/levelx/target/webapp/schema/CPCDefinitions\.xsd"|schemaLocation="CPCDefinitions.xsd"|g' "$XSD_FILE"
else
    # GNU sed
    sed -i 's|schemaLocation="../../../../wars/levelx/target/webapp/schema/CPCSchema\.xsd"|schemaLocation="CPCSchema.xsd"|g' "$XSD_FILE"
    sed -i 's|schemaLocation="../../../../wars/levelx/target/webapp/schema/CPCDefinitions\.xsd"|schemaLocation="CPCDefinitions.xsd"|g' "$XSD_FILE"
fi

echo ""
echo "Changes made:"
echo "  1. Fixed CPCSchema.xsd import path"
echo "  2. Fixed CPCDefinitions.xsd import path"
echo ""

# Verify changes
if grep -q "wars/levelx" "$XSD_FILE"; then
    echo "WARNING: Some broken paths still remain in $XSD_FILE"
    grep "wars/levelx" "$XSD_FILE"
    exit 1
else
    echo "✓ All paths fixed successfully!"
fi

# Show the fixed import statements
echo ""
echo "Fixed import statements:"
grep -A1 "cpcexport\|cpcdefinition" "$XSD_FILE" | grep schemaLocation

echo ""
echo "Backup saved as: $XSD_FILE.backup"
echo "To restore backup: mv $XSD_FILE.backup $XSD_FILE"
