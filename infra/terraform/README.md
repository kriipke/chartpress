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

`make wire-do` (from the repo root) reads the outputs, creates/updates the
`chartpress-s3` Secret in your release namespace, and writes
`chart/values-do.generated.yaml` (bucket/region/endpoint from the outputs):

```bash
make wire-do NAMESPACE=chartpress
```

Then deploy with the DO override values plus that generated overlay layered
last, so the release always targets the bucket that was actually provisioned —
even for `env=staging` or a non-`nyc3` region. `values-do.yaml` carries the prod
defaults for `s3.bucket/region/endpoint`; `values-do.generated.yaml` overrides
them with the current terraform env:

```bash
helm upgrade --install chartpress ./chart \
  -n chartpress -f chart/values-do.yaml -f chart/values-do.generated.yaml \
  --set backend.openai.apiKeySecret.name=chartpress-openai
```

To read a credential by hand: `terraform output -raw access_key`.

## Troubleshooting

**`Error creating bucket: Spaces credentials not configured`** (during `apply`)
The DO API token can't create a Spaces *bucket* — that needs S3-style Spaces
credentials. Export `SPACES_ACCESS_KEY_ID` / `SPACES_SECRET_ACCESS_KEY` (the
bootstrap key from [Bootstrap](#bootstrap-one-time)) and re-apply.

**`apply` prompts `var.do_token Enter a value:`**
`TF_VAR_do_token` isn't exported in this shell. Set it (or paste the token at
the prompt).

**`make wire-do` says `Output "access_key" not found`, or aborts with
"outputs missing/empty"**
`terraform apply` never completed, so `digitalocean_spaces_key.charts` — and its
`access_key`/`secret_key` outputs — don't exist. Confirm with:

```bash
terraform state list
# a successful apply shows BOTH:
#   digitalocean_spaces_bucket.charts
#   digitalocean_spaces_key.charts
```

The trap: a **failed** apply still records `bucket_name`, `region`, and
`endpoint` — they're derived from your config and `var.region`, not from a
created resource — so a non-empty `terraform output bucket_name` does *not*
prove the bucket exists. Only `access_key`/`secret_key`, which require the real
key resource, do. `make wire-do` guards on all five outputs and refuses to write
an empty `chartpress-s3` Secret, so if it aborts, finish `terraform apply`
first, then re-run it. (If an earlier jumped-ahead run already left an empty
Secret behind, delete it: `kubectl delete secret chartpress-s3 -n <ns>`.)

## Not managed here

- **The DOKS cluster.** chartpress deploys onto an existing cluster; unlike
  matchstaq's `k8s.tf`, this config is Spaces-only.
- **Per-user multi-tenancy.** Isolation lives in the app (auth +
  `ChartpressConfig` ownership + the backend presign check), not in the bucket.
