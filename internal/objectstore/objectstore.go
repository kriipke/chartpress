// internal/objectstore/objectstore.go
// Package objectstore wraps an S3-compatible bucket (AWS S3 / R2 / MinIO) behind
// an Uploader (operator) and a Presigner (backend), so callers test against fakes
// and never need a real bucket.
package objectstore

import (
	"os"
	"strings"
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
