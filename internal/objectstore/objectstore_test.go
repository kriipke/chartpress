// internal/objectstore/objectstore_test.go
package objectstore

import "testing"

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
