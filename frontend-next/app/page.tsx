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
          {/* Payment Analytics - Role Based */}
          <div className="lg:col-span-2 bg-slate-800/50 rounded-2xl border border-white/10 p-6 min-h-[500px]">
            {!user ? (
              <div className="flex items-center justify-center h-[400px]">
                <div className="text-center">
                  <div className="text-6xl mb-4">🔐</div>
                  <h2 className="text-xl font-semibold text-white mb-2">Welcome to PLM</h2>
                  <p className="text-slate-400 mb-6">Login to access payment analytics and transaction features</p>
                  <Link href="/login" className="px-6 py-3 bg-gradient-to-r from-emerald-500 to-cyan-500 text-white rounded-lg font-semibold">
                    Get Started
                  </Link>
                </div>
              </div>
            ) : isAdmin ? (
              <div>
                <h2 className="text-lg font-semibold text-slate-200 mb-4">📊 Admin Analytics</h2>
                <div className="grid grid-cols-2 gap-4 mb-6">
                  <div className="bg-gradient-to-br from-emerald-500/20 to-emerald-600/10 rounded-xl p-4 border border-emerald-500/30">
                    <div className="text-emerald-400 text-sm mb-1">💰 Platform Revenue</div>
                    <div className="text-2xl font-bold text-white">View Dashboard</div>
                  </div>
                  <div className="bg-gradient-to-br from-purple-500/20 to-purple-600/10 rounded-xl p-4 border border-purple-500/30">
                    <div className="text-purple-400 text-sm mb-1">📈 All Transactions</div>
                    <div className="text-2xl font-bold text-white">Access Full Data</div>
                  </div>
                </div>
                <Link
                  href="/admin/analytics"
                  className="block w-full py-4 text-center bg-gradient-to-r from-red-500 to-orange-500 text-white font-bold rounded-xl hover:opacity-90 transition-opacity"
                >
                  📊 Open Admin Analytics Dashboard
                </Link>
              </div>
            ) : (
              <div>
                <h2 className="text-lg font-semibold text-slate-200 mb-4">💳 Your Payment Analytics</h2>
                <div className="grid grid-cols-2 gap-4 mb-6">
                  <div className="bg-gradient-to-br from-emerald-500/20 to-emerald-600/10 rounded-xl p-4 border border-emerald-500/30">
                    <div className="text-emerald-400 text-sm mb-1">📤 Send Payments</div>
                    <div className="text-sm text-slate-300">Transfer across countries</div>
                  </div>
                  <div className="bg-gradient-to-br from-blue-500/20 to-blue-600/10 rounded-xl p-4 border border-blue-500/30">
                    <div className="text-blue-400 text-sm mb-1">📋 Transaction History</div>
                    <div className="text-sm text-slate-300">View all your payments</div>
                  </div>
                </div>
                <div className="space-y-3">
                  <Link
                    href="/dashboard"
                    className="block w-full py-4 text-center bg-gradient-to-r from-emerald-500 to-cyan-500 text-white font-bold rounded-xl hover:opacity-90 transition-opacity"
                  >
                    🌐 Open Payment Dashboard
                  </Link>
                  <Link
                    href="/transactions"
                    className="block w-full py-3 text-center bg-white/10 text-white font-semibold rounded-xl hover:bg-white/20 transition-colors"
                  >
                    📋 View Transaction History
                  </Link>
                </div>
              </div>
            )}
          </div>

          {/* Sidebar */}
          <div className="space-y-6">
            {/* Quick Links Panel */}
            <div className="bg-slate-800/50 rounded-2xl border border-white/10 p-6">
              <h3 className="text-xs uppercase tracking-wider text-slate-500 mb-4">Quick Actions</h3>
              <div className="space-y-4">
                {user ? (
                  <>
                    <Link href="/dashboard" className="flex items-center gap-3 p-3 rounded-lg bg-white/5 hover:bg-white/10 transition-colors">
                      <span className="text-xl">🌐</span>
                      <div>
                        <div className="text-white font-medium">Globe Dashboard</div>
                        <div className="text-xs text-slate-400">Select routes & pay</div>
                      </div>
                    </Link>
                    {!isAdmin && (
                      <Link href="/pay" className="flex items-center gap-3 p-3 rounded-lg bg-emerald-500/10 hover:bg-emerald-500/20 transition-colors border border-emerald-500/30">
                        <span className="text-xl">💳</span>
                        <div>
                          <div className="text-emerald-400 font-medium">Make Payment</div>
                          <div className="text-xs text-slate-400">Transfer funds now</div>
                        </div>
                      </Link>
                    )}
                  </>
                ) : (
                  <div className="text-center py-4 text-slate-500">
                    <p>Login to access features</p>
                  </div>
                )}
              </div>
            </div>

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
