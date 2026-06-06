# Postman collection

An importable Postman collection (v2.1) covering every HTTP endpoint of the metering service,
with descriptions, example responses, and status-code test assertions.

| File | Purpose |
|------|---------|
| `metering-service.postman_collection.json` | The 5 endpoints, grouped into Health / API Metering / Storage Metering. |
| `metering-service.postman_environment.json` | A `baseUrl` variable (defaults to `http://localhost:8080`). |

## Endpoints covered

- `GET /health` — liveness probe
- `POST /api/endpoint1` — track a request (metered; `429` once the request cap is hit)
- `GET /api/metrics` — per-endpoint request counts + totals
- `POST /upload` — upload an image/video and track its size (`multipart/form-data`, field `file`)
- `GET /storage` — total storage used

## Use it (Postman app)

1. Start the service: `make run` (from the repo root) — listens on `:8080`.
2. In Postman: **Import** → select both JSON files.
3. Select the **“Metering Service — Local”** environment (or rely on the collection's
   `baseUrl` variable). Change `baseUrl` if you run on a different host/port.
4. Send any request. For **Upload file**, open the request's **Body → form-data**, click the
   `file` row's value, and **select an image or video file** (other types are rejected with
   `415`).

## Run it headless (newman)

```bash
# from the repo root, with the server running on :8080
npx --yes newman run postman/metering-service.postman_collection.json \
  -e postman/metering-service.postman_environment.json

# attach a real file to exercise a successful 201 upload:
npx --yes newman run postman/metering-service.postman_collection.json \
  -e postman/metering-service.postman_environment.json \
  --folder "Storage Metering" \
  --form-data "file=@/path/to/image.png"
```

> Without an attached file the **Upload** request returns `400 FILE_REQUIRED` — that's expected,
> and the test assertion accepts it. Select/attach an image or video to see a `201`.
