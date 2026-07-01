# infra/terraform

DigitalOcean Spaces resources for chartpress: one private bucket for rendered
chart archives, plus a scoped Spaces access key the app uses to reach it.

Modeled on the matchstaq-v1 `infra/terraform` layout (provider + vars in
`main.tf`, Spaces resources + outputs in `spaces.tf`), trimmed to chartpress's
single-bucket object store.

## What this creates

| Resource | Name | Notes |
|----------|------|-------|
| `digitalocean_spaces_bucket.charts` | `chartpress-charts-${env}` | Private ACL, no versioning, no CORS |
| `digitalocean_spaces_key.charts` | `chartpress-${env}` | `readwrite`, scoped to the bucket above |

`env` defaults to `prod`, so a default apply yields `chartpress-charts-prod`.

## Bootstrap (one-time)

Creating a Spaces *bucket* requires S3-style Spaces credentials for the provider
— the DO API token alone can't do it. So before the first apply, export both the
API token and a Spaces key:

```bash
export DIGITALOCEAN_ACCESS_TOKEN=dop_v1_...        # API token
export TF_VAR_do_token="$DIGITALOCEAN_ACCESS_TOKEN"  # -> var.do_token
export SPACES_ACCESS_KEY_ID=...                    # a fullaccess Spaces key
export SPACES_SECRET_ACCESS_KEY=...
```

Get a Spaces key from the DO console (**API → Spaces Keys → Generate New Key**),
or mint a temporary one with `doctl`:

```bash
doctl spaces keys create tf-bootstrap --grants 'bucket=;permission=fullaccess'
# ...and delete it when done:
doctl spaces keys delete <access_key>
```

This bootstrap key is only for terraform to create the bucket. The app never
uses it — the app gets the scoped `digitalocean_spaces_key` this config mints.

## Apply

```bash
terraform init
terraform apply                 # env defaults to prod
# or: terraform apply -var env=staging
```

State is **local and gitignored** (`*.tfstate`). It contains the app key's
`secret_key`, so treat the file as a secret. Migrate to a remote Spaces backend
later if this becomes a multi-person deploy.

## Wire into the cluster

`make wire-do` (from the repo root) reads the outputs and creates/updates the
`chartpress-s3` Secret in your release namespace:

```bash
make wire-do NAMESPACE=chartpress
```

Then deploy with the DO override values, which point `s3.existingSecret` at that
Secret and carry the DO endpoint/region/bucket:

```bash
helm upgrade --install chartpress ./chart \
  -n chartpress -f chart/values-do.yaml \
  --set backend.openai.apiKeySecret.name=chartpress-openai
```

To read a credential by hand: `terraform output -raw access_key`.

## Not managed here

- **The DOKS cluster.** chartpress deploys onto an existing cluster; unlike
  matchstaq's `k8s.tf`, this config is Spaces-only.
- **Per-user multi-tenancy.** Isolation lives in the app (auth +
  `ChartpressConfig` ownership + the backend presign check), not in the bucket.
