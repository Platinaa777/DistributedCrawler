package interceptor

import (
	"context"
	"distributed-crawler/internal/infra/persistence"

	"google.golang.org/grpc"
)

// jobIDProvider is satisfied by proto messages that carry a job_id field.
type jobIDProvider interface {
	GetJobId() string
}

// jobIDFromIDProvider is satisfied by GetJobRequest where the ID itself is the job ID.
type jobIDFromIDProvider interface {
	GetId() string
}

// ShardKeyInterceptor returns a gRPC unary interceptor that extracts the
// crawl_job_id from the request and injects it as a shard key into the context.
// When sharding is disabled, the interceptor is a no-op passthrough.
func ShardKeyInterceptor(shardingEnabled bool) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		if !shardingEnabled {
			return handler(ctx, req)
		}

		var jobID string

		// Most request types that reference a job have GetJobId().
		if p, ok := req.(jobIDProvider); ok {
			jobID = p.GetJobId()
		}

		// GetJobRequest uses GetId() where the id IS the job id.
		// Only use this for job-specific endpoints.
		if jobID == "" {
			if p, ok := req.(jobIDFromIDProvider); ok {
				if isJobEndpoint(info.FullMethod) {
					jobID = p.GetId()
				}
			}
		}

		if jobID != "" {
			ctx = persistence.WithShardKey(ctx, jobID)
		}

		return handler(ctx, req)
	}
}

// isJobEndpoint checks if the gRPC method belongs to job-related RPCs
// where GetId() returns a job ID (not a task or preview ID).
func isJobEndpoint(method string) bool {
	jobMethods := map[string]bool{
		"/crawler.v1.CrawlerService/GetJob":    true,
		"/crawler.v1.CrawlerService/DeleteJob":  true,
	}
	return jobMethods[method]
}
