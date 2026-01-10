package contentstore

import (
	"bytes"
	"context"
	"distributed-crawler/internal/domain/crawl/services"
	"fmt"
	"io"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"go.uber.org/zap"
)

var _ services.ContentStore = (*MinIOStore)(nil)

// MinIOStore implements ContentStore using MinIO/S3
type MinIOStore struct {
	client     *minio.Client
	bucketName string
	logger     *zap.Logger
}

// NewMinIOStore creates a new MinIO-based content store
func NewMinIOStore(
	endpoint string,
	accessKeyID string,
	secretAccessKey string,
	useSSL bool,
	bucketName string,
	logger *zap.Logger,
) (*MinIOStore, error) {
	// Initialize MinIO client
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create MinIO client: %w", err)
	}

	store := &MinIOStore{
		client:     client,
		bucketName: bucketName,
		logger:     logger,
	}

	// Ensure bucket exists
	if err := store.ensureBucket(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ensure bucket exists: %w", err)
	}

	return store, nil
}

// ensureBucket creates the bucket if it doesn't exist
func (s *MinIOStore) ensureBucket(ctx context.Context) error {
	exists, err := s.client.BucketExists(ctx, s.bucketName)
	if err != nil {
		return fmt.Errorf("failed to check bucket existence: %w", err)
	}

	if !exists {
		err = s.client.MakeBucket(ctx, s.bucketName, minio.MakeBucketOptions{})
		if err != nil {
			return fmt.Errorf("failed to create bucket: %w", err)
		}
		s.logger.Info("Created MinIO bucket", zap.String("bucket", s.bucketName))
	}

	return nil
}

// Store saves content to MinIO
func (s *MinIOStore) Store(ctx context.Context, key string, content []byte, contentType string) error {
	reader := bytes.NewReader(content)
	_, err := s.client.PutObject(
		ctx,
		s.bucketName,
		key,
		reader,
		int64(len(content)),
		minio.PutObjectOptions{
			ContentType: contentType,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to store object %s: %w", key, err)
	}

	s.logger.Debug("Stored object to MinIO",
		zap.String("key", key),
		zap.Int("size", len(content)),
		zap.String("content_type", contentType),
	)

	return nil
}

// Get retrieves content from MinIO
func (s *MinIOStore) Get(ctx context.Context, key string) ([]byte, error) {
	obj, err := s.client.GetObject(ctx, s.bucketName, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get object %s: %w", key, err)
	}
	defer obj.Close()

	// Read all content
	content, err := io.ReadAll(obj)
	if err != nil {
		return nil, fmt.Errorf("failed to read object %s: %w", key, err)
	}

	s.logger.Debug("Retrieved object from MinIO",
		zap.String("key", key),
		zap.Int("size", len(content)),
	)

	return content, nil
}

// GetReader retrieves content as a reader
func (s *MinIOStore) GetReader(ctx context.Context, key string) (io.ReadCloser, error) {
	obj, err := s.client.GetObject(ctx, s.bucketName, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get object %s: %w", key, err)
	}

	// Validate object exists by checking stat
	_, err = obj.Stat()
	if err != nil {
		obj.Close()
		return nil, fmt.Errorf("failed to stat object %s: %w", key, err)
	}

	s.logger.Debug("Retrieved object reader from MinIO", zap.String("key", key))

	return obj, nil
}

// Delete removes content from MinIO
func (s *MinIOStore) Delete(ctx context.Context, key string) error {
	err := s.client.RemoveObject(ctx, s.bucketName, key, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete object %s: %w", key, err)
	}

	s.logger.Debug("Deleted object from MinIO", zap.String("key", key))

	return nil
}

// Exists checks if content exists in MinIO
func (s *MinIOStore) Exists(ctx context.Context, key string) (bool, error) {
	_, err := s.client.StatObject(ctx, s.bucketName, key, minio.StatObjectOptions{})
	if err != nil {
		// Check if error is "not found"
		errResponse := minio.ToErrorResponse(err)
		if errResponse.Code == "NoSuchKey" {
			return false, nil
		}
		return false, fmt.Errorf("failed to stat object %s: %w", key, err)
	}

	return true, nil
}

// PresignGetURL generates a presigned URL for downloading an object from MinIO
func (s *MinIOStore) PresignGetURL(key string, ttlMinutes int) (string, error) {
	// Generate presigned URL with specified TTL
	url, err := s.client.PresignedGetObject(
		context.Background(),
		s.bucketName,
		key,
		time.Duration(ttlMinutes)*time.Minute,
		nil,
	)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL for %s: %w", key, err)
	}

	s.logger.Debug("Generated presigned URL",
		zap.String("key", key),
		zap.Int("ttl_minutes", ttlMinutes),
	)

	return url.String(), nil
}
