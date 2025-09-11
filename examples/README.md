# Examples

## Minimal local run (SQLite)

```bash
# From repo root
export DATABASE_URL="sqlite:file:orch.sqlite?_fk=1&cache=shared&_pragma=busy_timeout(5000)"
go run ./cmd/orch --addr :8080
# New terminal
curl -s localhost:8080/healthz
curl -s -X POST localhost:8080/api/events \
  -H 'content-type: application/json' \
  -d '{"run_id":"demo","type":"started","payload":{"msg":"hi"}}'
curl -s localhost:8080/api/events?run=demo | jq
```

## Docker run (SQLite)

```bash
# Build image
docker build -t ghcr.io/${USER}/orch:dev .
# Run with a bind mount for the SQLite file if desired
docker run --rm -p 8080:8080 \
  -e DATABASE_URL="sqlite:file:/data/orch.sqlite?_fk=1&cache=shared&_pragma=busy_timeout(5000)" \
  -v $(pwd)/.data:/data \
  ghcr.io/${USER}/orch:dev
```

- Health: `GET /healthz`
- Events:
  - `POST /api/events` body: `{ "run_id": "demo", "type": "started", "payload": {...} }`
  - `GET /api/events?run=demo`
- Snapshots:
  - `POST /api/snapshots` body: `{ "run_id": "demo", "upto_seq": 2, "state": {...} }`
  - `GET /api/snapshots?run=demo`
