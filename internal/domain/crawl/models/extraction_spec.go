package models

type ValueType string

const (
	ValueString ValueType = "string"
	ValueInt    ValueType = "int"
	ValueFloat  ValueType = "float"
	ValueBool   ValueType = "bool"
	ValueURL    ValueType = "url"
	ValueJSON   ValueType = "json"
)

type TransformOp string

const (
	OpTrim         TransformOp = "trim"
	OpLower        TransformOp = "lower"
	OpUpper        TransformOp = "upper"
	OpNormalizeURL TransformOp = "normalize_url"
	OpUnique       TransformOp = "unique"
	OpLimit        TransformOp = "limit"
	OpParseInt     TransformOp = "to_int"
	OpParseFloat   TransformOp = "to_float"
	OpParsePrice   TransformOp = "parse_price"
	OpHTMLToText   TransformOp = "html_to_text"
	OpCollapseWS   TransformOp = "collapse_ws"
	OpSHA256       TransformOp = "sha256"
)

type MetricOp string

const (
	MetricLen                MetricOp = "len"
	MetricCount              MetricOp = "count"
	MetricWordCount          MetricOp = "word_count"
	MetricFieldPresent       MetricOp = "field_present"
	MetricStatusIsError      MetricOp = "status_is_error"
	MetricCountExternalLinks MetricOp = "count_external_links"
)

type ExtractionSpec struct {
	Fields     []FieldSpec
	Metrics    []MetricSpec
	Pagination []PaginationSpec
}

// PaginationSpec defines a pagination selector for extracting next-page URLs.
// It reuses the same semantics as ExtractorSpec (selector, attribute, multiple).
type PaginationSpec struct {
	Name      string `json:"name,omitempty"`      // Optional name for the pagination source (e.g., "next_page", "load_more")
	Selector  string `json:"selector,omitempty"`  // CSS selector for pagination elements (e.g., "a.next-page", ".pagination a")
	Attribute string `json:"attribute,omitempty"` // Attribute to extract URL from (default: "href")
	Multiple  bool   `json:"multiple,omitempty"`  // Extract all matching elements (true) or just first (false)
}

type FieldSpec struct {
	Name       string    // e.g. "title"
	Type       ValueType // "string" | "int" | "float" | "bool" | "url" | "json"
	Required   bool      // fail/mark bad quality if missing
	Extractor  ExtractorSpec
	Transforms []TransformSpec
}

type ExtractorSpec struct {
	Selector  string `json:"selector,omitempty"`  // CSS selector, e.g. "h1", "meta[name=description]"
	Attribute string `json:"attribute,omitempty"` // e.g. "href", "content"
	Multiple  bool   `json:"multiple,omitempty"`  // true => list result (before transforms)
	Index     *int   `json:"index,omitempty"`     // if Multiple=true choose one
}

type TransformSpec struct {
	Op  TransformOp // e.g. "trim", "normalize_url", "parse_price"
	Arg any         // free-form (string/number/object)
}

type MetricSpec struct {
	Name  string   `json:"name"`            // e.g. "title_len"
	Op    MetricOp `json:"op"`              // e.g. "len", "count", "status_is_error"
	Input string   `json:"input,omitempty"` // field name or special value like "body_text"
}
