# Börstracker

Real-time stock price monitoring with configurable audio and visual alerts. Anonymous sessions, no login required. Runs entirely in Docker Compose.

## Quick start

```bash
cp .env.example .env
docker compose up -d
```

Open [http://localhost:8989](http://localhost:8989) in your browser (HTTP only, no TLS).

## Architecture

```mermaid
flowchart TB
  Browser[React SPA via Caddy]
  Caddy[Caddy reverse proxy]
  API[api Go service]
  PF[price-fetcher Go service]
  PG[(PostgreSQL + TimescaleDB)]
  RD[(Redis)]
  YF[Yahoo Finance API]

  Browser --> Caddy
  Caddy --> API
  API --> PG
  API --> RD
  PF --> YF
  PF --> PG
  PF --> RD
  RD --> API
  API -->|WebSocket| Browser
```

### Services

| Service | Port (internal) | Role |
|---------|-----------------|------|
| `caddy` | 8989 (host) → 80 (container) | Static frontend + reverse proxy to API (HTTP only by default) |
| `api` | 8080 | REST API, WebSocket, session cookies |
| `price-fetcher` | 8081 | Polls Yahoo Finance every 5s, evaluates alerts |
| `postgres` | 5432 | Sessions, watchlists, alerts, price time series |
| `redis` | 6379 | Price cache (TTL 5s), pub/sub fan-out |

## Configuration

Copy `.env.example` to `.env`. Key variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `DATABASE_URL` | (compose) | PostgreSQL connection string |
| `REDIS_URL` | `redis://redis:6379/0` | Redis URL |
| `HTTP_PORT` | `8989` | Host port for the web UI (maps to Caddy :80) |
| `FRONTEND_ORIGIN` | `http://localhost:8989` | CORS allowed origin (must match browser URL) |
| `COOKIE_SECURE` | `false` | Set `true` in production behind HTTPS |
| `TRUST_PROXY` | `false` | `true` behind reverse proxy (WebSocket Origin via `X-Forwarded-*`) |
| `ALLOWED_ORIGINS` | | Comma-separated extra CORS origins |
| `SESSION_MAX_AGE_DAYS` | `30` | Rolling session cookie lifetime |
| `POLL_INTERVAL_SEC` | `5` | Yahoo poll interval per symbol set |
| `YAHOO_MAX_CONCURRENCY` | `10` | Max parallel Yahoo requests |

## Reverse proxy and WebSocket

The browser opens **`wss://your-domain/api/v1/ws`** (same host as the page). Traffic path:

```text
Browser → [your TLS proxy] → Docker Caddy :8989 → Go api :8080
```

### Checklist

1. **`.env` on the server** (must match the URL in the browser exactly):

   ```bash
   ENV=production
   FRONTEND_ORIGIN=https://bors.chickendinner.vip
   TRUST_PROXY=true
   COOKIE_SECURE=true
   ```

2. **External reverse proxy** must forward WebSocket upgrade headers to port `8989`:

   - `Upgrade: websocket`
   - `Connection: upgrade`
   - `X-Forwarded-Proto: https`
   - `X-Forwarded-Host: bors.chickendinner.vip`

   See [deploy/nginx-reverse-proxy.example.conf](deploy/nginx-reverse-proxy.example.conf) for Nginx.

3. **Cloudflare**: enable WebSockets for the hostname; orange-cloud proxy can block or timeout WS if misconfigured.

4. **Test** (from your machine):

   ```bash
   curl -i -N \
     -H "Connection: Upgrade" \
     -H "Upgrade: websocket" \
     -H "Sec-WebSocket-Version: 13" \
     -H "Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==" \
     -H "Origin: https://bors.chickendinner.vip" \
     https://bors.chickendinner.vip/api/v1/ws
   ```

   Expect `101 Switching Protocols`. `403` → wrong `FRONTEND_ORIGIN` / Origin check. `502` → proxy not forwarding WS.

## Adding new data sources

1. Implement `marketdata.Provider` in `backend/internal/marketdata/`.
2. Create a client package (e.g. `internal/stooq/`) with `FetchQuote` and `FetchChart`.
3. Wire the provider in `cmd/price-fetcher/main.go` based on an env var such as `MARKET_DATA_PROVIDER=yahoo|stooq`.
4. Document the new variable in `.env.example`.

## Running tests

### Backend (unit)

```bash
cd backend
go test ./...
go test -cover ./internal/alerts/...
```

### Backend (integration)

```bash
cd backend
go test -tags=integration ./test/integration/...
```

### Frontend

```bash
cd frontend
npm ci
npm test
npm run lint
```

## Scaling notes

- ~1000 WebSocket connections are handled comfortably by a single `api` container (2 CPU, 1 GB RAM).
- Yahoo polling depends on **unique symbols**, not user count.
- Horizontal scaling: run multiple `api` instances; Redis pub/sub already fans out price and alert events.

## Yahoo Finance

Data is fetched from the public chart API (`query1.finance.yahoo.com`). No API key required. If you see sustained 403/429 responses, check logs and consider an alternative provider before increasing poll frequency.

## License

MIT
