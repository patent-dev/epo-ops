# Scripts Directory

Utility scripts for EPO OPS client library development.

## Available Scripts

### `fix-xsd-paths.sh`

Fixes broken XSD import paths in `resources/ops.xsd`.

**Problem**: The original `ops.xsd` file from EPO contains broken import paths from their internal build system:
```xml
schemaLocation="../../../../wars/levelx/target/webapp/schema/CPCSchema.xsd"
schemaLocation="../../../../wars/levelx/target/webapp/schema/CPCDefinitions.xsd"
```

**Solution**: Replaces them with local relative paths:
```xml
schemaLocation="CPCSchema.xsd"
schemaLocation="CPCDefinitions.xsd"
```

**Usage**:
```bash
./scripts/fix-xsd-paths.sh
```

Creates backup at `resources/ops.xsd.backup` before making changes.

**When to use**:
- After downloading updated XSD schemas from EPO
- When importing new versions of ops.xsd
- To fix schema validation errors

---

### `convert-openapi.sh`

Converts EPO OPS Swagger 2.0 specification to OpenAPI 3.0 format.

**What it does**:
1. Converts `resources/ops.yaml` (Swagger 2.0) to `openapi.yaml` (OpenAPI 3.0)
2. Fixes OAuth2 flow: `authorizationCode` → `clientCredentials`
3. Removes invalid `format: string` from boolean parameters
4. **Adds missing usage stats endpoint** from official OPS v3.2 documentation (EPO forgot to include it in swagger)

**Usage**:
```bash
./scripts/convert-openapi.sh
```

Requires `npx` (comes with Node.js).

**When to use**:
- After EPO releases updated API specification
- When regenerating code from OpenAPI spec
- One-time setup (already executed)

---

## Development Notes

### XSD Schema Dependencies

All XSD files in `resources/` are needed and referenced by `ops.xsd`:

| File | Purpose | Size |
|------|---------|------|
| `ops.xsd` | Main OPS schema (master) | 30 KB |
| `exchange-documents.xsd` | Bibliographic data | 444 KB |
| `fulltext-documents.xsd` | Claims, description, abstract | 34 KB |
| `ops_legal.xsd` | Legal status data | 13 KB |
| `CPCSchema.xsd` | CPC classification export | 15 KB |
| `CPCDefinitions.xsd` | CPC definitions | 12 KB |
| `ccd.xsd` | Common Citation Document | 12 KB |
| `rplus.xsd` | Register Plus service | 118 KB |

**Total**: ~680 KB (embedded in compiled library via `//go:embed`)

### Why Keep All XSD Files?

Even though only 5 XSDs are currently embedded in `xml.go`, all 8 files form a **complete schema chain**:

```
ops.xsd (master)
  ├── imports: exchange-documents.xsd
  ├── imports: fulltext-documents.xsd
  ├── imports: CPCSchema.xsd
  ├── imports: CPCDefinitions.xsd
  ├── imports: rplus.xsd
  ├── imports: ccd.xsd
  └── includes: ops_legal.xsd
```

Users who need full XML validation require all schemas in the chain.

### Documentation Files (Optional)

Large documentation files are kept in `resources/` for reference but should be linked online in README:

- `14.11_user_documentation_3.1_en.pdf` (65 KB) - Legacy docs
- `en-ops-v3.2-documentation-version-1.3.20.pdf` (4.7 MB) - Current API docs
- `CPC_International_documentation_and_sample_package.zip` (17 KB) - Sample data

These are **not embedded** in the library and can be deleted if you prefer to reference online documentation.
