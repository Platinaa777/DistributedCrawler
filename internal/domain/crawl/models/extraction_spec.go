package models

type SourceType string

const (
	SourceHTML            SourceType = "html"
	SourceText            SourceType = "text"
	SourceResponseHeaders SourceType = "response_headers"
	SourceFetchMeta       SourceType = "fetch_meta"
)

type SelectorType string

const (
	SelectorCSS        SelectorType = "css"
	SelectorXPath      SelectorType = "xpath"
	SelectorRegex      SelectorType = "regex"
	SelectorJSONLD     SelectorType = "jsonld"
	SelectorMeta       SelectorType = "meta"
	SelectorHeader     SelectorType = "header"
	SelectorURL        SelectorType = "url"
	SelectorStatusCode SelectorType = "status_code"
)

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
	Fields  []FieldSpec
	Metrics []MetricSpec
}

type FieldSpec struct {
	Name       string    // e.g. "title"
	Type       ValueType // "string" | "int" | "float" | "bool" | "url" | "json"
	Required   bool      // fail/mark bad quality if missing
	Extractor  ExtractorSpec
	Transforms []TransformSpec

	// Optional: helps UI show nice labels, hints, grouping, etc.
	Label string // e.g. "Page title"
}

type ExtractorSpec struct {
	Source       SourceType   `json:"source"`              // "html" | "text" | "response_headers" | "fetch_meta"
	SelectorType SelectorType `json:"selector_type"`       // "css" | "xpath" | "regex" | "jsonld" | "meta" | "header" | "url" | "status_code"
	Selector     string       `json:"selector,omitempty"`  // e.g. "h1", "meta[name=description]"
	Attribute    string       `json:"attribute,omitempty"` // e.g. "href", "content"
	Multiple     bool         `json:"multiple,omitempty"`  // true => list result (before transforms)
	Index        *int         `json:"index,omitempty"`     // if Multiple=true choose one
	Default      any          `json:"default,omitempty"`   // default value if missing (string/number/bool)
}

type TransformSpec struct {
	Op  TransformOp // e.g. "trim", "normalize_url", "parse_price"
	Arg any         // free-form (string/number/object)
}

type MetricSpec struct {
	Name  string   `json:"name"`            // e.g. "title_len"
	Op    MetricOp `json:"op"`              // e.g. "len", "count", "status_is_error"
	Input string   `json:"input,omitempty"` // field name or special value like "body_text"
	Arg   any      `json:"arg,omitempty"`   // optional parameters
}
