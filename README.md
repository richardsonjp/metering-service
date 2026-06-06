# Metering Service

Built with [Fiber v2](https://github.com/gofiber/fiber).

---

## Contents

- [Quick start](#quick-start)
- [Configuration](#configuration)
- [API reference](#api-reference)
- [Architecture](#architecture)
- [Concurrency model](#concurrency-model)
- [Error handling](#error-handling)
- [Testing](#testing)
- [Assumptions & design decisions](#assumptions--design-decisions)
- [Known limitations & future improvements](#known-limitations--future-improvements)

---

## Quick start

Requirements: **Go 1.23+**. Nothing else.

```bash
make run            # or: go run ./cmd/apiserver server
```

The server listens on `:8080` by default. Try it:

```bash
curl -X POST localhost:8080/api/endpoint1
curl       localhost:8080/api/metrics

# upload a file and meter its size
head -c 1048576 /dev/urandom > sample.bin
curl -F file=@sample.bin localhost:8080/upload
curl localhost:8080/storage
```

Other useful targets:

```bash
make help           # print all Makefile commands
make test           # go test -race ./...
make cover          # tests + total coverage summary
make lint           # go vet ./...
make build          # build ./bin/metering-service
make config         # print the resolved configuration as JSON
```

---

## Configuration

Configuration is read from the environment (with a safe default per field) into a single
grouped struct in `config/config.go`, using struct tags via
[`caarlos0/env`](https://github.com/caarlos0/env). A `.env` file is loaded automatically if
present (see `.env.example`). A value that is *set but malformed* (e.g. `REQUEST_LIMIT=abc`)
makes the service fail fast at startup rather than run with a surprising default.

Inspect the resolved config any time with `make config` (or `go run ./cmd/apiserver config`).

| Variable              | Default            | Meaning                                                    |
|-----------------------|--------------------|------------------------------------------------------------|
| `SYSTEM_APP_NAME`     | `metering-service` | Application name (shown by `config`).                      |
| `SERVER_ADDR`         | `:8080`            | HTTP listen address.                                       |
| `SYSTEM_TIME_ZONE`    | `Asia/Jakarta`     | IANA timezone applied to log timestamps.                   |
| `REQUEST_LIMIT`       | `1000`             | Global cap on metered requests (`<=0` = unlimited).        |
| `STORAGE_LIMIT_BYTES` | `1073741824`       | Global storage cap in bytes — 1 GiB (`<=0` = unlimited).   |
| `MAX_UPLOAD_BYTES`    | `1073741824`       | Per-file upload cap in bytes — 1 GiB (`<=0` = unlimited).  |
| `LOG_LEVEL`           | `info`             | Log level: `debug` / `info` / `warn` / `error`.            |

> Dependencies are pure-Go libraries (`gofiber/fiber`, `caarlos0/env`, `joho/godotenv`,
> `sirupsen/logrus`, `mattn/go-isatty`, `gabriel-vasile/mimetype`, `stretchr/testify`) — there
> are still no external *services* to run.

### Logging

Logging uses a small `logrus` wrapper in `pkg/utils/logs` that writes to **stdout only** (no
remote/third-party sink). Output is a **pretty, timestamp-first console format** with a
color-coded level (INFO green, WARN yellow, ERROR red); colors auto-disable when stdout is not a
terminal (piped/redirected output stays clean). The level is set by `LOG_LEVEL`. On startup the
server also **prints the full route table** (Fiber's `EnablePrintRoutes`).

```text
2026-06-06 16:01:58.813  INFO   [Server:Addr]: :8080
2026-06-06 16:01:58.852  INFO   POST /api/endpoint1 -> 200 client_ip=127.0.0.1 latency=21µs response_body="{...}"
2026-06-06 16:01:58.868  WARN   POST /upload -> 415 client_ip=127.0.0.1 request_body="<multipart/form-data, 206 bytes>" response_body="{...}"
```

**Access log:** an `AccessLog` middleware logs one line per request with method, path, status,
latency, client IP, and the **request and response payloads**. The level follows the status
(≥500 error, ≥400 warn, else info). Bodies are summarized for safety — multipart uploads are
logged as `<multipart/form-data, N bytes>` (never the raw binary) and textual bodies are
truncated at 2 KB. Recovered panics are logged with their stack server-side while the HTTP `500`
response stays leak-free.

---

## API reference

All success responses use the envelope `{ "message"?, "data" }`. All errors use
`{ "code", "message", "status", "details"? }`.

> An importable **Postman collection** for all endpoints lives in
> [`postman/`](postman/) (collection + environment + usage notes).

### `POST /api/endpoint1` — track a request

Metered. Increments this endpoint's counter and the global request total.

```json
200 OK
{ "message": "request recorded",
  "data": { "endpoint": "/api/endpoint1", "count": 12, "total": 12, "remaining": 988 } }
```

`429 Too Many Requests` (`REQUEST_LIMIT_EXCEEDED`) once the global request limit is reached.

### `GET /api/metrics` — request counts for all endpoints

Not metered.

```json
200 OK
{ "data": {
    "endpoints": { "/api/endpoint1": 12, "/upload": 3 },
    "total_requests": 15,
    "limit": 1000,
    "remaining": 985 } }
```

### `POST /upload` — upload a file and track its size

Metered (counts toward the request total) **and** gated by the storage limit. Send
`multipart/form-data` with a `file` field. **Only image and video files are accepted** — the
type is detected from the file's magic bytes (via
[`gabriel-vasile/mimetype`](https://github.com/gabriel-vasile/mimetype)), so it can't be bypassed
by renaming a file or faking a `Content-Type` header. Coverage is broad: PNG/JPEG/GIF/WebP/BMP and
MP4/MOV (QuickTime)/M4V/MKV/WebM/AVI/MPEG/FLV, etc.

```json
201 Created
{ "message": "file recorded",
  "data": { "filename": "sample.png", "size": 1048576, "size_human": "1.0 MiB",
            "total_used_bytes": 1048576, "total_used_human": "1.0 MiB",
            "remaining_bytes": 1072693248 } }
```

| Status | Code                     | When                                                       |
|--------|--------------------------|------------------------------------------------------------|
| 400    | `FILE_REQUIRED`          | No `file` field, or the file is empty.                     |
| 413    | `FILE_TOO_LARGE`         | A single file exceeds `MAX_UPLOAD_BYTES`.                  |
| 415    | `UNSUPPORTED_FILE_TYPE`  | The file is not an image or a video (by magic-byte sniff). |
| 429    | `REQUEST_LIMIT_EXCEEDED` | Global request limit reached.                              |
| 507    | `STORAGE_LIMIT_EXCEEDED` | The upload would exceed the global storage limit.          |

### `GET /storage` — total storage used

Not metered.

```json
200 OK
{ "data": { "used_bytes": 1048576, "used_human": "1.0 MiB",
            "limit_bytes": 1073741824, "limit_human": "1.0 GiB",
            "remaining_bytes": 1072693248, "remaining_human": "1023.0 MiB" } }
```

### `GET /health`

Liveness probe → `200 { "message": "ok" }`.

---

## Architecture

A clean, layered design. The HTTP layer never touches state directly; all metering state
lives behind the in-memory **meter store** (the data/"repository" layer).

```
            HTTP request
                │
        ┌───────▼────────┐   Fiber middleware
        │   Recovery     │   panic → 500
        │   Metering     │   count + enforce request cap (POST routes only)
        └───────┬────────┘
                │
        ┌───────▼────────┐   handlers/  (parse, validate at boundary, respond)
        │    Handler     │   api_metering/ , storage/
        └───────┬────────┘
                │
        ┌───────▼────────┐   internal/services/metering  (business logic; no Fiber)
        │    Service     │
        └───────┬────────┘
                │
        ┌───────▼────────┐   internal/meter  (concurrency-safe in-memory store)
        │  Meter store   │   atomic request counters · channel-actor for storage
        └────────────────┘
```

```
cmd/apiserver/
  main.go                       entrypoint
  app/
    server.go                   bootstrap + graceful shutdown
    store/store.go              dependency wiring (no DB)
    routes/                     routes.go (app + middleware) + api_metering.go, storage.go
    handlers/api_metering/      POST /api/endpoint1, GET /api/metrics
    handlers/storage/           POST /upload, GET /storage
internal/
  meter/                        storage.go (atomic request counters + storage actor)
  services/metering/            business logic over the store
  middleware/                   metering.go, access_log.go, recovery.go
pkg/utils/
  errors/                       AppError + registry + Fiber responder
  api/                          success envelope
  bytesize/                     human-readable byte formatting
config/                         env-driven configuration
```

This layering and the `errors`/`api` envelopes follow the conventions captured in
[`.cursor/rules`](.cursor/rules) (adapted from an existing Fiber service skeleton, with the
database-specific rules removed since this service is in-memory).

---

## Concurrency model

The service is designed to absorb a high volume of concurrent requests with **no data
races** (the test suite runs under `-race`). The request hot path is lock-free atomics;
storage uses a single owner goroutine fed over a channel (an "actor"). No mutex is taken on
the per-request counter path.

**Request counters — lock-free atomics.** Per-endpoint and global request counts are
`atomic.Int64`. The endpoint map uses double-checked locking so each key is created at most
once; after warm-up every increment is `RLock` + `atomic.Add`, so independent endpoints
never contend.

**Exact request cap — atomic add-then-rollback.** The global cap is enforced with
`n := total.Add(1); if n > limit { total.Add(-1); reject }`. This admits **exactly** the
configured number of requests even under massive contention (proven by
`TestRecordRequest_ConcurrentCapIsExact`, which fires 5,000 goroutines at a limit of 1,000
and asserts exactly 1,000 are admitted).

**Storage reservation — a channel-actor (goroutines + channels).** Storage usage is owned by a
single background goroutine (the "actor"). A check-then-add would let two concurrent uploads
both pass the check and jointly exceed the limit; instead, `ReserveStorage` sends a `{size,
reply}` command over a channel and **blocks** for the answer, so the actor serialises every
reservation and the accept/`507` decision is exact and race-free
(`TestReserveStorage_ConcurrentNeverExceeds`). This is "use goroutines and channels where
appropriate": uploads are low-frequency, so serialising them through one goroutine costs
nothing, and a single owner is a clean fit for a check-then-commit decision.

**Graceful shutdown** also uses a goroutine + channel: the server runs in a goroutine while the
main goroutine blocks on an `os.Signal` channel and drains in-flight requests via
`ShutdownWithContext` on SIGINT/SIGTERM (then the storage actor is stopped via `Close`).

---

## Error handling

Errors are modelled as a structured `AppError` (`pkg/utils/errors`) carrying a stable code,
HTTP status, message, and optional details. `errors.Respond(c, err)` renders any error as a
consistent JSON envelope — including framework errors (e.g. Fiber's body-limit → 413) — so
clients always see the same shape:

```json
{ "code": "STORAGE_LIMIT_EXCEEDED", "status": 507,
  "message": "Storage limit reached; upload rejected",
  "details": ["uploading 60 B would exceed the 200 B storage limit (150 B already used)"] }
```

New codes are registered in `pkg/utils/errors/registry.go`.

---

## Testing

```bash
make test     # go test -race ./...
make cover    # adds a total coverage summary
```

- **Framework:** standard `go test` + [testify](https://github.com/stretchr/testify),
  table-driven where it helps.
- **Race detector:** the whole suite runs under `-race`.
- **Coverage:** ~86% of statements (`go test -race -coverpkg=./... ./...`). The core
  packages (`meter`, `services`, handlers, middleware) sit at 90–100%; the remainder is the
  `main`/`Run` server loop, which is intentionally not unit-tested.
- **What's covered:**
  - **Concurrency (the focus):** exact request-cap admission and storage reservation under
    thousands of goroutines, all race-clean.
  - **Service logic:** tracking, metrics, upload accounting, threshold errors.
  - **HTTP end-to-end** (`routes_test.go`): every endpoint, positive and negative, asserting
    status codes (200/201/400/413/429/507) and JSON shapes.
  - **Utilities:** byte formatting, the error responder (all branches), config loading.

### Stress testing (live API)

Beyond the unit tests, `scripts/stress_test.sh` load-tests the **running** server with
[vegeta](https://github.com/tsenart/vegeta) — no Go test harness, just HTTP traffic.

```bash
brew install vegeta            # or: go install github.com/tsenart/vegeta/v12@latest
make stress                    # or: ./scripts/stress_test.sh
```

It builds the binary, serves it locally, and runs two sections (tunable via
`RATE` / `DURATION` / `MAX_WORKERS` / `PORT`, or point at an existing server with `TARGET`):

- **Throughput** (uncapped limits): hammers every endpoint and reports throughput + latency
  percentiles. Useful to compare the lock-free atomic counter (`POST /api/endpoint1`) against
  the single-goroutine storage actor (`POST /upload`).
- **Cap enforcement under load**: blasts past the request and storage caps and shows the exact
  splits — e.g. `200:1000 / 429:5000` with `total_requests == 1000`, and `201:992 / 507:5008`
  with storage stopped precisely at the limit (the actor holds the cap, race-free, under load).

Besides the per-scenario terminal reports, it writes an **interactive HTML report** via
`vegeta plot` (latency-over-time, one series per scenario) to `stress-report.html` — override the
path with `REPORT=/path/to/report.html`. Open it in a browser.

---

## Assumptions & design decisions

- **Global thresholds.** `REQUEST_LIMIT` caps the *total* number of metered requests across
  all endpoints; `STORAGE_LIMIT_BYTES` caps *total* storage. Per-endpoint counts are still
  tracked and exposed via `/api/metrics`.
- **What is metered.** The two mutating endpoints (`POST /api/endpoint1`, `POST /upload`)
  are metered and count toward the global request cap. The read endpoints
  (`GET /api/metrics`, `GET /storage`) are deliberately **not** metered, so polling them
  never consumes the budget. A rejected upload still counts as a request (it was made), but
  reserves no storage.
- **In-memory & non-persistent.** All state lives in process memory and resets on restart —
  appropriate for this exercise and the simplest correct design for the concurrency focus.
- **Uploads are measured, not persisted.** `fileHeader.Size` is the actual size of the parsed
  multipart part, so metering is accurate. The request body is buffered in memory up to
  `BodyLimit` (fasthttp) before the handler runs, but the file is not additionally written to
  disk. Streaming uploads to disk/object storage is a documented improvement below.
- **Image/video only, by magic bytes.** The upload type is validated by sniffing the leading
  bytes with [`gabriel-vasile/mimetype`](https://github.com/gabriel-vasile/mimetype)
  (`pkg/utils/media`), accepting only `image/*` and `video/*`. This resists spoofed
  extensions/headers and covers the common formats — PNG/JPEG/GIF/WebP/BMP and
  MP4/MOV(QuickTime)/M4V/MKV/WebM/AVI/MPEG/FLV, etc. Unknown/binary content sniffs as
  `application/octet-stream` and is rejected with `415`. (Earlier this used Go's stdlib
  `http.DetectContentType`, which silently rejected QuickTime `.mov` because its `ftyp` brand is
  `"qt  "`, not `"mp4"`.)
- **Goroutine design decisions.** Using goroutine in here was optional as atomic covers 
  race conditions and concurrency problem, goroutine is use here to demonstrate on how the user works
  with goroutines and channel.
- **Status code choices.** `429` for the request cap, `507 Insufficient Storage` for the
  storage cap, `413` for an over-sized single file, `400` for a missing/empty file.

---

## Known limitations & future improvements

- **Single instance, in-memory state.** Counters and storage totals live in process memory and
  reset on restart — no persistence or cross-instance sharing.
- **Lifetime caps, no time window.** The 1k-request and 1 GiB limits are cumulative totals
  (until restart), not per-minute/hour/day. A production rate limiter would use a sliding window
  or token bucket with expiry/reset.
- **Storage reservation is serialized.** The storage actor is one goroutine, so extreme
  concurrent-upload throughput is ultimately bottlenecked there (fine at this scale — uploads
  are low-frequency). Using atomic would actually be better.
- **Uploads are metered, not stored.** Only the file size is recorded.
  Real persistence would stream uploads to disk/S3.
- **No authentication or per-tenant metering.** Metering is global; a real system would meter per
  API key / user with authn/authz and per-tenant quotas.
- **No metrics/trace export.** Operational logs only.