# nx-remote-cache

Self-hosted remote cache server for [Nx](https://nx.dev), implementing the
HTTP contract from
[nx.dev/docs/kb/self-hosted-caching](https://nx.dev/docs/kb/self-hosted-caching#build-your-own-caching-server)
in Go.

## API

| Method | Path              | Auth              | Success | Failure                              |
|--------|-------------------|-------------------|---------|---------------------------------------|
| PUT    | `/v1/cache/{hash}` | Bearer, write scope | 200    | 401 no/bad token · 403 read-only token · 409 hash already stored · 411 missing Content-Length |
| GET    | `/v1/cache/{hash}` | Bearer, read scope  | 200 + binary body | 401 no/bad token · 403 forbidden · 404 not found |
| GET    | `/health`         | none              | 200     | —                                      |

Cache entries are content-addressed and immutable: a hash is written once
(`PUT`) and never overwritten — a second `PUT` for the same hash returns 409.

## Configuration

All configuration is via environment variables.

| Variable | Default | Notes |
|---|---|---|
| `PORT` | `3000` | listen port |
| `STORAGE_BACKEND` | `local` | `local` or `s3` |
| `CACHE_DIR` | `/var/lib/nx-remote-cache` | local backend only |
| `S3_BUCKET` / `S3_REGION` / `S3_PREFIX` | — | s3 backend |
| `S3_ENDPOINT` | — | set for R2/MinIO/non-AWS S3-compatible stores |
| `S3_USE_PATH_STYLE` | `false` | set `true` for MinIO |
| `CACHE_READ_TOKENS` | — | comma-separated bearer tokens, read-only |
| `CACHE_WRITE_TOKENS` | — | comma-separated bearer tokens, read+write (required, at least one) |
| `MAX_CACHE_ENTRY_BYTES` | `524288000` (500MB) | reject larger uploads |

Use `local` for a single always-on instance with a persistent volume. Use
`s3` when the server runs as multiple replicas or on ephemeral
infrastructure (recommended for anything backing CI at scale) — an S3
bucket (or R2/MinIO) is the shared durable store.

## Running locally

```bash
docker compose up --build
```

or without Docker:

```bash
make run
```

## Authentication strategy

Two token scopes exist because CI has two trust levels:

- **Write tokens** (`CACHE_WRITE_TOKENS`) can upload and download. Only give
  these to jobs you trust to publish good artifacts — typically pushes to
  `main`/protected branches, run with a token from a repository secret.
- **Read tokens** (`CACHE_READ_TOKENS`) can only download. Give these to
  untrusted contexts — e.g. builds triggered by pull requests from forks —
  so a malicious fork PR can benefit from the cache but can never poison it.
  A fork PR's workflow run does not have access to repository secrets by
  default, so simply omitting the write secret from `pull_request` (as
  opposed to `pull_request_target`) triggers already achieves this; the
  read token can be safely embedded in the workflow file if desired since
  it grants no write capability.

Rotate tokens by adding the new one to the CSV env var, updating repo
secrets, then removing the old one — no downtime, since multiple tokens per
scope are supported.

This server has no built-in TLS. Terminate TLS at a reverse proxy or load
balancer in front of it and only expose the plaintext port on a private
network — bearer tokens must never travel over plain HTTP on the public
internet.

## Nx client configuration

No `nx.json` changes are required. Point the Nx client at the server via
environment variables in your CI job (and locally, if desired):

```bash
export NX_SELF_HOSTED_REMOTE_CACHE_SERVER=https://nx-cache.example.com
export NX_SELF_HOSTED_REMOTE_CACHE_ACCESS_TOKEN=$CACHE_TOKEN
```

## CI/CD for this repo

- `.github/workflows/ci.yml` — on every push/PR: `go vet`, `go build`,
  `go test -race -cover`, `golangci-lint`, and a Docker build-only check.
- `.github/workflows/docker-publish.yml` — on push to `main` or a `vX.Y.Z`
  tag: builds a multi-arch (amd64/arm64) image and pushes it to
  `ghcr.io/<owner>/<repo>`.

Deploy by pulling the published image and running it with your storage/auth
env vars set (e.g. as a systemd unit, a Kubernetes Deployment, or an ECS
service) behind a TLS-terminating load balancer.

## Example: consumer repo workflow

A separate repo building with Nx would use the cache like this:

```yaml
# .github/workflows/ci.yml (in the Nx workspace repo)
jobs:
  build:
    runs-on: ubuntu-latest
    env:
      NX_SELF_HOSTED_REMOTE_CACHE_SERVER: https://nx-cache.example.com
      # Write token only available on trusted refs; fork PRs fall back to
      # the read-only token (or no token) and simply get cache misses on
      # write, cache hits on read.
      NX_SELF_HOSTED_REMOTE_CACHE_ACCESS_TOKEN: >-
        ${{ github.event_name == 'pull_request' && github.event.pull_request.head.repo.fork
            && secrets.NX_CACHE_READ_TOKEN
            || secrets.NX_CACHE_WRITE_TOKEN }}
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
      - run: npm ci
      - run: npx nx affected -t build test lint
```

Store `NX_CACHE_READ_TOKEN` and `NX_CACHE_WRITE_TOKEN` as repository (or
organization) secrets in the consumer repo, matching values in this
server's `CACHE_READ_TOKENS` / `CACHE_WRITE_TOKENS`.
