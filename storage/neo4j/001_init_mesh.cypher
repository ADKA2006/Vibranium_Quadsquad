// ============================================================================
// PREDICTIVE LIQUIDITY MESH - NEO4J RELATIONSHIP MESH
// Cypher Initialization Scripts
// ============================================================================

// ============================================================================
// CONSTRAINTS AND INDEXES
// Run these first to ensure data integrity and query performance
// ============================================================================

// Unique constraint on SME node IDs
CREATE CONSTRAINT sme_id_unique IF NOT EXISTS
FOR (s:SME) REQUIRE s.id IS UNIQUE;

// Unique constraint on Liquidity Provider node IDs
CREATE CONSTRAINT lp_id_unique IF NOT EXISTS
FOR (lp:LiquidityProvider) REQUIRE lp.id IS UNIQUE;

// Unique constraint on Hub node IDs
CREATE CONSTRAINT hub_id_unique IF NOT EXISTS
FOR (h:Hub) REQUIRE h.id IS UNIQUE;

// Index for fast path traversal queries
CREATE INDEX sme_active IF NOT EXISTS FOR (s:SME) ON (s.is_active);
CREATE INDEX lp_active IF NOT EXISTS FOR (lp:LiquidityProvider) ON (lp.is_active);
CREATE INDEX hub_region IF NOT EXISTS FOR (h:Hub) ON (h.region);

// ============================================================================
// CLEAR EXISTING DATA (Optional - for fresh initialization)
// Uncomment if you need to reset the mesh
// ============================================================================

// MATCH (n) DETACH DELETE n;

// ============================================================================
// SEED SME (Small/Medium Enterprise) NODES
// These represent the endpoints that send/receive transactions
// ============================================================================

// Create SME nodes with properties for routing decisions
MERGE (sme1:SME {id: 'sme_001'})
SET sme1.name = 'Acme Manufacturing',
    sme1.region = 'NA_WEST',
    sme1.is_active = true,
    sme1.tier = 'gold',
    sme1.created_at = datetime(),
    sme1.avg_transaction_volume = 50000;

MERGE (sme2:SME {id: 'sme_002'})
SET sme2.name = 'TechFlow Solutions',
    sme2.region = 'NA_EAST',
    sme2.is_active = true,
    sme2.tier = 'platinum',
    sme2.created_at = datetime(),
    sme2.avg_transaction_volume = 150000;

MERGE (sme3:SME {id: 'sme_003'})
SET sme3.name = 'Global Logistics Corp',
    sme3.region = 'EU_CENTRAL',
    sme3.is_active = true,
    sme3.tier = 'gold',
    sme3.created_at = datetime(),
    sme3.avg_transaction_volume = 75000;

MERGE (sme4:SME {id: 'sme_004'})
SET sme4.name = 'Pacific Trading Co',
    sme4.region = 'APAC',
    sme4.is_active = true,
    sme4.tier = 'silver',
    sme4.created_at = datetime(),
    sme4.avg_transaction_volume = 35000;

MERGE (sme5:SME {id: 'sme_005'})
SET sme5.name = 'Nordic Shipping AB',
    sme5.region = 'EU_NORTH',
    sme5.is_active = true,
    sme5.tier = 'gold',
    sme5.created_at = datetime(),
    sme5.avg_transaction_volume = 80000;

// ============================================================================
// SEED LIQUIDITY PROVIDER NODES
// These are the intermediate nodes that provide liquidity for routing
// ============================================================================

MERGE (lp1:LiquidityProvider {id: 'lp_alpha'})
SET lp1.name = 'Alpha Capital',
    lp1.region = 'NA_CENTRAL',
    lp1.is_active = true,
    lp1.liquidity_pool = 10000000,
    lp1.reserve_ratio = 0.15,
    lp1.created_at = datetime();

MERGE (lp2:LiquidityProvider {id: 'lp_beta'})
SET lp2.name = 'Beta Finance',
    lp2.region = 'EU_WEST',
    lp2.is_active = true,
    lp2.liquidity_pool = 8000000,
    lp2.reserve_ratio = 0.12,
    lp2.created_at = datetime();

MERGE (lp3:LiquidityProvider {id: 'lp_gamma'})
SET lp3.name = 'Gamma Settlements',
    lp3.region = 'APAC',
    lp3.is_active = true,
    lp3.liquidity_pool = 12000000,
    lp3.reserve_ratio = 0.18,
    lp3.created_at = datetime();

// ============================================================================
// SEED HUB NODES
// High-throughput routing hubs that connect multiple providers
// ============================================================================

MERGE (hub1:Hub {id: 'hub_primary'})
SET hub1.name = 'Primary Global Hub',
    hub1.region = 'NA_CENTRAL',
    hub1.is_active = true,
    hub1.max_throughput = 100000,
    hub1.current_load = 0.35,
    hub1.created_at = datetime();

MERGE (hub2:Hub {id: 'hub_secondary'})
SET hub2.name = 'Secondary EU Hub',
    hub2.region = 'EU_CENTRAL',
    hub2.is_active = true,
    hub2.max_throughput = 75000,
    hub2.current_load = 0.45,
    hub2.created_at = datetime();

MERGE (hub3:Hub {id: 'hub_backup'})
SET hub3.name = 'Backup APAC Hub',
    hub3.region = 'APAC',
    hub3.is_active = true,
    hub3.max_throughput = 50000,
    hub3.current_load = 0.25,
    hub3.created_at = datetime();

// ============================================================================
// CREATE LIQUIDITY EDGES (PROVIDES_LIQUIDITY)
// Edges from LiquidityProviders to Hubs with routing properties
// ============================================================================

// LP Alpha -> Primary Hub
MATCH (lp:LiquidityProvider {id: 'lp_alpha'}), (hub:Hub {id: 'hub_primary'})
MERGE (lp)-[r:PROVIDES_LIQUIDITY]->(hub)
SET r.base_fee = 0.0015,           // 0.15% base fee
    r.latency = 12,                 // 12ms average latency
    r.liquidity_volume = 5000000,   // Volume available via this edge
    r.is_active = true,
    r.priority = 1,
    r.last_health_check = datetime();

// LP Beta -> Primary Hub
MATCH (lp:LiquidityProvider {id: 'lp_beta'}), (hub:Hub {id: 'hub_primary'})
MERGE (lp)-[r:PROVIDES_LIQUIDITY]->(hub)
SET r.base_fee = 0.0018,
    r.latency = 25,
    r.liquidity_volume = 3000000,
    r.is_active = true,
    r.priority = 2,
    r.last_health_check = datetime();

// LP Beta -> Secondary Hub
MATCH (lp:LiquidityProvider {id: 'lp_beta'}), (hub:Hub {id: 'hub_secondary'})
MERGE (lp)-[r:PROVIDES_LIQUIDITY]->(hub)
SET r.base_fee = 0.0012,
    r.latency = 8,
    r.liquidity_volume = 5000000,
    r.is_active = true,
    r.priority = 1,
    r.last_health_check = datetime();

// LP Gamma -> Backup Hub
MATCH (lp:LiquidityProvider {id: 'lp_gamma'}), (hub:Hub {id: 'hub_backup'})
MERGE (lp)-[r:PROVIDES_LIQUIDITY]->(hub)
SET r.base_fee = 0.0010,
    r.latency = 15,
    r.liquidity_volume = 8000000,
    r.is_active = true,
    r.priority = 1,
    r.last_health_check = datetime();

// LP Gamma -> Primary Hub (cross-region redundancy)
MATCH (lp:LiquidityProvider {id: 'lp_gamma'}), (hub:Hub {id: 'hub_primary'})
MERGE (lp)-[r:PROVIDES_LIQUIDITY]->(hub)
SET r.base_fee = 0.0022,
    r.latency = 85,
    r.liquidity_volume = 4000000,
    r.is_active = true,
    r.priority = 3,
    r.last_health_check = datetime();

// ============================================================================
// CREATE ACCESS EDGES (HAS_ACCESS)
// Edges from SMEs to Liquidity Providers
// ============================================================================

// SME1 -> LP Alpha (primary), LP Beta (backup)
MATCH (sme:SME {id: 'sme_001'}), (lp:LiquidityProvider {id: 'lp_alpha'})
MERGE (sme)-[r:HAS_ACCESS]->(lp)
SET r.base_fee = 0.0008,
    r.latency = 5,
    r.liquidity_volume = 2000000,
    r.is_active = true,
    r.contract_tier = 'gold';

MATCH (sme:SME {id: 'sme_001'}), (lp:LiquidityProvider {id: 'lp_beta'})
MERGE (sme)-[r:HAS_ACCESS]->(lp)
SET r.base_fee = 0.0015,
    r.latency = 45,
    r.liquidity_volume = 1000000,
    r.is_active = true,
    r.contract_tier = 'standard';

// SME2 -> LP Alpha, LP Gamma
MATCH (sme:SME {id: 'sme_002'}), (lp:LiquidityProvider {id: 'lp_alpha'})
MERGE (sme)-[r:HAS_ACCESS]->(lp)
SET r.base_fee = 0.0005,
    r.latency = 8,
    r.liquidity_volume = 5000000,
    r.is_active = true,
    r.contract_tier = 'platinum';

MATCH (sme:SME {id: 'sme_002'}), (lp:LiquidityProvider {id: 'lp_gamma'})
MERGE (sme)-[r:HAS_ACCESS]->(lp)
SET r.base_fee = 0.0012,
    r.latency = 95,
    r.liquidity_volume = 3000000,
    r.is_active = true,
    r.contract_tier = 'gold';

// SME3 -> LP Beta (primary EU provider)
MATCH (sme:SME {id: 'sme_003'}), (lp:LiquidityProvider {id: 'lp_beta'})
MERGE (sme)-[r:HAS_ACCESS]->(lp)
SET r.base_fee = 0.0007,
    r.latency = 10,
    r.liquidity_volume = 3000000,
    r.is_active = true,
    r.contract_tier = 'gold';

// SME4 -> LP Gamma (APAC)
MATCH (sme:SME {id: 'sme_004'}), (lp:LiquidityProvider {id: 'lp_gamma'})
MERGE (sme)-[r:HAS_ACCESS]->(lp)
SET r.base_fee = 0.0010,
    r.latency = 12,
    r.liquidity_volume = 2000000,
    r.is_active = true,
    r.contract_tier = 'silver';

// SME5 -> LP Beta, LP Alpha
MATCH (sme:SME {id: 'sme_005'}), (lp:LiquidityProvider {id: 'lp_beta'})
MERGE (sme)-[r:HAS_ACCESS]->(lp)
SET r.base_fee = 0.0009,
    r.latency = 18,
    r.liquidity_volume = 2500000,
    r.is_active = true,
    r.contract_tier = 'gold';

MATCH (sme:SME {id: 'sme_005'}), (lp:LiquidityProvider {id: 'lp_alpha'})
MERGE (sme)-[r:HAS_ACCESS]->(lp)
SET r.base_fee = 0.0020,
    r.latency = 120,
    r.liquidity_volume = 1000000,
    r.is_active = true,
    r.contract_tier = 'standard';

// ============================================================================
// CREATE HUB INTERCONNECT EDGES
// High-speed links between hubs for cross-region routing
// ============================================================================

MATCH (h1:Hub {id: 'hub_primary'}), (h2:Hub {id: 'hub_secondary'})
MERGE (h1)-[r:INTERCONNECT]->(h2)
SET r.base_fee = 0.0005,
    r.latency = 35,
    r.liquidity_volume = 20000000,
    r.is_active = true,
    r.bandwidth_gbps = 10;

MERGE (h2)-[r2:INTERCONNECT]->(h1)
SET r2.base_fee = 0.0005,
    r2.latency = 35,
    r2.liquidity_volume = 20000000,
    r2.is_active = true,
    r2.bandwidth_gbps = 10;

MATCH (h1:Hub {id: 'hub_primary'}), (h3:Hub {id: 'hub_backup'})
MERGE (h1)-[r:INTERCONNECT]->(h3)
SET r.base_fee = 0.0008,
    r.latency = 75,
    r.liquidity_volume = 15000000,
    r.is_active = true,
    r.bandwidth_gbps = 5;

MERGE (h3)-[r2:INTERCONNECT]->(h1)
SET r2.base_fee = 0.0008,
    r2.latency = 75,
    r2.liquidity_volume = 15000000,
    r2.is_active = true,
    r2.bandwidth_gbps = 5;

MATCH (h2:Hub {id: 'hub_secondary'}), (h3:Hub {id: 'hub_backup'})
MERGE (h2)-[r:INTERCONNECT]->(h3)
SET r.base_fee = 0.0010,
    r.latency = 95,
    r.liquidity_volume = 10000000,
    r.is_active = true,
    r.bandwidth_gbps = 2;

MERGE (h3)-[r2:INTERCONNECT]->(h2)
SET r2.base_fee = 0.0010,
    r2.latency = 95,
    r2.liquidity_volume = 10000000,
    r2.is_active = true,
    r2.bandwidth_gbps = 2;

// ============================================================================
// VERIFICATION QUERIES
// Use these to validate the mesh is correctly initialized
// ============================================================================

// Count all nodes by type
// MATCH (n) RETURN labels(n)[0] AS NodeType, count(n) AS Count;

// Count all edges by type
// MATCH ()-[r]->() RETURN type(r) AS EdgeType, count(r) AS Count;

// Verify all edges have required properties
// MATCH ()-[r]->() WHERE r.base_fee IS NULL OR r.latency IS NULL OR r.liquidity_volume IS NULL
// RETURN type(r), count(r) AS MissingProperties;

// Find all paths from SME1 to any Hub
// MATCH path = (sme:SME {id: 'sme_001'})-[*..3]->(hub:Hub)
// RETURN path, 
//        reduce(fee = 0.0, r IN relationships(path) | fee + r.base_fee) AS total_fee,
//        reduce(lat = 0, r IN relationships(path) | lat + r.latency) AS total_latency;
