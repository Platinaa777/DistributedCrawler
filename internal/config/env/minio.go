package env

import (
	"distributed-crawler/internal/config"
	"fmt"
	"os"
	"strconv"
)

const (
	minioEndpointEnvName        = "MINIO_ENDPOINT"
	minioAccessKeyIDEnvName     = "MINIO_USER"
	minioSecretAccessKeyEnvName = "MINIO_PWD"
	minioUseSSLEnvName          = "MINIO_USE_SSL"
	minioBucketNameEnvName      = "MINIO_BUCKET_NAME"
	minioPublicBaseURLEnvName   = "MINIO_PUBLIC_BASE_URL"
)

type minioConfig struct {
	endpoint        string
	accessKeyID     string
	secretAccessKey string
	useSSL          bool
	bucketName      string
	publicBaseURL   string
}

func NewMinIOConfig() (config.MinIOConfig, error) {
	endpoint := os.Getenv(minioEndpointEnvName)
	if endpoint == "" {
		return nil, fmt.Errorf("%s environment variable is required", minioEndpointEnvName)
	}

	accessKeyID := os.Getenv(minioAccessKeyIDEnvName)
	if accessKeyID == "" {
		return nil, fmt.Errorf("%s environment variable is required", minioAccessKeyIDEnvName)
	}

	secretAccessKey := os.Getenv(minioSecretAccessKeyEnvName)
	if secretAccessKey == "" {
		return nil, fmt.Errorf("%s environment variable is required", minioSecretAccessKeyEnvName)
	}

	// MINIO_USE_SSL is optional, defaults to false
	useSSL := false
	if useSSLStr := os.Getenv(minioUseSSLEnvName); useSSLStr != "" {
		parsed, err := strconv.ParseBool(useSSLStr)
		if err != nil {
			return nil, fmt.Errorf("%s must be a valid boolean (true/false)", minioUseSSLEnvName)
		}
		useSSL = parsed
	}

	bucketName := os.Getenv(minioBucketNameEnvName)
	if bucketName == "" {
		return nil, fmt.Errorf("%s environment variable is required", minioBucketNameEnvName)
	}

	return &minioConfig{
		endpoint:        endpoint,
		accessKeyID:     accessKeyID,
		secretAccessKey: secretAccessKey,
		useSSL:          useSSL,
		bucketName:      bucketName,
		publicBaseURL:   os.Getenv(minioPublicBaseURLEnvName),
	}, nil
}

func (c *minioConfig) Endpoint() string {
	return c.endpoint
}

func (c *minioConfig) AccessKeyID() string {
	return c.accessKeyID
}

func (c *minioConfig) SecretAccessKey() string {
	return c.secretAccessKey
}

func (c *minioConfig) UseSSL() bool {
	return c.useSSL
}

func (c *minioConfig) BucketName() string {
	return c.bucketName
}

func (c *minioConfig) PublicBaseURL() string {
	return c.publicBaseURL
}
