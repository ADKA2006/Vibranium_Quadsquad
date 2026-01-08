/**
 * Predictive Liquidity Mesh - Authentication Module
 * Handles login, token storage, and role-based access
 */

const Auth = {
    TOKEN_KEY: 'plm_token',
    USER_KEY: 'plm_user',

    // Get stored token
    getToken() {
        return localStorage.getItem(this.TOKEN_KEY);
    },

    // Get stored user
    getUser() {
        const user = localStorage.getItem(this.USER_KEY);
        return user ? JSON.parse(user) : null;
    },

    // Check if logged in
    isLoggedIn() {
        return !!this.getToken();
    },

    // Check if user is admin
    isAdmin() {
        const user = this.getUser();
        return user && user.role === 'ADMIN';
    },

    // Login and store token
    async login(email, password) {
        try {
            const response = await fetch('/api/v1/auth/login', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ email, password })
            });

            if (!response.ok) {
                const error = await response.json();
                throw new Error(error.error || 'Login failed');
            }

            const data = await response.json();

            // Store token and user
            localStorage.setItem(this.TOKEN_KEY, data.token);
            localStorage.setItem(this.USER_KEY, JSON.stringify(data.user));

            return data;
        } catch (error) {
            throw error;
        }
    },

    // Logout
    logout() {
        localStorage.removeItem(this.TOKEN_KEY);
        localStorage.removeItem(this.USER_KEY);
        updateAuthUI();
        hideAdminPanel();
    },

    // Make authenticated request
    async authFetch(url, options = {}) {
        const token = this.getToken();
        if (!token) {
            throw new Error('Not authenticated');
        }

        const headers = {
            ...options.headers,
            'Authorization': `Bearer ${token}`
        };

        return fetch(url, { ...options, headers });
    }
};

// ============================================================================
// LOGIN MODAL
// ============================================================================

function createLoginModal() {
    const modal = document.createElement('div');
    modal.id = 'login-modal';
    modal.className = 'modal';
    modal.innerHTML = `
        <div class="modal-backdrop" onclick="hideLoginModal()"></div>
        <div class="modal-content">
            <div class="modal-header">
                <h2>🔐 Login to PLM</h2>
                <button class="modal-close" onclick="hideLoginModal()">×</button>
            </div>
            <form id="login-form" onsubmit="handleLogin(event)">
                <div class="form-group">
                    <label for="email">Email</label>
                    <input type="email" id="login-email" placeholder="admin@plm.local" required>
                </div>
                <div class="form-group">
                    <label for="password">Password</label>
                    <input type="password" id="login-password" placeholder="Enter password" required>
                </div>
                <div id="login-error" class="error-message"></div>
                <button type="submit" class="btn-primary" id="login-btn">
                    <span>Login</span>
                </button>
            </form>
            <div class="login-hint">
                <p><strong>Demo accounts:</strong></p>
                <code>admin@plm.local</code> / <code>admin123</code> (Admin)<br>
                <code>user@plm.local</code> / <code>user123</code> (User)
            </div>
        </div>
    `;
    document.body.appendChild(modal);
}

function showLoginModal() {
    document.getElementById('login-modal').classList.add('active');
    document.getElementById('login-email').focus();
}

function hideLoginModal() {
    document.getElementById('login-modal').classList.remove('active');
    document.getElementById('login-error').textContent = '';
}

async function handleLogin(event) {
    event.preventDefault();

    const email = document.getElementById('login-email').value;
    const password = document.getElementById('login-password').value;
    const errorEl = document.getElementById('login-error');
    const btn = document.getElementById('login-btn');

    btn.disabled = true;
    btn.innerHTML = '<span>Logging in...</span>';
    errorEl.textContent = '';

    try {
        await Auth.login(email, password);
        hideLoginModal();
        updateAuthUI();
        addEvent('path', `Logged in as ${Auth.getUser().username} (${Auth.getUser().role})`);

        if (Auth.isAdmin()) {
            showAdminPanel();
        }
    } catch (error) {
        errorEl.textContent = error.message;
    } finally {
        btn.disabled = false;
        btn.innerHTML = '<span>Login</span>';
    }
}

// ============================================================================
// AUTH UI UPDATE
// ============================================================================

function updateAuthUI() {
    const authContainer = document.getElementById('auth-container');

    if (Auth.isLoggedIn()) {
        const user = Auth.getUser();
        const roleClass = user.role === 'ADMIN' ? 'role-admin' : 'role-user';

        authContainer.innerHTML = `
            <div class="user-info">
                <span class="user-role ${roleClass}">${user.role}</span>
                <span class="user-name">${user.username}</span>
            </div>
            <button class="btn-logout" onclick="Auth.logout()">Logout</button>
        `;

        // Show admin panel if admin
        if (Auth.isAdmin()) {
            showAdminPanel();
        }
    } else {
        authContainer.innerHTML = `
            <button class="btn-login" onclick="showLoginModal()">Login</button>
        `;
        hideAdminPanel();
    }
}

// ============================================================================
// ADMIN PANEL
// ============================================================================

function createAdminPanel() {
    const panel = document.createElement('div');
    panel.id = 'admin-panel';
    panel.className = 'admin-panel hidden';
    panel.innerHTML = `
        <div class="panel">
            <div class="panel-title">⚡ Admin Controls</div>
            
            <div class="admin-section">
                <h4>Create Node</h4>
                <form id="create-node-form" onsubmit="handleCreateNode(event)">
                    <input type="text" id="node-id" placeholder="Node ID (e.g., sme_006)" required>
                    <select id="node-type" required>
                        <option value="SME">SME</option>
                        <option value="LiquidityProvider">Liquidity Provider</option>
                        <option value="Hub">Hub</option>
                    </select>
                    <input type="text" id="node-region" placeholder="Region (optional)">
                    <button type="submit" class="btn-admin">+ Add Node</button>
                </form>
                <div id="node-result" class="result-message"></div>
            </div>
            
            <div class="admin-section">
                <h4>Create Edge</h4>
                <form id="create-edge-form" onsubmit="handleCreateEdge(event)">
                    <input type="text" id="edge-source" placeholder="Source Node ID" required>
                    <input type="text" id="edge-target" placeholder="Target Node ID" required>
                    <input type="number" id="edge-fee" placeholder="Base Fee (e.g., 0.001)" step="0.0001" required>
                    <input type="number" id="edge-latency" placeholder="Latency (ms)" required>
                    <button type="submit" class="btn-admin">+ Add Edge</button>
                </form>
                <div id="edge-result" class="result-message"></div>
            </div>
            
            <div class="admin-section">
                <h4>Path Preview</h4>
                <form id="preview-form" onsubmit="handlePreview(event)">
                    <input type="text" id="preview-source" placeholder="Source (e.g., sme_001)" required>
                    <input type="text" id="preview-dest" placeholder="Destination (e.g., sme_003)" required>
                    <button type="submit" class="btn-preview">🔍 Find Paths</button>
                </form>
                <div id="preview-result" class="result-message"></div>
            </div>
        </div>
    `;

    // Insert after event panel
    const sidebar = document.querySelector('.sidebar');
    sidebar.appendChild(panel);
}

function showAdminPanel() {
    const panel = document.getElementById('admin-panel');
    if (panel) panel.classList.remove('hidden');
}

function hideAdminPanel() {
    const panel = document.getElementById('admin-panel');
    if (panel) panel.classList.add('hidden');
}

async function handleCreateNode(event) {
    event.preventDefault();

    const resultEl = document.getElementById('node-result');

    try {
        const response = await Auth.authFetch('/api/v1/admin/nodes', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                id: document.getElementById('node-id').value,
                type: document.getElementById('node-type').value,
                region: document.getElementById('node-region').value || undefined
            })
        });

        const data = await response.json();

        if (response.ok) {
            resultEl.className = 'result-message success';
            resultEl.textContent = `✅ Node ${data.node_id} created!`;
            addEvent('path', `Admin created node: ${data.node_id}`);
            document.getElementById('create-node-form').reset();
        } else {
            resultEl.className = 'result-message error';
            resultEl.textContent = `❌ ${data.error}`;
        }
    } catch (error) {
        resultEl.className = 'result-message error';
        resultEl.textContent = `❌ ${error.message}`;
    }
}

async function handleCreateEdge(event) {
    event.preventDefault();

    const resultEl = document.getElementById('edge-result');

    try {
        const response = await Auth.authFetch('/api/v1/admin/edges', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                source_id: document.getElementById('edge-source').value,
                target_id: document.getElementById('edge-target').value,
                base_fee: parseFloat(document.getElementById('edge-fee').value),
                latency_ms: parseInt(document.getElementById('edge-latency').value)
            })
        });

        const data = await response.json();

        if (response.ok) {
            resultEl.className = 'result-message success';
            resultEl.textContent = `✅ Edge created!`;
            addEvent('liquidity', `Admin created edge: ${data.source_id} → ${data.target_id}`);
            document.getElementById('create-edge-form').reset();
        } else {
            resultEl.className = 'result-message error';
            resultEl.textContent = `❌ ${data.error}`;
        }
    } catch (error) {
        resultEl.className = 'result-message error';
        resultEl.textContent = `❌ ${error.message}`;
    }
}

async function handlePreview(event) {
    event.preventDefault();

    const resultEl = document.getElementById('preview-result');
    const source = document.getElementById('preview-source').value;
    const dest = document.getElementById('preview-dest').value;

    try {
        const response = await Auth.authFetch(`/api/v1/settle/preview?source=${source}&destination=${dest}`);
        const data = await response.json();

        if (response.ok && data.paths) {
            let html = `<div class="paths-list">`;
            data.paths.forEach((p, i) => {
                html += `<div class="path-item">
                    <strong>#${i + 1}</strong> ${p.path.join(' → ')}<br>
                    <small>Fee: ${p.total_fee_percent.toFixed(3)}% | Latency: ${p.total_latency_ms}ms</small>
                </div>`;
            });
            html += `</div><small>Computed in ${data.compute_time}</small>`;

            resultEl.className = 'result-message success';
            resultEl.innerHTML = html;
        } else {
            resultEl.className = 'result-message error';
            resultEl.textContent = `❌ ${data.error}`;
        }
    } catch (error) {
        resultEl.className = 'result-message error';
        resultEl.textContent = `❌ ${error.message}`;
    }
}

// ============================================================================
// INITIALIZATION
// ============================================================================

document.addEventListener('DOMContentLoaded', () => {
    createLoginModal();
    createAdminPanel();
    updateAuthUI();
});
