package telemetry

import (
	"go.opentelemetry.io/otel/sdk/trace"
	otelTrace "go.opentelemetry.io/otel/trace"
)

// adaptiveSampler samples 100% of errors and a configurable rate for successful operations
type adaptiveSampler struct {
	successRate float64
	description string
}

// NewAdaptiveSampler creates a sampler that samples 100% of errors and successRate of successful spans
func NewAdaptiveSampler(successRate float64) trace.Sampler {
	if successRate < 0 {
		successRate = 0
	}
	if successRate > 1 {
		successRate = 1
	}
	return &adaptiveSampler{
		successRate: successRate,
		description: "AdaptiveSampler",
	}
}

// ShouldSample implements trace.Sampler
func (s *adaptiveSampler) ShouldSample(p trace.SamplingParameters) trace.SamplingResult {
	// Check if this span has error attributes - always sample errors
	for _, attr := range p.Attributes {
		if attr.Key == "error" || attr.Key == "error.type" || attr.Key == "otel.status_code" {
			if attr.Value.AsString() == "ERROR" || attr.Value.AsBool() {
				return trace.SamplingResult{
					Decision:   trace.RecordAndSample,
					Tracestate: otelTrace.SpanContextFromContext(p.ParentContext).TraceState(),
				}
			}
		}
	}

	// For parent-based sampling: if parent is sampled, sample this span too
	parentSpanContext := otelTrace.SpanContextFromContext(p.ParentContext)
	if parentSpanContext.IsSampled() {
		return trace.SamplingResult{
			Decision:   trace.RecordAndSample,
			Tracestate: parentSpanContext.TraceState(),
		}
	}

	// Use probability sampling for new traces
	// TraceID-based deterministic sampling for consistency
	if p.TraceID.IsValid() {
		// Use first 8 bytes of trace ID as sampling decision
		x := float64(p.TraceID[0]) / 255.0
		if x < s.successRate {
			return trace.SamplingResult{
				Decision:   trace.RecordAndSample,
				Tracestate: parentSpanContext.TraceState(),
			}
		}
	}

	return trace.SamplingResult{
		Decision:   trace.Drop,
		Tracestate: parentSpanContext.TraceState(),
	}
}

// Description implements trace.Sampler
func (s *adaptiveSampler) Description() string {
	return s.description
}
