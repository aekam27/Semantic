# Semantic

[![CI](https://github.com/aekam27/Semantic/actions/workflows/ci.yml/badge.svg)](https://github.com/aekam27/Semantic/actions/workflows/ci.yml)

Semantic search over MongoDB Atlas vector indexes.

Two processes: a Go API that authenticates callers and runs `$vectorSearch`
queries, and a Python sidecar that turns query text into embeddings with
`sentence-transformers`.

```
client ──> Go API (:8001) ──> Python embedding service (:8000)
                │
                ├──> MongoDB Atlas  ($vectorSearch)
                └──> Redis          (issued token set)
```

## Requirements

- Go 1.25.12 or newer (the `go` directive pins this; earlier patch releases
  carry known standard library advisories)
- Python 3.10+
- MongoDB Atlas with a vector search index
- Redis

## Configuration

Everything is read from the environment. Copy [.env.example](.env.example) and
fill it in.

| Variable | Required | Default | Description |
| --- | --- | --- | --- |
| `JWT_SECRET` | yes | — | HMAC signing key, at least 32 bytes. |
| `MONGODB_URI` | yes | — | Full connection string including credentials. |
| `ADDR` | no | `:8001` | Listen address. |
| `MONGODB_DATABASE` | no | `semantic` | Database name. |
| `MONGODB_COLLECTION` | no | `vectormap` | Collection to search. |
| `REDIS_ADDR` | no | `localhost:6379` | Redis address. |
| `EMBEDDING_SERVICE_URL` | no | `http://localhost:8000/v1/vectorEmbeddings/` | Python sidecar endpoint. |
| `TOKEN_TTL` | no | `15m` | Token lifetime. Accepts `15m` or a bare number of seconds. |
| `REQUEST_TIMEOUT` | no | `30s` | Timeout for calls to the embedding service. |

The server refuses to start if `JWT_SECRET` or `MONGODB_URI` is missing, rather
than coming up in a state where every request fails.

## Running

Start the embedding sidecar:

```bash
cd python_server
pip install -r requirements.txt
uvicorn app:app --port 8000
```

Then the API:

```bash
go run .
```

## API

### `GET /healthz`

```json
{ "status": "ok" }
```

### `POST /v1/generateToken`

> **This endpoint is unauthenticated.** It hands a valid token to anyone who
> supplies an email address, which means the bearer token on `/v1/mget` is not
> currently an access control boundary. Put this service behind a gateway that
> authenticates callers, or replace this endpoint with a real credential check,
> before exposing it. See [Known gaps](#known-gaps).

```bash
curl -X POST localhost:8001/v1/generateToken \
  -H 'Content-Type: application/json' \
  -d '{"email":"user@example.com"}'
```

```json
{ "token": "eyJhbGciOiJIUzI1NiIs..." }
```

### `POST /v1/mget`

Requires `Authorization: Bearer <token>`. Returns gzipped JSON.

```bash
curl -X POST localhost:8001/v1/mget \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  --compressed \
  -d '{
    "searchIndex": "vector_index",
    "path": "embedding",
    "project": "title",
    "text": "how do I reset my password",
    "limit": 10
  }'
```

| Field | Required | Description |
| --- | --- | --- |
| `searchIndex` | yes | Atlas vector index name. |
| `path` | yes | Document field holding the embedding. |
| `text` | yes | Query text; embedded by the sidecar. |
| `project` | no | Single field to project. Omit to return whole documents. |
| `limit` | no | Results to return. Defaults to and is capped at 100. |

## Tokens

Tokens are HS256 JWTs carrying `email` and `exp`. Issued tokens are also written
to a Redis set that every instance polls every 10 seconds, which is what makes
revocation possible: clearing a token from the set (or marking it `hotlisted`)
takes it out of circulation without waiting for expiry. Issuing a token forces
an immediate refresh, so it works on the very next request.

Validation checks the signature, pins the algorithm to HS256, requires an
expiry, and then confirms the cached record agrees with the claims.

## Development

```bash
go test -race ./...
go vet ./...
gofmt -l .
go run golang.org/x/vuln/cmd/govulncheck@latest ./...
```

## Known gaps

- **`/v1/generateToken` has no authentication.** Anyone who can reach it can
  mint a token for any email. This predates the current work and fixing it
  needs a decision about what should authenticate a caller, so it is left as
  is — but it makes the bearer token decorative until it is addressed.
- **The token set is a single Redis key.** Two instances issuing tokens at the
  same instant can lose one of the writes. Tokens are re-issuable, so the cost
  is a client retry.
- **No rate limiting.** Both endpoints are uncapped.
- Package names use underscores (`auth_service`, `cache_service`), which is not
  idiomatic Go. Renaming them is churn that would bury more substantive changes,
  so it is deferred.

## Roadmap

- Elasticsearch-backed search alongside Atlas
- Rate limiting
