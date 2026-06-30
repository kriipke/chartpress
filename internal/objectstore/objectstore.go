// internal/objectstore/objectstore.go
// Package objectstore wraps an S3-compatible bucket (AWS S3 / R2 / MinIO) behind
// an Uploader (operator) and a Presigner (backend), so callers test against fakes
// and never need a real bucket.
package objectstore

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// Config holds S3-compatible connection settings (BYO/external bucket).
type Config struct {
	Endpoint  string
	Bucket    string
	Region    string
	AccessKey string
	SecretKey string
	UseSSL    bool
}

// ConfigFromEnv reads the S3_* environment variables (set from the chart's S3
// Secret/values). UseSSL is true only when S3_USE_SSL == "true" (case-insensitive).
func ConfigFromEnv() Config {
	return Config{
		Endpoint:  os.Getenv("S3_ENDPOINT"),
		Bucket:    os.Getenv("S3_BUCKET"),
		Region:    os.Getenv("S3_REGION"),
		AccessKey: os.Getenv("S3_ACCESS_KEY"),
		SecretKey: os.Getenv("S3_SECRET_KEY"),
		UseSSL:    strings.EqualFold(os.Getenv("S3_USE_SSL"), "true"),
	}
}

// Uploader writes (and removes) chart archives. Used by the operator.
type Uploader interface {
	Upload(ctx context.Context, key string, r io.Reader, size int64) error
	Remove(ctx context.Context, key string) error
}

// Presigner mints time-limited GET URLs for chart archives. Used by the backend.
type Presigner interface {
	PresignGet(ctx context.Context, key string, expiry time.Duration) (string, error)
}

// Client is the minio-backed implementation of both Uploader and Presigner.
type Client struct {
	mc     *minio.Client
	bucket string
}

// New builds a minio client for the configured bucket. It does not connect;
// connection errors surface on the first Upload/Remove call (PresignGet is local).
func New(cfg Config) (*Client, error) {
	if cfg.Endpoint == "" {
		return nil, fmt.Errorf("objectstore: endpoint is required")
	}
	if cfg.Bucket == "" {
		return nil, fmt.Errorf("objectstore: bucket is required")
	}
	mc, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
		Region: cfg.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("objectstore: %w", err)
	}
	return &Client{mc: mc, bucket: cfg.Bucket}, nil
}

func (c *Client) Upload(ctx context.Context, key string, r io.Reader, size int64) error {
	_, err := c.mc.PutObject(ctx, c.bucket, key, r, size, minio.PutObjectOptions{ContentType: "application/zip"})
	return err
}

func (c *Client) Remove(ctx context.Context, key string) error {
	return c.mc.RemoveObject(ctx, c.bucket, key, minio.RemoveObjectOptions{})
}

func (c *Client) PresignGet(ctx context.Context, key string, expiry time.Duration) (string, error) {
	u, err := c.mc.PresignedGetObject(ctx, c.bucket, key, expiry, url.Values{})
	if err != nil {
		return "", err
	}
	return u.String(), nil
}
