# Full-Stack Application Setup: Next.js 14 + Go Backend

Implementation plan for a comprehensive full-stack setup with authentication, Neo4j country data, admin dashboard, and FX rate worker.

## Proposed Changes

### 1. Next.js 14 Frontend Setup

#### [NEW] frontend-next/ 
Create a new Next.js 14 application directory with:
- Pages for home, login, register, and admin dashboard
- API routes that proxy to Go backend
- Auth context with PASETO token management
- Modern UI with dark theme matching existing design

**Key files:**
- `frontend-next/app/page.tsx` - Home page
- `frontend-next/app/login/page.tsx` - Login form
- `frontend-next/app/register/page.tsx` - User registration form  
- `frontend-next/app/admin/page.tsx` - Admin dashboard with node management
- `frontend-next/lib/auth.ts` - Auth utilities and token handling

---

### 2. User & Admin Authentication Enhancements

#### [MODIFY] [auth/password.go](file:///home/bhuvan1707/Desktop/Mock%20Hack%20B2B/Vibranium_Quadsquad_Unofficial/auth/password.go)
Already has Argon2id implementation - no changes needed.

#### [NEW] storage/users/store.go
Create user storage with in-memory store (upgradeable to Postgres):
- `CreateUser(user)` - Hash password with Argon2id, store user
- `GetUserByEmail(email)` - Retrieve user for login
- `GetUserByID(id)` - For token validation

#### [MODIFY] [api/handlers/protected.go](file:///home/bhuvan1707/Desktop/Mock%20Hack%20B2B/Vibranium_Quadsquad_Unofficial/api/handlers/protected.go)
- Add `HandleRegister` endpoint for user registration
- Add proper password verification in `HandleLogin`

#### [MODIFY] [cmd/server/main.go](file:///home/bhuvan1707/Desktop/Mock%20Hack%20B2B/Vibranium_Quadsquad_Unofficial/cmd/server/main.go)
- Add `/api/v1/auth/register` endpoint
- Integrate user store

---

### 3. Neo4j Country Data Bootstrap

#### [NEW] storage/neo4j/countries.go
- Hardcoded list of top 50 GDP countries with properties:
  - `CountryCode` (ISO 3166-1 alpha-3)
  - `Name` (Country name)
  - `Currency` (ISO 4217 currency code)
  - `BaseCredibility`: 0.85
  - `SuccessRate`: Based on economic stability data
- `BootstrapCountries(ctx)` function to create nodes

**Top 50 GDP Countries (sample):**
| Rank | Country | Code | Currency | SuccessRate |
|------|---------|------|----------|-------------|
| 1 | United States | USA | USD | 0.95 |
| 2 | China | CHN | CNY | 0.92 |
| 3 | Germany | DEU | EUR | 0.94 |
| 4 | Japan | JPN | JPY | 0.93 |
| 5 | India | IND | INR | 0.88 |
| ... | ... | ... | ... | ... |

#### [MODIFY] [cmd/server/main.go](file:///home/bhuvan1707/Desktop/Mock%20Hack%20B2B/Vibranium_Quadsquad_Unofficial/cmd/server/main.go)
- Call `BootstrapCountries` on startup if Neo4j is connected

---

### 4. Admin Dashboard Node Management

#### [NEW] api/handlers/country_admin.go
- `HandleListCountries` - GET /api/v1/admin/countries
- `HandleCreateCountry` - POST /api/v1/admin/countries
- `HandleDeleteCountry` - DELETE /api/v1/admin/countries/{code}

#### [MODIFY] [storage/neo4j/client.go](file:///home/bhuvan1707/Desktop/Mock%20Hack%20B2B/Vibranium_Quadsquad_Unofficial/storage/neo4j/client.go)
- Add `GetAllCountries(ctx)` method
- Add `CreateCountryNode(ctx, country)` method
- Add `DeleteCountryNode(ctx, code)` method

---

### 5. FX Rate Worker

#### [NEW] workers/fx_rates/worker.go
Background worker to fetch FX rates:
- Uses ExchangeRate-API (free tier: https://www.exchangerate-api.com/)
- Fetches rates for all 50 country currencies relative to USD
- Updates Neo4j country nodes with current rates
- Runs on configurable interval (default: 1 hour)

#### [NEW] workers/fx_rates/types.go
- `FXRate` struct with currency pair, rate, timestamp
- `ExchangeRateAPIResponse` for API parsing

#### [MODIFY] [cmd/server/main.go](file:///home/bhuvan1707/Desktop/Mock%20Hack%20B2B/Vibranium_Quadsquad_Unofficial/cmd/server/main.go)
- Start FX rate worker goroutine on server startup

---

## User Review Required

> [!IMPORTANT]
> **ExchangeRate-API Key**: You'll need to obtain a free API key from https://www.exchangerate-api.com/. The free tier allows 1500 requests/month.

> [!WARNING]  
> **Next.js Port**: The Next.js frontend will run on port 3000, while the existing Go backend runs on port 8080. You may need to configure CORS or use a reverse proxy.

---

## Verification Plan

### Automated Tests

**1. Run existing Go tests:**
```bash
cd /home/bhuvan1707/Desktop/Mock\ Hack\ B2B/Vibranium_Quadsquad_Unofficial
go test ./tests/... -v
```

**2. Test auth package:**
```bash
go test ./auth/... -v
```

### Manual Verification

**1. User Registration Flow:**
```bash
# Start the server
go run ./cmd/server/main.go

# Register a new user
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"SecurePass123","username":"testuser"}'

# Login with the new user
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"SecurePass123"}'
```

**2. Neo4j Country Nodes:**
```bash
# With Neo4j running (docker-compose up neo4j)
# Open Neo4j Browser at http://localhost:7474
# Run Cypher query:
MATCH (c:Country) RETURN c LIMIT 10
```

**3. Admin Node Management:**
```bash
# Login as admin first
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@plm.local","password":"admin123"}' | jq -r .token)

# List countries
curl http://localhost:8080/api/v1/admin/countries \
  -H "Authorization: Bearer $TOKEN"

# Create a country
curl -X POST http://localhost:8080/api/v1/admin/countries \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"code":"TEST","name":"Test Country","currency":"TST","base_credibility":0.85,"success_rate":0.90}'

# Delete a country
curl -X DELETE http://localhost:8080/api/v1/admin/countries/TEST \
  -H "Authorization: Bearer $TOKEN"
```

**4. FX Rate Worker:**
- Check server logs for FX rate fetch messages
- Query Neo4j for updated rates:
```cypher
MATCH (c:Country) WHERE c.fx_rate IS NOT NULL RETURN c.name, c.currency, c.fx_rate
```

**5. Next.js Frontend:**
```bash
cd frontend-next
npm run dev
# Open http://localhost:3000 in browser
# Test login, registration, and admin dashboard
```