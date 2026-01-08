// Environment configuration
export const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

export interface User {
    id: string;
    email: string;
    username: string;
    role: 'ADMIN' | 'USER';
    is_active: boolean;
}

export interface AuthResponse {
    token: string;
    expires_at: string;
    user: User;
}

const TOKEN_KEY = 'plm_token';
const USER_KEY = 'plm_user';

export const auth = {
    getToken(): string | null {
        if (typeof window === 'undefined') return null;
        return localStorage.getItem(TOKEN_KEY);
    },

    getUser(): User | null {
        if (typeof window === 'undefined') return null;
        const user = localStorage.getItem(USER_KEY);
        return user ? JSON.parse(user) : null;
    },

    setAuth(token: string, user: User): void {
        localStorage.setItem(TOKEN_KEY, token);
        localStorage.setItem(USER_KEY, JSON.stringify(user));
    },

    clearAuth(): void {
        localStorage.removeItem(TOKEN_KEY);
        localStorage.removeItem(USER_KEY);
    },

    isLoggedIn(): boolean {
        return !!this.getToken();
    },

    isAdmin(): boolean {
        const user = this.getUser();
        return user?.role === 'ADMIN';
    },

    async login(email: string, password: string): Promise<AuthResponse> {
        const response = await fetch(`${API_BASE_URL}/api/v1/auth/login`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ email, password }),
        });

        if (!response.ok) {
            const error = await response.json();
            throw new Error(error.error || 'Login failed');
        }

        const data: AuthResponse = await response.json();
        this.setAuth(data.token, data.user);
        return data;
    },

    async register(email: string, password: string, username: string): Promise<AuthResponse> {
        const response = await fetch(`${API_BASE_URL}/api/v1/auth/register`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ email, password, username }),
        });

        if (!response.ok) {
            const error = await response.json();
            throw new Error(error.error || 'Registration failed');
        }

        const data: AuthResponse = await response.json();
        this.setAuth(data.token, data.user);
        return data;
    },

    logout(): void {
        this.clearAuth();
    },

    async authFetch(url: string, options: RequestInit = {}): Promise<Response> {
        const token = this.getToken();
        if (!token) {
            throw new Error('Not authenticated');
        }

        const headers = {
            ...options.headers,
            Authorization: `Bearer ${token}`,
        };

        return fetch(url, { ...options, headers });
    },
};
