package env

import (
	"fmt"
	"os"
	"strconv"

	"distributed-crawler/internal/config"
)

const (
	memoryBrokerAddrEnvName     = "MEMORY_BROKER_ADDR"
	memoryBrokerCapacityEnvName = "MEMORY_BROKER_CAPACITY"

	memoryBrokerDefaultCapacity = 1000
)

type memoryBrokerConfig struct {
	address  string
	capacity int
}

// NewMemoryBrokerConfig reads MEMORY_BROKER_ADDR (required) and
// MEMORY_BROKER_CAPACITY (optional, default 1000) from the environment.
func NewMemoryBrokerConfig() (config.MemoryBrokerConfig, error) {
	addr := os.Getenv(memoryBrokerAddrEnvName)
	if addr == "" {
		return nil, fmt.Errorf("%s environment variable is required", memoryBrokerAddrEnvName)
	}

	capacity := memoryBrokerDefaultCapacity
	if capStr := os.Getenv(memoryBrokerCapacityEnvName); capStr != "" {
		n, err := strconv.Atoi(capStr)
		if err != nil || n <= 0 {
			return nil, fmt.Errorf("invalid %s value %q: must be a positive integer", memoryBrokerCapacityEnvName, capStr)
		}
		capacity = n
	}

	return &memoryBrokerConfig{address: addr, capacity: capacity}, nil
}

func (c *memoryBrokerConfig) Address() string    { return c.address }
func (c *memoryBrokerConfig) QueueCapacity() int { return c.capacity }
