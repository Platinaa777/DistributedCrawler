package env

import (
	"distributed-crawler/internal/config"
	"fmt"
	"os"
	"strconv"
)

const (
	redisAddressEnvName  = "REDIS_ADDRESS"
	redisPasswordEnvName = "REDIS_PWD"
	redisDBEnvName       = "REDIS_DB"
)

type redisConfig struct {
	address  string
	password string
	db       int
}

func NewRedisConfig() (config.RedisConfig, error) {
	address := os.Getenv(redisAddressEnvName)
	if address == "" {
		return nil, fmt.Errorf("%s environment variable is required", redisAddressEnvName)
	}

	password := os.Getenv(redisPasswordEnvName)
	if password == "" {
		return nil, fmt.Errorf("%s environment variable is required", redisPasswordEnvName)
	}

	// DB is optional, defaults to 0
	dbStr := os.Getenv(redisDBEnvName)
	db := 0
	if dbStr != "" {
		var err error
		db, err = strconv.Atoi(dbStr)
		if err != nil {
			return nil, fmt.Errorf("invalid %s value: %w", redisDBEnvName, err)
		}
	}

	return &redisConfig{
		address:  address,
		password: password,
		db:       db,
	}, nil
}

func (c *redisConfig) Address() string {
	return c.address
}

func (c *redisConfig) Password() string {
	return c.password
}

func (c *redisConfig) DB() int {
	return c.db
}
