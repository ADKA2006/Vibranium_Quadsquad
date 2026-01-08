# Vibranium_Quadsquad
Domain : FinTech

Idea: Predictive Linquidity Mesh

# OmniMesh: Decentralized Liquidity & Settlement Engine

**OmniMesh** is a high-performance, fault-tolerant financial routing mesh designed to solve the "De-risking" crisis. It bypasses traditional banking bottlenecks using **Entropy-Weighted Routing** and **ZK-Netting** to provide stable, private, and resilient liquidity for SMEs in emerging markets.



---

##  Key Innovations

* **Entropy-Weighted Routing:** Uses Information Entropy to weight graph edges. Instead of "cheapest," it routes through paths least likely to be congested based on real-time network signals.
* **De-risking Resistance:** A P2P structure independent of Western correspondent banking appetites, ensuring geopolitical shifts don't paralyze local commerce.
* **ZK-Netting:** Privacy-preserving matching engine that settles requirements without exposing SME metadata.
* **Heuristic Settlement Scorer:** * **80% Cost:** Fee optimization.
    * **10% Sovereignty:** National credibility signals.
    * **10% Integrity:** Historical node success rates.

---

##  System Architecture & Intricacies

### 1. Intelligent Routing Engine
Utilizes **Yen’s $K$-Shortest Path Algorithm** combined with a sliding window rate limiter (`golang.org/x/time/rate`) and adaptive load shedding.
* **Failover:** If a liquidity node's entropy spikes, the mesh re-calculates 3 alternative paths instantly.

### 2. High-Availability Storage Layer
* **PostgreSQL (HA Ledger):** The immutable source of truth for all transactions. Managed via `pgx/v5`.
* **Neo4j (The Mesh):** Visualizes and manages the relationship mesh between liquidity providers.
* **Redis Sentinel:** Global cache and high-speed **Circuit Breaker** to prevent cascading failures.

### 3. Messaging & Consistency
* **NATS JetStream:** Orchestrates asynchronous task distribution.
* **Outbox Pattern:** Implemented with **Snowflake IDs** to guarantee idempotency—ensuring zero double-spending or lost transactions.

---

##  Tech Stack (The "Go" Specialist)

| Component | Technology | Go Dependency / Tool |
| :--- | :--- | :--- |
| **Language** | Go 1.24+ | `runtime/pprof`, `sync.Pool` |
| **Messaging** | NATS JetStream | `github.com/nats-io/nats.go` |
| **API/RPC** | gRPC + Echo | `google.golang.org/grpc`, `labstack/echo` |
| **DB Drivers** | PGX v5 & Neo4j | `jackc/pgx/v5`, `neo4j/neo4j-go-driver` |
| **Security** | HashiCorp Vault | `github.com/hashicorp/vault/api` |
| **Reliability** | Go-Zero & Breaker | `zeromicro/go-zero`, `sony/gobreaker` |

---

##  Reliability & Observability

### Resilience Patterns
* **Adaptive Load Shedding:** Automatically drops low-priority traffic during peak ledger congestion.
* **mTLS 1.3:** Mandatory encryption for all internal node-to-node communication.
* **Circuit Breaking:** Trips at **200ms** latency to protect the storage layer from saturation.

### The "Golden Signals" Dashboard
Monitored via **Prometheus** and **Grafana**, tracking Latency, Traffic, Errors, and Saturation. Real-time mesh health is visualized using **Cytoscape.js**.

System Architecture


<div align="center">
  <img src="assets/images/System ArchitecturB2B.jpeg" 
       alt="OmniMesh System Architecture Diagram" 
       style="max-width: 100%; height: auto; border-radius: 8px; border: 1px solid #30363d;">
  <p align="center">
    <i>OmniMesh High-Availability Infrastructure: Highlighting Go Worker Nodes, NATS Messaging, and Polyglot Storage Layer.</i>
  </p>
</div>

# Predictive Liquidity Mesh (PLM)

🌐 **Anti-fragile cross-border payment network** with intelligent routing, chaos simulation, and automatic retry logic.

---

## 📁 Repository Structure

```
project-root/
│
├── Dockerfile                    # Backend Dockerfile (Go)
├── frontend-next/
│   └── Dockerfile                # Frontend Dockerfile (Next.js)
│
├── infra/
│   └── docker-compose.yml        # Docker Compose (relative paths)
│
├── cmd/server/                   # Go server entry point
├── api/                          # API handlers
├── payments/                     # Payment & anti-fragility logic
├── storage/                      # Database clients
└── ...
```

---

## 🚀 Quick Start (Docker)

### Prerequisites
- Docker 20+ & Docker Compose v2
- 4GB RAM minimum

### Run Locally

```bash
# 1. Clone the repository
git clone https://github.com/ADKA2006/Vibranium_Quadsquad_Unofficial.git
cd Vibranium_Quadsquad_Unofficial

# 2. Build and start all services
cd infra
docker compose up --build

# 3. Access the application
# Frontend: http://localhost:3000
# Backend:  http://localhost:8080
```

### Stop Services
```bash
cd infra
docker compose down
```

---

## 🔗 Service Endpoints

| Service | URL | Description |
|---------|-----|-------------|
| **Frontend** | http://localhost:3000 | Next.js Dashboard |
| **Backend API** | http://localhost:8080 | Go REST API |
| **Neo4j Browser** | http://localhost:7474 | Graph Database UI |

### Default Credentials
- **User:** `user@plm.local` / `user123`
- **Admin:** `admin@plm.local` / `admin123`

---

## 🐳 Docker Hub Deployment

### Build Images
```bash
# From project root
docker build -t <dockerhub-username>/plm-backend:latest .
docker build -t <dockerhub-username>/plm-frontend:latest ./frontend-next
```

### Push to Docker Hub
```bash
docker login
docker push <dockerhub-username>/plm-backend:latest
docker push <dockerhub-username>/plm-frontend:latest
```

### Pull and Run on Another Machine
```bash
docker pull <dockerhub-username>/plm-backend:latest
docker pull <dockerhub-username>/plm-frontend:latest

# Then use infra/docker-compose.yml
cd infra
docker compose up -d
```

---

## 🌍 Ngrok Deployment (Fallback)

If cloud deployment fails, use Ngrok to expose local services:

### 1. Install Ngrok
```bash
# Download from https://ngrok.com/download
# Or via snap:
snap install ngrok
```

### 2. Authenticate
```bash
ngrok config add-authtoken <your-auth-token>
```

### 3. Start Services Locally
```bash
cd infra
docker compose up --build
```

### 4. Expose with Ngrok
```bash
# Terminal 1: Expose Frontend
ngrok http 3000

# Terminal 2: Expose Backend (if needed)
ngrok http 8080
```

### 5. Public URLs
After running Ngrok, you'll get URLs like:
- Frontend: `https://abc123.ngrok.io`
- Backend: `https://xyz789.ngrok.io`

**⚠️ Note:** Don't restart Ngrok as URLs will change.

---

## 🔧 Environment Variables

Copy `.env.example` to `.env`:

| Variable | Description | Default |
|----------|-------------|---------|
| `TOKEN_SECRET` | JWT signing key (32 chars) | Dev default |
| `NEO4J_PASSWORD` | Neo4j database password | `password` |
| `POSTGRES_PASSWORD` | PostgreSQL password | `postgres` |
| `STRIPE_SECRET_KEY` | Stripe API key | Mock mode |

---

## 🛡️ Key Features

- **Anti-Fragility:** Auto-retry via alternative routes (3 attempts)
- **Chaos Simulation:** Random node failures with recovery
- **Credibility Scoring:** Dynamic node reliability tracking
- **Digital Signatures:** HMAC-SHA256 receipt verification
- **Role-Based Access:** Admin analytics vs User payments

---

## 📚 Documentation

- [ABOUT.md](./ABOUT.md) - Architecture & deployment details
- [WALKTHROUGH.md](./WALKTHROUGH.md) - Feature implementation log

---
## 🚀 One-Command Deployment (For Third Parties)

### Prerequisites
- Docker & Docker Compose installed

### Quick Start
```bash
# Download the docker-compose file
curl -O https://raw.githubusercontent.com/ADKA2006/Vibranium_Quadsquad_Unofficial/main/docker-compose.hub.yml

# Start everything with one command
docker compose -f docker-compose.hub.yml up -d

# Access the application
# Frontend: http://localhost:3000
# Backend:  http://localhost:8080
```

### Docker Hub Images
| Image | URL |
|-------|-----|
| Backend | `docker.io/bhuvan1707/hackathon-backend:latest` |
| Frontend | `docker.io/bhuvan1707/hackathon-frontend:latest` |

### Stop Services
```bash
docker compose -f docker-compose.hub.yml down
```

### Login Credentials
- **User:** `user@plm.local` / `user123`
- **Admin:** `admin@plm.local` / `admin123`
