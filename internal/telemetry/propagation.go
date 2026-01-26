package telemetry

import (
	"context"

	"go.opentelemetry.io/otel"
)

// MapCarrier implements propagation.TextMapCarrier for map[string]string
type MapCarrier map[string]string

// Get returns the value associated with the passed key
func (c MapCarrier) Get(key string) string {
	return c[key]
}

// Set stores the key-value pair
func (c MapCarrier) Set(key, value string) {
	c[key] = value
}

// Keys returns the keys stored in this carrier
func (c MapCarrier) Keys() []string {
	keys := make([]string, 0, len(c))
	for k := range c {
		keys = append(keys, k)
	}
	return keys
}

// InjectTraceContext extracts trace context from the context and returns it as a map
// This is used when publishing messages to RabbitMQ to propagate trace context
func InjectTraceContext(ctx context.Context) map[string]string {
	carrier := make(MapCarrier)
	otel.GetTextMapPropagator().Inject(ctx, carrier)
	return carrier
}

// ExtractTraceContext creates a context with trace info extracted from the map
// This is used when consuming messages from RabbitMQ to restore trace context
func ExtractTraceContext(ctx context.Context, traceCtx map[string]string) context.Context {
	if traceCtx == nil || len(traceCtx) == 0 {
		return ctx
	}
	return otel.GetTextMapPropagator().Extract(ctx, MapCarrier(traceCtx))
}
