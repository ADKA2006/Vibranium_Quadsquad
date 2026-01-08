/**
 * Predictive Liquidity Mesh - Real-Time Visualization
 * Uses Cytoscape.js for graph rendering with WebSocket updates
 */

// Initialize Cytoscape graph
const cy = cytoscape({
    container: document.getElementById('cy'),

    style: [
        // Node base styles
        {
            selector: 'node',
            style: {
                'label': 'data(label)',
                'text-valign': 'bottom',
                'text-halign': 'center',
                'text-margin-y': 8,
                'font-size': '11px',
                'color': '#ffffff',
                'text-outline-color': '#1a1a3e',
                'text-outline-width': 2,
                'width': 45,
                'height': 45,
                'border-width': 3,
                'border-color': '#ffffff20',
                'transition-property': 'background-color, border-color, width, height',
                'transition-duration': '0.3s'
            }
        },
        // SME nodes - cyan
        {
            selector: 'node[type="SME"]',
            style: {
                'background-color': '#00d4ff',
                'border-color': '#00d4ff50'
            }
        },
        // Liquidity Provider nodes - purple
        {
            selector: 'node[type="LiquidityProvider"]',
            style: {
                'background-color': '#aa66ff',
                'border-color': '#aa66ff50'
            }
        },
        // Hub nodes - orange/gold
        {
            selector: 'node[type="Hub"]',
            style: {
                'background-color': '#ffaa00',
                'border-color': '#ffaa0050',
                'width': 55,
                'height': 55
            }
        },
        // Circuit breaker OPEN state - RED with pulsing
        {
            selector: 'node[circuitState="open"]',
            style: {
                'background-color': '#ff4444',
                'border-color': '#ff4444',
                'border-width': 5
            }
        },
        // Inactive nodes - dimmed
        {
            selector: 'node[isActive="false"]',
            style: {
                'opacity': 0.4
            }
        },
        // Edge base styles
        {
            selector: 'edge',
            style: {
                'width': 2,
                'line-color': '#4a4a7f',
                'target-arrow-color': '#4a4a7f',
                'target-arrow-shape': 'triangle',
                'curve-style': 'bezier',
                'opacity': 0.7,
                'transition-property': 'line-color, width, opacity',
                'transition-duration': '0.3s'
            }
        },
        // Active transaction path - glowing green
        {
            selector: 'edge[active="true"]',
            style: {
                'line-color': '#00ff88',
                'target-arrow-color': '#00ff88',
                'width': 4,
                'opacity': 1
            }
        },
        // Glowing animation class
        {
            selector: 'edge.glowing',
            style: {
                'line-color': '#00ff88',
                'width': 6,
                'opacity': 1
            }
        },
        // Selected elements
        {
            selector: ':selected',
            style: {
                'border-width': 4,
                'border-color': '#ffffff'
            }
        }
    ],

    layout: { name: 'preset' },

    // Interaction settings
    minZoom: 0.3,
    maxZoom: 3,
    wheelSensitivity: 0.2
});

// Initial mesh data - matching Neo4j topology
const initialMesh = {
    nodes: [
        // SMEs
        { data: { id: 'sme_001', label: 'Acme Mfg', type: 'SME', isActive: 'true' }, position: { x: 100, y: 200 } },
        { data: { id: 'sme_002', label: 'TechFlow', type: 'SME', isActive: 'true' }, position: { x: 100, y: 400 } },
        { data: { id: 'sme_003', label: 'Global Log', type: 'SME', isActive: 'true' }, position: { x: 700, y: 100 } },
        { data: { id: 'sme_004', label: 'Pacific Trade', type: 'SME', isActive: 'true' }, position: { x: 700, y: 300 } },
        { data: { id: 'sme_005', label: 'Nordic Ship', type: 'SME', isActive: 'true' }, position: { x: 700, y: 500 } },

        // Liquidity Providers
        { data: { id: 'lp_alpha', label: 'Alpha Capital', type: 'LiquidityProvider', isActive: 'true', circuitState: 'closed' }, position: { x: 250, y: 250 } },
        { data: { id: 'lp_beta', label: 'Beta Finance', type: 'LiquidityProvider', isActive: 'true', circuitState: 'closed' }, position: { x: 250, y: 450 } },
        { data: { id: 'lp_gamma', label: 'Gamma Settle', type: 'LiquidityProvider', isActive: 'true', circuitState: 'closed' }, position: { x: 550, y: 350 } },

        // Hubs
        { data: { id: 'hub_primary', label: 'Primary Hub', type: 'Hub', isActive: 'true', circuitState: 'closed' }, position: { x: 400, y: 200 } },
        { data: { id: 'hub_secondary', label: 'EU Hub', type: 'Hub', isActive: 'true', circuitState: 'closed' }, position: { x: 400, y: 400 } },
        { data: { id: 'hub_backup', label: 'APAC Hub', type: 'Hub', isActive: 'true', circuitState: 'closed' }, position: { x: 550, y: 150 } }
    ],
    edges: [
        // SME -> LP edges
        { data: { id: 'e1', source: 'sme_001', target: 'lp_alpha', baseFee: 0.0008 } },
        { data: { id: 'e2', source: 'sme_001', target: 'lp_beta', baseFee: 0.0015 } },
        { data: { id: 'e3', source: 'sme_002', target: 'lp_alpha', baseFee: 0.0005 } },
        { data: { id: 'e4', source: 'sme_002', target: 'lp_gamma', baseFee: 0.0012 } },
        { data: { id: 'e5', source: 'sme_003', target: 'lp_beta', baseFee: 0.0007 } },
        { data: { id: 'e6', source: 'sme_004', target: 'lp_gamma', baseFee: 0.0010 } },
        { data: { id: 'e7', source: 'sme_005', target: 'lp_beta', baseFee: 0.0009 } },
        { data: { id: 'e8', source: 'sme_005', target: 'lp_alpha', baseFee: 0.0020 } },

        // LP -> Hub edges
        { data: { id: 'e9', source: 'lp_alpha', target: 'hub_primary', baseFee: 0.0015 } },
        { data: { id: 'e10', source: 'lp_beta', target: 'hub_primary', baseFee: 0.0018 } },
        { data: { id: 'e11', source: 'lp_beta', target: 'hub_secondary', baseFee: 0.0012 } },
        { data: { id: 'e12', source: 'lp_gamma', target: 'hub_backup', baseFee: 0.0010 } },
        { data: { id: 'e13', source: 'lp_gamma', target: 'hub_primary', baseFee: 0.0022 } },

        // Hub interconnects
        { data: { id: 'e14', source: 'hub_primary', target: 'hub_secondary', baseFee: 0.0005 } },
        { data: { id: 'e15', source: 'hub_secondary', target: 'hub_primary', baseFee: 0.0005 } },
        { data: { id: 'e16', source: 'hub_primary', target: 'hub_backup', baseFee: 0.0008 } },
        { data: { id: 'e17', source: 'hub_backup', target: 'hub_primary', baseFee: 0.0008 } },

        // Hub -> SME edges (destination)
        { data: { id: 'e18', source: 'hub_primary', target: 'sme_003', baseFee: 0.0006 } },
        { data: { id: 'e19', source: 'hub_backup', target: 'sme_004', baseFee: 0.0008 } },
        { data: { id: 'e20', source: 'hub_secondary', target: 'sme_005', baseFee: 0.0007 } }
    ]
};

// Add elements to graph
cy.add(initialMesh.nodes);
cy.add(initialMesh.edges);

// Layout
cy.layout({
    name: 'preset'
}).run();

// Fit to container with padding
cy.fit(50);

// ============================================================================
// WebSocket Connection
// ============================================================================

let ws = null;
let reconnectAttempts = 0;
const maxReconnectAttempts = 10;

function connectWebSocket() {
    const wsUrl = `ws://${window.location.hostname}:8080/ws`;

    try {
        ws = new WebSocket(wsUrl);

        ws.onopen = () => {
            console.log('WebSocket connected');
            updateConnectionStatus(true);
            reconnectAttempts = 0;
        };

        ws.onclose = () => {
            console.log('WebSocket disconnected');
            updateConnectionStatus(false);
            scheduleReconnect();
        };

        ws.onerror = (error) => {
            console.error('WebSocket error:', error);
        };

        ws.onmessage = (event) => {
            try {
                const message = JSON.parse(event.data);
                handleMessage(message);
            } catch (e) {
                console.error('Failed to parse message:', e);
            }
        };
    } catch (e) {
        console.error('Failed to connect WebSocket:', e);
        updateConnectionStatus(false);
    }
}

function scheduleReconnect() {
    if (reconnectAttempts < maxReconnectAttempts) {
        reconnectAttempts++;
        const delay = Math.min(1000 * Math.pow(2, reconnectAttempts), 30000);
        console.log(`Reconnecting in ${delay}ms...`);
        setTimeout(connectWebSocket, delay);
    }
}

function updateConnectionStatus(connected) {
    const statusDot = document.getElementById('ws-status');
    const statusText = document.getElementById('ws-status-text');

    if (connected) {
        statusDot.className = 'status-dot connected';
        statusText.textContent = 'Connected';
    } else {
        statusDot.className = 'status-dot disconnected';
        statusText.textContent = 'Disconnected';
    }
}

// ============================================================================
// Message Handlers
// ============================================================================

function handleMessage(message) {
    switch (message.type) {
        case 'PATH_UPDATE':
            handlePathUpdate(message.data);
            break;
        case 'CIRCUIT_BREAKER':
            handleCircuitBreaker(message.data);
            break;
        case 'LIQUIDITY_UPDATE':
            handleLiquidityUpdate(message.data);
            break;
        case 'NODE_STATUS':
            handleNodeStatus(message.data);
            break;
    }
}

// Animate transaction path with glowing edges
function handlePathUpdate(data) {
    const { path, status, transaction_id } = data;
    // Handle both camelCase and snake_case for old_path
    const oldPath = data.old_path || data.OldPath;

    // If rerouted, show old path fading out
    if (status === 'rerouted' && oldPath) {
        highlightPath(oldPath, '#ff4444', 500);
    }

    // Highlight new path
    const color = status === 'completed' ? '#00ff88' :
        status === 'failed' ? '#ff4444' :
            status === 'rerouted' ? '#ffaa00' : '#00d4ff';

    highlightPath(path, color, 2000);

    // Add event to log
    addEvent('path', `Transaction ${transaction_id?.slice(0, 8) || 'XXX'}: ${path.join(' → ')} [${status}]`);

    // Update active paths count
    updateMetric('active-paths', getActivePaths());
}

function highlightPath(path, color, duration) {
    if (!path || path.length < 2) return;

    // Animate each edge in sequence
    for (let i = 0; i < path.length - 1; i++) {
        const source = path[i];
        const target = path[i + 1];

        setTimeout(() => {
            // Find edge
            const edge = cy.edges().filter(e =>
                e.data('source') === source && e.data('target') === target
            );

            if (edge.length) {
                // Add glow effect
                edge.addClass('glowing');
                edge.style('line-color', color);
                edge.style('target-arrow-color', color);

                // Remove after duration
                setTimeout(() => {
                    edge.removeClass('glowing');
                    edge.style('line-color', '#4a4a7f');
                    edge.style('target-arrow-color', '#4a4a7f');
                }, duration);
            }

            // Pulse the node
            const node = cy.getElementById(target);
            if (node.length) {
                node.animate({
                    style: { 'border-width': 6 }
                }, { duration: 200 }).animate({
                    style: { 'border-width': 3 }
                }, { duration: 200 });
            }
        }, i * 300); // Stagger animation
    }
}

// Handle circuit breaker state change - turn node RED when open
function handleCircuitBreaker(data) {
    const { node_id, state } = data;
    const node = cy.getElementById(node_id);

    if (node.length) {
        node.data('circuitState', state);

        if (state === 'open') {
            // Turn node red and pulse
            node.style('background-color', '#ff4444');
            node.style('border-color', '#ff4444');
            node.animate({
                style: { 'width': 60, 'height': 60 }
            }, { duration: 200 }).animate({
                style: { 'width': 45, 'height': 45 }
            }, { duration: 200 });
        } else if (state === 'closed') {
            // Restore original color based on type
            const type = node.data('type');
            const colors = {
                'SME': '#00d4ff',
                'LiquidityProvider': '#aa66ff',
                'Hub': '#ffaa00'
            };
            node.style('background-color', colors[type] || '#00d4ff');
            node.style('border-color', colors[type] + '50');
        }
    }

    addEvent('circuit', `Circuit breaker ${node_id}: ${state.toUpperCase()}`);
    updateMetric('circuits-open', getOpenCircuits());
}

// Handle liquidity update
function handleLiquidityUpdate(data) {
    const { source_id, target_id, new_volume, change_percent } = data;

    const edge = cy.edges().filter(e =>
        e.data('source') === source_id && e.data('target') === target_id
    );

    if (edge.length) {
        // Visual feedback - brief color change
        const color = change_percent > 0 ? '#00ff88' : '#ff4444';
        edge.style('line-color', color);
        setTimeout(() => {
            edge.style('line-color', '#4a4a7f');
        }, 1000);
    }

    addEvent('liquidity', `${source_id} → ${target_id}: ${change_percent > 0 ? '+' : ''}${change_percent.toFixed(1)}%`);
}

// Handle node status change
function handleNodeStatus(data) {
    const { node_id, is_active } = data;
    const node = cy.getElementById(node_id);

    if (node.length) {
        node.data('isActive', is_active ? 'true' : 'false');
    }
}

// ============================================================================
// UI Helpers
// ============================================================================

function addEvent(type, text) {
    const events = document.getElementById('events');
    const event = document.createElement('div');
    event.className = `event ${type}`;
    event.innerHTML = `
        <div>${text}</div>
        <div class="event-time">${new Date().toLocaleTimeString()}</div>
    `;
    events.insertBefore(event, events.firstChild);

    // Keep only last 50 events
    while (events.children.length > 50) {
        events.removeChild(events.lastChild);
    }
}

function updateMetric(id, value) {
    const el = document.getElementById(id);
    if (el) el.textContent = value;
}

function getActivePaths() {
    return cy.edges('.glowing').length;
}

function getOpenCircuits() {
    return cy.nodes('[circuitState="open"]').length;
}

// ============================================================================
// Demo Mode - Simulate events when not connected
// ============================================================================

let demoMode = false;

function startDemoMode() {
    demoMode = true;
    console.log('Starting demo mode...');

    // Simulate path updates
    setInterval(() => {
        if (!demoMode) return;

        const paths = [
            ['sme_001', 'lp_alpha', 'hub_primary', 'sme_003'],
            ['sme_002', 'lp_gamma', 'hub_backup', 'sme_004'],
            ['sme_005', 'lp_beta', 'hub_secondary', 'hub_primary', 'sme_003'],
            ['sme_001', 'lp_beta', 'hub_secondary', 'sme_005']
        ];

        const path = paths[Math.floor(Math.random() * paths.length)];
        handlePathUpdate({
            transaction_id: Math.random().toString(36).substring(7),
            path: path,
            status: Math.random() > 0.9 ? 'rerouted' : 'completed'
        });

        updateMetric('tps', Math.floor(Math.random() * 50 + 10));
        updateMetric('latency', Math.floor(Math.random() * 5 + 1) + 'ms');
    }, 3000);

    // Simulate occasional circuit breaker events
    setInterval(() => {
        if (!demoMode) return;

        if (Math.random() > 0.8) {
            const nodes = ['lp_alpha', 'lp_beta', 'lp_gamma', 'hub_primary', 'hub_secondary'];
            const nodeId = nodes[Math.floor(Math.random() * nodes.length)];
            const state = Math.random() > 0.5 ? 'open' : 'closed';

            handleCircuitBreaker({ node_id: nodeId, state: state });
        }
    }, 8000);
}

// ============================================================================
// Node Click Interaction - Show connected edges with arrows
// ============================================================================

cy.on('tap', 'node', function(evt) {
    const node = evt.target;
    const nodeId = node.id();
    
    // Reset all edges to default style
    cy.edges().removeClass('highlighted').style({
        'line-color': '#4a4a7f',
        'target-arrow-color': '#4a4a7f',
        'width': 2,
        'opacity': 0.7
    });
    
    // Find all edges connected to this node (both incoming and outgoing)
    const connectedEdges = node.connectedEdges();
    
    // Highlight outgoing edges (from this node) in green
    const outgoingEdges = connectedEdges.filter(edge => edge.data('source') === nodeId);
    outgoingEdges.addClass('highlighted').style({
        'line-color': '#00ff88',
        'target-arrow-color': '#00ff88',
        'width': 4,
        'opacity': 1,
        'target-arrow-shape': 'triangle'
    });
    
    // Highlight incoming edges (to this node) in cyan
    const incomingEdges = connectedEdges.filter(edge => edge.data('target') === nodeId);
    incomingEdges.addClass('highlighted').style({
        'line-color': '#00d4ff',
        'target-arrow-color': '#00d4ff',
        'width': 4,
        'opacity': 1,
        'target-arrow-shape': 'triangle'
    });
    
    // Animate the node
    node.animate({
        style: { 'border-width': 6, 'border-color': '#ffffff' }
    }, { duration: 200 }).animate({
        style: { 'border-width': 3, 'border-color': '#ffffff20' }
    }, { duration: 200 });
    
    // Log the interaction
    addEvent('path', `Node ${node.data('label') || nodeId}: ${outgoingEdges.length} outgoing, ${incomingEdges.length} incoming`);
});

// Click on background to reset edge highlighting
cy.on('tap', function(evt) {
    if (evt.target === cy) {
        // Clicked on background - reset all edges
        cy.edges().removeClass('highlighted').style({
            'line-color': '#4a4a7f',
            'target-arrow-color': '#4a4a7f',
            'width': 2,
            'opacity': 0.7
        });
    }
});

// Initialize
connectWebSocket();

// Start demo mode after 3 seconds if not connected
setTimeout(() => {
    if (!ws || ws.readyState !== WebSocket.OPEN) {
        startDemoMode();
    }
}, 3000);

console.log('Predictive Liquidity Mesh Dashboard initialized');
