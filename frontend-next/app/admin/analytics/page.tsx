'use client';

import { useState, useEffect, useCallback } from 'react';
import { useAuth } from '@/lib/auth-context';
import { auth, API_BASE_URL } from '@/lib/auth';
import Link from 'next/link';
import dynamic from 'next/dynamic';

const ChartComponent = dynamic(() => import('@/components/TransactionCharts'), { ssr: false });

interface Transaction {
    id: string;
    user_id: string;
    amount: number;
    status: string;
    total_fees: number;
    final_amount: number;
    created_at: string;
    route: string[];
    admin_profit: number;
}

interface Analytics {
    total_volume: number;
    total_platform_fee: number;
    total_transactions: number;
    success_count: number;
    failed_count: number;
    pending_count: number;
    success_rate: number;
    daily_volume: Record<string, number>;
    daily_fees: Record<string, number>;
}

interface AdminData {
    stats: {
        total_profit: number;
        total_volume: number;
        success_count: number;
        failed_count: number;
        pending_count: number;
    };
    all_transactions: Transaction[];
    analytics: Analytics;
}

export default function AdminAnalyticsPage() {
    const { user, isAdmin, isLoading: authLoading } = useAuth();
    const [data, setData] = useState<AdminData | null>(null);
    const [isLoading, setIsLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);

    const fetchData = useCallback(async () => {
        try {
            const response = await auth.authFetch(`${API_BASE_URL}/api/v1/admin/payments/stats`);
            if (!response.ok) {
                const err = await response.json();
                throw new Error(err.error || 'Failed to fetch analytics');
            }
            const result = await response.json();
            setData(result);
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Failed to load analytics');
        } finally {
            setIsLoading(false);
        }
    }, []);

    useEffect(() => {
        if (user && isAdmin) {
            fetchData();
            const interval = setInterval(fetchData, 30000); // Refresh every 30s
            return () => clearInterval(interval);
        }
    }, [user, isAdmin, fetchData]);

    if (authLoading || isLoading) {
        return (
            <div className="min-h-screen flex items-center justify-center bg-slate-950 text-slate-400">
                <div className="text-center">
                    <div className="animate-spin text-4xl mb-4">📊</div>
                    <p>Loading analytics...</p>
                </div>
            </div>
        );
    }

    if (!user || !isAdmin) {
        return (
            <div className="min-h-screen flex items-center justify-center bg-slate-950">
                <div className="text-center">
                    <p className="text-4xl mb-4">🔒</p>
                    <p className="text-red-400 mb-4">Admin access required</p>
                    <Link href="/login" className="px-6 py-3 bg-emerald-500 text-white rounded-lg font-semibold">Login as Admin</Link>
                </div>
            </div>
        );
    }

    if (error) {
        return (
            <div className="min-h-screen flex items-center justify-center bg-slate-950">
                <div className="text-center">
                    <p className="text-4xl mb-4">❌</p>
                    <p className="text-red-400 mb-4">{error}</p>
                    <button onClick={fetchData} className="px-6 py-3 bg-emerald-500 text-white rounded-lg">Retry</button>
                </div>
            </div>
        );
    }

    const analytics = data?.analytics;
    const transactions = data?.all_transactions || [];

    // Prepare chart data
    const volumeLabels = Object.keys(analytics?.daily_volume || {}).sort();
    const volumeData = volumeLabels.map(d => analytics?.daily_volume[d] || 0);
    const feesData = volumeLabels.map(d => analytics?.daily_fees[d] || 0);

    return (
        <div className="min-h-screen bg-slate-950">
            <header className="bg-gradient-to-r from-purple-900/50 to-red-900/50 backdrop-blur-lg border-b border-white/10 sticky top-0 z-50">
                <div className="max-w-7xl mx-auto px-6 py-4 flex items-center justify-between">
                    <div className="flex items-center gap-4">
                        <Link href="/admin" className="text-slate-400 hover:text-white">← Admin</Link>
                        <h1 className="text-xl font-bold text-white">📊 Platform Analytics</h1>
                    </div>
                    <div className="flex items-center gap-3">
                        <button onClick={fetchData} className="px-3 py-1 bg-white/10 rounded text-sm text-white hover:bg-white/20">
                            🔄 Refresh
                        </button>
                        <span className="px-3 py-1 bg-red-500 text-white rounded text-sm font-bold">ADMIN</span>
                    </div>
                </div>
            </header>

            <main className="max-w-7xl mx-auto px-6 py-8">
                {/* Key Metrics */}
                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-5 gap-4 mb-8">
                    <div className="bg-gradient-to-br from-emerald-500/20 to-emerald-600/10 rounded-2xl p-6 border border-emerald-500/30">
                        <div className="text-emerald-400 text-sm mb-2">💰 Platform Revenue</div>
                        <div className="text-3xl font-bold text-white">${(analytics?.total_platform_fee || 0).toFixed(2)}</div>
                    </div>
                    <div className="bg-gradient-to-br from-blue-500/20 to-blue-600/10 rounded-2xl p-6 border border-blue-500/30">
                        <div className="text-blue-400 text-sm mb-2">📈 Total Volume</div>
                        <div className="text-3xl font-bold text-white">${(analytics?.total_volume || 0).toFixed(2)}</div>
                    </div>
                    <div className="bg-gradient-to-br from-purple-500/20 to-purple-600/10 rounded-2xl p-6 border border-purple-500/30">
                        <div className="text-purple-400 text-sm mb-2">📝 Transactions</div>
                        <div className="text-3xl font-bold text-white">{analytics?.total_transactions || 0}</div>
                    </div>
                    <div className="bg-gradient-to-br from-green-500/20 to-green-600/10 rounded-2xl p-6 border border-green-500/30">
                        <div className="text-green-400 text-sm mb-2">✅ Success Rate</div>
                        <div className="text-3xl font-bold text-white">{(analytics?.success_rate || 0).toFixed(1)}%</div>
                    </div>
                    <div className="bg-gradient-to-br from-red-500/20 to-red-600/10 rounded-2xl p-6 border border-red-500/30">
                        <div className="text-red-400 text-sm mb-2">❌ Failed</div>
                        <div className="text-3xl font-bold text-white">{analytics?.failed_count || 0}</div>
                    </div>
                </div>

                {/* Charts */}
                {transactions.length > 0 && (
                    <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-8">
                        <div className="bg-slate-800/50 rounded-2xl p-6 border border-white/10">
                            <h3 className="text-lg font-semibold text-white mb-4">📈 Daily Volume</h3>
                            <div className="h-64">
                                <ChartComponent
                                    type="line"
                                    data={{ labels: volumeLabels, data: volumeData }}
                                    color="emerald"
                                />
                            </div>
                        </div>
                        <div className="bg-slate-800/50 rounded-2xl p-6 border border-white/10">
                            <h3 className="text-lg font-semibold text-white mb-4">💰 Daily Fees Collected</h3>
                            <div className="h-64">
                                <ChartComponent
                                    type="bar"
                                    data={{ labels: volumeLabels, data: feesData }}
                                    color="purple"
                                />
                            </div>
                        </div>
                    </div>
                )}

                {/* All Transactions Table */}
                <div className="bg-slate-800/50 rounded-2xl p-6 border border-white/10">
                    <div className="flex items-center justify-between mb-4">
                        <h3 className="text-lg font-semibold text-white">📋 All Transactions</h3>
                        <span className="text-slate-400 text-sm">{transactions.length} total</span>
                    </div>

                    {transactions.length === 0 ? (
                        <div className="text-center py-12 text-slate-500">
                            <div className="text-4xl mb-4">📭</div>
                            <p>No transactions yet</p>
                        </div>
                    ) : (
                        <div className="overflow-x-auto">
                            <table className="w-full">
                                <thead>
                                    <tr className="text-left text-slate-500 text-sm border-b border-white/10">
                                        <th className="pb-3">ID</th>
                                        <th className="pb-3">User</th>
                                        <th className="pb-3">Amount</th>
                                        <th className="pb-3">Fees</th>
                                        <th className="pb-3">Profit</th>
                                        <th className="pb-3">Status</th>
                                        <th className="pb-3">Route</th>
                                        <th className="pb-3">Date</th>
                                    </tr>
                                </thead>
                                <tbody>
                                    {transactions.map((txn) => (
                                        <tr key={txn.id} className="border-b border-white/5 hover:bg-white/5">
                                            <td className="py-3 font-mono text-xs text-slate-400">{txn.id.slice(0, 12)}...</td>
                                            <td className="py-3 text-slate-300 text-sm">{txn.user_id.slice(0, 10)}...</td>
                                            <td className="py-3 text-white font-medium">${txn.amount.toFixed(2)}</td>
                                            <td className="py-3 text-red-400">${txn.total_fees.toFixed(2)}</td>
                                            <td className="py-3 text-emerald-400 font-medium">${txn.admin_profit.toFixed(2)}</td>
                                            <td className="py-3">
                                                <span className={`px-2 py-1 rounded-full text-xs ${txn.status === 'success' ? 'bg-emerald-500/20 text-emerald-400' :
                                                        txn.status === 'failed' ? 'bg-red-500/20 text-red-400' :
                                                            'bg-yellow-500/20 text-yellow-400'
                                                    }`}>
                                                    {txn.status}
                                                </span>
                                            </td>
                                            <td className="py-3 text-slate-400 text-sm">{txn.route?.join('→')}</td>
                                            <td className="py-3 text-slate-500 text-sm">
                                                {new Date(txn.created_at).toLocaleString()}
                                            </td>
                                        </tr>
                                    ))}
                                </tbody>
                            </table>
                        </div>
                    )}
                </div>
            </main>
        </div>
    );
}
