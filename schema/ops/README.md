# EPO OPS RWS schema (vendored)

The authoritative XML schema for the OPS RESTful Web Service v3.2 responses this client
parses. Vendored for reference and as the spec anchor for the parsers.

- Source: `https://ops.epo.org/3.2/schema/` (fetching requires a valid OAuth bearer token)
- Version: OPS 3.2
- Fetched: 2026-06-09

> Attribution: these schema files are published by the European Patent Office (EPO) as part
> of the Open Patent Services. They are included here unmodified, for reference only, and
> remain the property of the EPO under the EPO OPS terms of use - they are not covered by this
> project's MIT licence. (The embedded `Copyright` notices belong to the W3C/Sun MathML and
> ISO entity sets that the EPO schema imports.)

## Files

- `ops.xsd` - the `ops:world-patent-data` wrapper (search, family, register, images, ...)
- `exchange-documents.xsd` - bibliographic / family / published-data content (the big one)
- `fulltext-documents.xsd` - claims and description full text
- `ops_legal.xsd` - INPADOC legal-status events
- `rplus.xsd` - EP Register (register-plus)
- `ccd.xsd` - Common Citation Document
- `CPCSchema.xsd`, `CPCDefinitions.xsd` - CPC classification export

## Notes

- This is the **OPS REST** schema, not the DOCDB **bulk** exchange schema. They share the
  `http://www.epo.org/exchange` namespace but differ in serialization: OPS REST carries the
  number format in `document-id-type` on each `<document-id>` (docdb / epodoc / original),
  whereas the bulk product uses `data-format` on the reference element. This client targets
  the OPS REST shape.
- `ops.xsd` imports the CPC schemas via EPO-internal build paths
  (`../../../../wars/levelx/...`); the copies here were fetched directly from
  `/3.2/schema/`. `xlink.xsd` is referenced from `www.w3.org` and is not vendored.

## CCD (Common Citation Document)

`ops.xsd` declares an optional `ccd:ccd` element under `ops:world-patent-data` and imports
`ccd.xsd`, but no OPS REST endpoint serving it could be found (every probed path returns 404,
no demo example carries a `ccd` element). It is therefore schema-defined but not exposed by
the REST API for this account/version - there is no endpoint to wrap. Revisit if EPO exposes
a CCD constituent.
