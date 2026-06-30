// internal/objectstore/objectstore_test.go
package objectstore

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestConfigFromEnv(t *testing.T) {
	t.Setenv("S3_ENDPOINT", "s3.example.com")
	t.Setenv("S3_BUCKET", "charts")
	t.Setenv("S3_REGION", "us-west-2")
	t.Setenv("S3_ACCESS_KEY", "AK")
	t.Setenv("S3_SECRET_KEY", "SK")
	t.Setenv("S3_USE_SSL", "true")

	c := ConfigFromEnv()
	if c.Endpoint != "s3.example.com" || c.Bucket != "charts" || c.Region != "us-west-2" ||
		c.AccessKey != "AK" || c.SecretKey != "SK" || !c.UseSSL {
		t.Fatalf("config = %+v", c)
	}
}

func TestConfigFromEnvUseSSLDefaultsFalse(t *testing.T) {
	t.Setenv("S3_USE_SSL", "")
	if ConfigFromEnv().UseSSL {
		t.Fatal("useSSL must be false when S3_USE_SSL is unset")
	}
}

// Compile-time proof the minio Client satisfies both consumer interfaces.
var (
	_ Uploader  = (*Client)(nil)
	_ Presigner = (*Client)(nil)
)

func TestNewRejectsMissingEndpointOrBucket(t *testing.T) {
	if _, err := New(Config{Bucket: "b"}); err == nil {
		t.Fatal("expected error when endpoint is empty")
	}
	if _, err := New(Config{Endpoint: "s3.example.com"}); err == nil {
		t.Fatal("expected error when bucket is empty")
	}
}

// PresignedGetObject computes the URL+signature locally (no network), so we can
// assert its shape against the real minio client without a bucket.
func TestPresignGetProducesSignedURL(t *testing.T) {
	c, err := New(Config{
		Endpoint: "s3.example.com", Bucket: "my-bucket", Region: "us-east-1",
		AccessKey: "AKIAEXAMPLE", SecretKey: "secretsecretsecret", UseSSL: true,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	u, err := c.PresignGet(context.Background(), "charts/demo.zip", 15*time.Minute)
	if err != nil {
		t.Fatalf("PresignGet: %v", err)
	}
	for _, want := range []string{
		"https://s3.example.com/my-bucket/charts/demo.zip",
		"X-Amz-Signature=",
		"X-Amz-Expires=900",
	} {
		if !strings.Contains(u, want) {
			t.Fatalf("presigned url %q missing %q", u, want)
		}
	}
}
