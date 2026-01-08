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

---

## 🔗 Service Endpoints

| Service | URL | Description |
|---------|-----|-------------|
| **Frontend** | http://localhost:3000 | Next.js Dashboard |
| **Backend API** | http://localhost:8080 | Go REST API |
| **Neo4j Browser** | http://localhost:7474 | Graph Database UI |


## 🛡️ Key Features

- **Anti-Fragility:** Auto-retry via alternative routes (3 attempts)
- **Chaos Simulation:** Random node failures with recovery
- **Credibility Scoring:** Dynamic node reliability tracking
- **Digital Signatures:** HMAC-SHA256 receipt verification
- **Role-Based Access:** Admin analytics vs User payments

---

---
## One-Command Deployment

### Prerequisites
- Docker & Docker Compose installed

### Quick Start
```bash
# Download the docker-compose file
curl -O https://raw.githubusercontent.com/ADKA2006/Vibranium_Quadsquad/main/docker-compose.hub.yml

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
