# One private Spaces bucket for rendered chart archives (charts/<name>.zip),
# plus a scoped access key the app uses to reach it.
#
# One bucket is deliberate: chartpress's object store is single-bucket, and
# per-user isolation is enforced in the app (auth + ChartpressConfig ownership +
# the backend's presign check), never at the bucket. Versioning is off — the
# archive is a derived artifact the operator re-renders from the CR on every
# reconcile, so there is nothing to recover. No CORS rule — downloads are plain
# browser navigations to presigned URLs, which don't trigger CORS.

resource "digitalocean_spaces_bucket" "charts" {
  name   = "chartpress-charts-${var.env}"
  region = var.region
  acl    = "private"
}

# App-scoped key: readwrite on just this bucket. The operator uploads/removes
# archives; the backend presigns GETs (signing is local, but uses this key).
resource "digitalocean_spaces_key" "charts" {
  name = "chartpress-${var.env}"
  grant {
    bucket     = digitalocean_spaces_bucket.charts.name
    permission = "readwrite"
  }
}

output "bucket_name" {
  description = "Set as s3.bucket in the Helm values (values-do.yaml)."
  value       = digitalocean_spaces_bucket.charts.name
}

output "region" {
  description = "Set as s3.region in the Helm values."
  value       = var.region
}

output "endpoint" {
  description = "Set as s3.endpoint in the Helm values (host only, no scheme)."
  value       = "${var.region}.digitaloceanspaces.com"
}

output "access_key" {
  description = "S3 access key for the chartpress-s3 Secret. Read with: terraform output -raw access_key"
  value       = digitalocean_spaces_key.charts.access_key
  sensitive   = true
}

output "secret_key" {
  description = "S3 secret key for the chartpress-s3 Secret. Read with: terraform output -raw secret_key"
  value       = digitalocean_spaces_key.charts.secret_key
  sensitive   = true
}
