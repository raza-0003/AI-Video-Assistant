package storage

import (
	"bytes"
	"context"
	"fmt"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3Store struct {
	client *s3.Client
	bucket string
}

// NewS3Store loads AWS credentials from the standard credential chain
// (env vars, shared config, or IAM role) and returns a ready-to-use client.
func NewS3Store(ctx context.Context, region, bucket string) (*S3Store, error) {
	cfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(region))
	if err != nil {
		return nil, err
	}
	return &S3Store{client: s3.NewFromConfig(cfg), bucket: bucket}, nil
}

// Upload writes content to key under the configured bucket and returns the S3 key.
func (s *S3Store) Upload(ctx context.Context, key string, content []byte) (string, error) {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: &s.bucket,
		Key:    &key,
		Body:   bytes.NewReader(content),
	})
	if err != nil {
		return "", fmt.Errorf("s3 upload failed: %w", err)
	}
	return key, nil
}

// Download fetches the object at key from the configured bucket.
func (s *S3Store) Download(ctx context.Context, key string) ([]byte, error) {
	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: &s.bucket,
		Key:    &key,
	})
	if err != nil {
		return nil, fmt.Errorf("s3 download failed: %w", err)
	}
	defer out.Body.Close()

	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(out.Body); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
