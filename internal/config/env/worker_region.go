package env

import "os"

const defaultWorkerRegion = "default"

// WorkerRegionConfig holds the region tag for this worker instance.
type WorkerRegionConfig struct {
	region string
}

// NewWorkerRegionConfig reads WORKER_REGION from environment.
func NewWorkerRegionConfig() *WorkerRegionConfig {
	region := os.Getenv("WORKER_REGION")
	if region == "" {
		region = defaultWorkerRegion
	}
	return &WorkerRegionConfig{region: region}
}

// Region returns the configured worker region.
func (c *WorkerRegionConfig) Region() string {
	return c.region
}
