package cql

// validFields maps CQL field names to their descriptions.
// These are the official EPO OPS search fields as documented in the API specification.
//
// See: https://www.epo.org/searching-for-patents/data/web-services/ops.html
var validFields = map[string]string{
	// Text fields
	"ti":  "title",
	"ab":  "abstract",
	"ta":  "title or abstract",
	"txt": "full text",

	// Parties
	"pa": "applicant name",
	"in": "inventor name",
	"ia": "inventor or applicant",

	// Numbers
	"pn":  "publication number",
	"ap":  "application number",
	"pr":  "priority number",
	"num": "any number (pn, ap, or pr)",

	// Dates
	"pd":  "publication date",
	"ad":  "application date",
	"prd": "priority date",

	// Classifications
	"ic":   "IPC classification",
	"cpc":  "CPC classification",
	"ecla": "ECLA classification",

	// Citations
	"ct":  "cited patent",
	"rf":  "reference",
	"ctc": "cited category",

	// Designations
	"de": "designated states",
	"ep": "European states",
	"pc": "PCT contracting states",
}

// validOperators maps CQL operators to whether they're valid.
// EPO OPS supports standard boolean operators and proximity operators.
var validOperators = map[string]bool{
	// Boolean operators (case-insensitive)
	"AND": true,
	"OR":  true,
	"NOT": true,
	"and": true,
	"or":  true,
	"not": true,

	// Proximity operators
	"PROX": true,
	"prox": true,
	"ADJ":  true,
	"adj":  true,
	"NEAR": true,
	"near": true,

	// With prefix (for searching within fields)
	"WITH": true,
	"with": true,
}

// IsValidField checks if a field name is valid in EPO CQL.
func IsValidField(field string) bool {
	_, ok := validFields[field]
	return ok
}

// IsValidOperator checks if an operator is valid in EPO CQL.
func IsValidOperator(op string) bool {
	return validOperators[op]
}

// GetFieldDescription returns the description of a CQL field.
// Returns empty string if the field is not valid.
func GetFieldDescription(field string) string {
	return validFields[field]
}

// GetValidFields returns a slice of all valid field names.
func GetValidFields() []string {
	fields := make([]string, 0, len(validFields))
	for field := range validFields {
		fields = append(fields, field)
	}
	return fields
}
