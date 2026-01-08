'use client';

import { useAuth } from '@/lib/auth-context';
import Link from 'next/link';
import { useEffect, useState } from 'react';

export default function Home() {
  const { user, logout, isAdmin } = useAuth();
  const [wsStatus, setWsStatus] = useState<'connecting' | 'connected' | 'disconnected'>('connecting');

  useEffect(() => {
    const ws = new WebSocket('ws://localhost:8080/ws');

    ws.onopen = () => setWsStatus('connected');
    ws.onclose = () => setWsStatus('disconnected');
    ws.onerror = () => setWsStatus('disconnected');

    return () => ws.close();
  }, []);

  return (
    <div className="min-h-screen">
      {/* Header */}
      <header className="bg-slate-900/80 backdrop-blur-lg border-b border-white/10 sticky top-0 z-50">
        <div className="max-w-7xl mx-auto px-6 py-4 flex items-center justify-between">
          <div className="flex items-center gap-3">
            <span className="text-2xl">⚡</span>
            <h1 className="text-xl font-bold bg-gradient-to-r from-emerald-400 to-cyan-400 bg-clip-text text-transparent">
              Predictive Liquidity Mesh
            </h1>
          </div>

          <div className="flex items-center gap-6">
            {/* WebSocket Status */}
            <div className="flex items-center gap-2 text-sm">
              <span className={`w-2.5 h-2.5 rounded-full animate-pulse ${wsStatus === 'connected' ? 'bg-emerald-400' :
                wsStatus === 'connecting' ? 'bg-yellow-400' : 'bg-red-400'
                }`} />
              <span className="text-slate-400">
                {wsStatus === 'connected' ? 'Connected' :
                  wsStatus === 'connecting' ? 'Connecting...' : 'Disconnected'}
              </span>
            </div>

            {/* Auth */}
            {user ? (
              <div className="flex items-center gap-4">
                <div className="flex items-center gap-2">
                  <span className={`px-2 py-0.5 rounded text-xs font-bold ${isAdmin ? 'bg-red-500 text-white' : 'bg-cyan-500 text-slate-900'
                    }`}>
                    {user.role}
                  </span>
                  <span className="text-slate-300 font-medium">{user.username}</span>
                </div>
                <button
                  onClick={logout}
                  className="px-4 py-2 text-sm bg-white/5 hover:bg-red-500/20 rounded-lg transition-colors"
                >
                  Logout
                </button>
              </div>
            ) : (
              <div className="flex gap-2">
                <Link
                  href="/login"
                  className="px-4 py-2 text-sm bg-gradient-to-r from-emerald-500 to-cyan-500 text-slate-900 font-semibold rounded-lg hover:opacity-90 transition-opacity"
                >
                  Login
                </Link>
                <Link
                  href="/register"
                  className="px-4 py-2 text-sm bg-white/10 hover:bg-white/20 rounded-lg transition-colors"
                >
                  Register
                </Link>
              </div>
            )}
          </div>
        </div>
      </header>

      {/* Main Content */}
      <main className="max-w-7xl mx-auto px-6 py-8">
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          {/* Mesh Visualization Placeholder */}
          <div className="lg:col-span-2 bg-slate-800/50 rounded-2xl border border-white/10 p-6 min-h-[500px]">
            <h2 className="text-lg font-semibold text-slate-200 mb-4">Network Mesh</h2>
            <div className="flex items-center justify-center h-[400px] text-slate-500">
              <div className="text-center">
                <div className="text-6xl mb-4">🕸️</div>
                <p>Cytoscape.js visualization will render here</p>
                <p className="text-sm mt-2">Connect to see real-time node updates</p>
              </div>
            </div>
          </div>

          {/* Sidebar */}
          <div className="space-y-6">
            {/* Stats Panel */}
            <div className="bg-slate-800/50 rounded-2xl border border-white/10 p-6">
              <h3 className="text-xs uppercase tracking-wider text-slate-500 mb-4">Network Stats</h3>
              <div className="space-y-4">
                <StatItem label="Transactions/sec" value="0" status="success" />
                <StatItem label="Avg Latency" value="0ms" status="normal" />
                <StatItem label="Active Paths" value="0" status="normal" />
                <StatItem label="Circuit Breakers Open" value="0" status="normal" />
              </div>
            </div>

            {/* User Panel - Available to all logged in users */}
            {user && (
              <div className="bg-slate-800/50 rounded-2xl border border-white/10 p-6">
                <h3 className="text-xs uppercase tracking-wider text-slate-500 mb-4">🌐 Explore</h3>
                <div className="space-y-3">
                  <Link
                    href="/dashboard"
                    className="block w-full py-3 text-center bg-gradient-to-r from-purple-500 via-pink-500 to-red-500 text-white font-semibold rounded-lg hover:opacity-90 transition-opacity shadow-lg shadow-purple-500/30"
                  >
                    🌐 3D Globe Dashboard
                  </Link>
                  <Link
                    href="/graph"
                    className="block w-full py-3 text-center bg-gradient-to-r from-emerald-500 to-cyan-500 text-slate-900 font-semibold rounded-lg hover:opacity-90 transition-opacity"
                  >
                    🔗 FX Rate Network
                  </Link>
                </div>
              </div>
            )}

            {/* Admin Panel - Admin only */}
            {isAdmin && (
              <div className="bg-slate-800/50 rounded-2xl border border-red-500/30 p-6">
                <h3 className="text-xs uppercase tracking-wider text-red-400 mb-4">⚡ Admin Controls</h3>
                <div className="space-y-3">
                  <Link
                    href="/admin"
                    className="block w-full py-3 text-center bg-gradient-to-r from-red-500 to-orange-500 text-white font-semibold rounded-lg hover:opacity-90 transition-opacity"
                  >
                    Node Management
                  </Link>
                  <Link
                    href="/admin/countries"
                    className="block w-full py-3 text-center bg-white/10 hover:bg-white/20 text-white font-semibold rounded-lg transition-colors"
                  >
                    Country Manager
                  </Link>
                </div>
              </div>
            )}

            {/* Quick Actions */}
            <div className="bg-slate-800/50 rounded-2xl border border-white/10 p-6">
              <h3 className="text-xs uppercase tracking-wider text-slate-500 mb-4">Legend</h3>
              <div className="flex flex-wrap gap-3">
                <LegendItem color="bg-cyan-400" label="SME" />
                <LegendItem color="bg-purple-400" label="Liquidity Provider" />
                <LegendItem color="bg-amber-400" label="Hub" />
                <LegendItem color="bg-red-400" label="Circuit Open" />
              </div>
            </div>
          </div>
        </div>
      </main>
    </div>
  );
}

function StatItem({ label, value, status }: { label: string; value: string; status: 'success' | 'warning' | 'danger' | 'normal' }) {
  const colors = {
    success: 'text-emerald-400',
    warning: 'text-amber-400',
    danger: 'text-red-400',
    normal: 'text-white',
  };

  return (
    <div className="flex justify-between items-center py-2 border-b border-white/5 last:border-0">
      <span className="text-slate-400">{label}</span>
      <span className={`font-semibold text-lg ${colors[status]}`}>{value}</span>
    </div>
  );
}

function LegendItem({ color, label }: { color: string; label: string }) {
  return (
    <div className="flex items-center gap-2">
      <span className={`w-3 h-3 rounded-full ${color}`} />
      <span className="text-sm text-slate-400">{label}</span>
    </div>
  );
}
