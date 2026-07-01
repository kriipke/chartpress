# infra/terraform — DigitalOcean cloud resources for chartpress.
#
# Provisions the single Spaces bucket chartpress stores rendered chart archives
# in, plus a scoped Spaces access key the app uses to upload (operator) and
# presign GETs (backend). See README.md for the one-time bootstrap and how the
# outputs get wired into the Helm release.

terraform {
  required_version = ">= 1.9.0"
  required_providers {
    digitalocean = {
      source  = "digitalocean/digitalocean"
      version = "~> 2.43"
    }
  }
  # State is local + gitignored (see .gitignore). It holds the app key's
  # secret_key, so treat the state file as a secret. Migrate to a remote
  # (Spaces) backend later if this becomes a multi-person deploy.
}

provider "digitalocean" {
  token = var.do_token
  # Spaces *bucket* operations use S3-style credentials, not the API token
  # above. The provider reads them from SPACES_ACCESS_KEY_ID /
  # SPACES_SECRET_ACCESS_KEY in the environment (see README.md → Bootstrap).
}

variable "do_token" {
  description = "DigitalOcean API token. Loaded from TF_VAR_do_token / DIGITALOCEAN_ACCESS_TOKEN."
  type        = string
  sensitive   = true
}

variable "env" {
  description = <<-EOT
    Deployment environment, suffixed onto the bucket + key names
    (chartpress-charts-$${env}). Defaults to "prod"; override when you add
    preview/staging pipelines. Must be a DNS-label-safe token because it lands
    in a globally-unique Spaces bucket name.
  EOT
  type        = string
  default     = "prod"
  validation {
    condition     = can(regex("^[a-z0-9]([a-z0-9-]*[a-z0-9])?$", var.env))
    error_message = "env must be lowercase alphanumeric with internal hyphens (e.g. prod, staging, pr-42)."
  }
}

variable "region" {
  description = "DigitalOcean region slug for the Spaces bucket."
  type        = string
  default     = "nyc3"
}
