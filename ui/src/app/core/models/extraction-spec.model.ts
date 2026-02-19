export interface ExtractionSpec {
  fields: FieldSpec[];
  pagination?: PaginationSpec[];
  items?: ItemsSpec;
}

export interface ItemsSpec {
  container_selector: string;
  fields: FieldSpec[];
}

export interface PaginationSpec {
  name?: string;      // Optional name for the pagination source (e.g., "next_page", "load_more")
  selector: string;   // CSS selector for pagination elements (e.g., "a.next-page", ".pagination a")
  attribute?: string; // Attribute to extract URL from (default: "href")
  multiple?: boolean; // Extract all matching elements (true) or just first (false)
}

export interface FieldSpec {
  name: string;
  type: 'string' | 'int' | 'float' | 'bool' | 'url' | 'json';
  required: boolean;
  extractor: ExtractorSpec;
  transforms: TransformSpec[];
  label?: string;
}

export interface ExtractorSpec {
  selector: string;
  attribute: string;
  multiple: boolean;
  index?: number;
  // Legacy fields kept optional for backward compatibility with existing forms.
  source?: string;
  selector_type?: string;
  default_value?: string;
}

export interface TransformSpec {
  op: 'trim' | 'lower' | 'upper' | 'normalize_url' | 'unique' | 'limit' | 'to_int' | 'to_float' | 'parse_price' | 'html_to_text' | 'collapse_ws' | 'sha256';
  arg?: string;
}
