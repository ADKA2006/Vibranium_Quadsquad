# About PLM - Predictive Liquidity Mesh

## Overview

Predictive Liquidity Mesh (PLM) is a high-performance, anti-fragile cross-border payment network that routes transactions through an intelligent graph of country nodes. The system optimizes for:
- **Lowest fees** via multi-hop routing
- **Highest reliability** via credibility scoring
- **Automatic recovery** via anti-fragility retry logic

## Tech Stack

| Layer | Technology |
|-------|------------|
| **Backend** | Go 1.23 |
| **Frontend** | Next.js 14 (Standalone) |
| **Graph DB** | Neo4j 5 |
| **Ledger** | PostgreSQL 16 |
| **Cache** | Redis 7 |
| **Messaging** | NATS JetStream |
| **Proxy** | Caddy 2 |

---

## 🚀 Deployment & Port Mapping

### Quick Start
```bash
# Start the entire PLM mesh with one command
docker compose up -d

# View logs
docker compose logs -f plm-core
```

### Service Ports

| Service | Internal Port | External Port | Description |
|---------|---------------|---------------|-------------|
| **plm-core** | 8080, 3000 | 8080, 3000 | Go API + Next.js UI |
| **plm-proxy** | 80, 443 | 80, 443 | Caddy HTTPS Proxy |
| **plm-mesh-db** | 7474, 7687 | 7474, 7687 | Neo4j Graph |
| **plm-ledger** | 5432 | (internal only) | PostgreSQL |
| **plm-cache** | 6379 | (internal only) | Redis |
| **plm-nats** | 4222 | (internal only) | NATS Messaging |

### Network Architecture

```
                    ┌─────────────────────────────────────────┐
                    │             plm-external                │
                    │    (User-facing traffic only)           │
                    └───────────────┬───────────────────────┬─┘
                                    │                       │
                              ┌─────▼─────┐           ┌─────▼─────┐
                              │ plm-proxy │           │ plm-core  │
                              │  (Caddy)  │───────────│ (Go+Next) │
                              │  :80/:443 │           │ :8080/:3k │
                              └───────────┘           └─────┬─────┘
                                                            │
                    ┌───────────────────────────────────────┼──────────────────┐
                    │                        plm-internal (isolated)           │
                    │                                       │                  │
              ┌─────▼─────┐  ┌─────────────┐  ┌─────────────┐  ┌──────────────┐
              │ plm-mesh  │  │ plm-ledger  │  │  plm-cache  │  │   plm-nats   │
              │  (Neo4j)  │  │  (Postgres) │  │   (Redis)   │  │ (JetStream)  │
              └───────────┘  └─────────────┘  └─────────────┘  └──────────────┘
```

### Environment Configuration

Copy `.env.example` to `.env` and configure:

```bash
cp .env.example .env
# Edit .env with your production values
```

**Required for Production:**
- `TOKEN_SECRET` - 32-byte PASETO key
- `NEO4J_PASSWORD` - Neo4j auth
- `POSTGRES_PASSWORD` - Postgres auth
- `RECEIPT_SIGNATURE_KEY` - HMAC signing key
- `USER_ID_SALT` - User ID hashing salt

**Optional API Keys:**
- `STRIPE_SECRET_KEY` / `STRIPE_PUBLISHABLE_KEY`
- `EXCHANGE_RATE_API_KEY`

---

## Security Features

- **Non-root containers** - All services run as `plm_user:plm_group`
- **Internal network isolation** - Databases not exposed to host
- **Environment-based secrets** - No hardcoded credentials
- **Health checks** - Dependency ordering via healthchecks
- **HTTPS by default** - Caddy auto-TLS

---

## Anti-Fragility System

When a payment fails mid-route:
1. **Delay notification** sent to user
2. **Automatic retry** via alternative paths (up to 3 attempts)
3. **Hub-based re-routing** through USA, GBR, HKG, SGP, ARE
4. **Full refund** triggered if all retries fail

---

## Development

```bash
# Run backend only (dev mode)
go run ./cmd/server/main.go

# Run frontend only (dev mode)
cd frontend-next && npm run dev

# Run with hot-reload
docker compose -f docker-compose.dev.yml up
```
