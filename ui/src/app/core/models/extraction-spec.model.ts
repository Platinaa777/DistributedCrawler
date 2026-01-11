export interface ExtractionSpec {
  fields: FieldSpec[];
  metrics: MetricSpec[];
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

export interface MetricSpec {
  name: string;
  op: 'len' | 'count' | 'word_count' | 'field_present' | 'status_is_error' | 'count_external_links';
  input: string;
  arg?: string;
}

export interface TransformSpec {
  op: 'trim' | 'lower' | 'upper' | 'normalize_url' | 'unique' | 'limit' | 'to_int' | 'to_float' | 'parse_price' | 'html_to_text' | 'collapse_ws' | 'sha256';
  arg?: string;
}
