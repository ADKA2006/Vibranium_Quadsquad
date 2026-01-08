'use client';

import { useAuth } from '@/lib/auth-context';
import Link from 'next/link';

export default function AdminPage() {
    const { user, isAdmin, isLoading } = useAuth();

    if (isLoading) {
        return (
            <div className="min-h-screen flex items-center justify-center bg-slate-950 text-slate-400">
                Loading...
            </div>
        );
    }

    if (!user) {
        return (
            <div className="min-h-screen flex items-center justify-center bg-slate-950">
                <div className="text-center">
                    <p className="text-slate-400 mb-4">Please login to access admin</p>
                    <Link href="/login" className="px-6 py-3 bg-emerald-500 text-white rounded-lg font-semibold">
                        Login
                    </Link>
                </div>
            </div>
        );
    }

    if (!isAdmin) {
        return (
            <div className="min-h-screen flex items-center justify-center bg-slate-950">
                <div className="text-center">
                    <p className="text-4xl mb-4">🔒</p>
                    <p className="text-red-400 mb-4">Admin access required</p>
                    <Link href="/" className="text-slate-400 hover:text-white">← Back to Home</Link>
                </div>
            </div>
        );
    }

    return (
        <div className="min-h-screen bg-slate-950">
            {/* Header */}
            <header className="bg-slate-900/80 backdrop-blur-lg border-b border-white/10 sticky top-0 z-50">
                <div className="max-w-7xl mx-auto px-6 py-4 flex items-center justify-between">
                    <div className="flex items-center gap-4">
                        <Link href="/" className="text-slate-400 hover:text-white">← Back</Link>
                        <h1 className="text-xl font-bold text-white">⚙️ Node Management Dashboard</h1>
                    </div>
                    <span className="px-3 py-1 bg-red-500 text-white rounded text-sm font-bold">ADMIN</span>
                </div>
            </header>

            <main className="max-w-7xl mx-auto px-6 py-8">
                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
                    {/* Country Nodes */}
                    <Link
                        href="/admin/countries"
                        className="bg-slate-800/50 rounded-2xl border border-white/10 p-6 hover:border-emerald-500/50 transition-colors group"
                    >
                        <div className="text-4xl mb-4">🌍</div>
                        <h2 className="text-lg font-semibold text-white mb-2">Country Nodes</h2>
                        <p className="text-slate-400 text-sm">Manage country nodes in the network. Add, edit, or remove countries.</p>
                        <div className="mt-4 text-emerald-400 text-sm group-hover:translate-x-1 transition-transform">
                            Manage Countries →
                        </div>
                    </Link>

                    {/* FX Rate Network */}
                    <Link
                        href="/graph"
                        className="bg-slate-800/50 rounded-2xl border border-white/10 p-6 hover:border-blue-500/50 transition-colors group"
                    >
                        <div className="text-4xl mb-4">🔗</div>
                        <h2 className="text-lg font-semibold text-white mb-2">FX Rate Network</h2>
                        <p className="text-slate-400 text-sm">View the network graph showing trade connections between countries.</p>
                        <div className="mt-4 text-blue-400 text-sm group-hover:translate-x-1 transition-transform">
                            View Network →
                        </div>
                    </Link>

                    {/* 3D Globe */}
                    <Link
                        href="/dashboard"
                        className="bg-slate-800/50 rounded-2xl border border-white/10 p-6 hover:border-purple-500/50 transition-colors group"
                    >
                        <div className="text-4xl mb-4">🌐</div>
                        <h2 className="text-lg font-semibold text-white mb-2">3D Globe Dashboard</h2>
                        <p className="text-slate-400 text-sm">Interactive globe with country markers and blocking controls.</p>
                        <div className="mt-4 text-purple-400 text-sm group-hover:translate-x-1 transition-transform">
                            Open Dashboard →
                        </div>
                    </Link>

                    {/* System Status */}
                    <div className="bg-slate-800/50 rounded-2xl border border-white/10 p-6">
                        <div className="text-4xl mb-4">📊</div>
                        <h2 className="text-lg font-semibold text-white mb-2">System Status</h2>
                        <div className="space-y-2 text-sm">
                            <div className="flex justify-between">
                                <span className="text-slate-400">Neo4j</span>
                                <span className="text-emerald-400">● Connected</span>
                            </div>
                            <div className="flex justify-between">
                                <span className="text-slate-400">WebSocket</span>
                                <span className="text-emerald-400">● Active</span>
                            </div>
                            <div className="flex justify-between">
                                <span className="text-slate-400">FX Worker</span>
                                <span className="text-amber-400">● Dry-run</span>
                            </div>
                        </div>
                    </div>

                    {/* User Info */}
                    <div className="bg-slate-800/50 rounded-2xl border border-white/10 p-6">
                        <div className="text-4xl mb-4">👤</div>
                        <h2 className="text-lg font-semibold text-white mb-2">Current User</h2>
                        <div className="space-y-2 text-sm">
                            <div className="flex justify-between">
                                <span className="text-slate-400">Username</span>
                                <span className="text-white">{user.username}</span>
                            </div>
                            <div className="flex justify-between">
                                <span className="text-slate-400">Email</span>
                                <span className="text-white">{user.email}</span>
                            </div>
                            <div className="flex justify-between">
                                <span className="text-slate-400">Role</span>
                                <span className="text-red-400">{user.role}</span>
                            </div>
                        </div>
                    </div>
                </div>
            </main>
        </div>
    );
}
