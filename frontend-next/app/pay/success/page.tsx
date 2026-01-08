'use client';

import { useState, useEffect, useCallback } from 'react';
import { useAuth } from '@/lib/auth-context';
import { auth, API_BASE_URL } from '@/lib/auth';
import Link from 'next/link';
import dynamic from 'next/dynamic';

// Dynamic import for Chart.js (client-side only)
const ChartComponent = dynamic(() => import('@/components/TransactionCharts'), { ssr: false });

interface ChartData {
    volume_chart: { labels: string[]; data: number[] };
    fees_chart: { labels: string[]; data: number[] };
    status_chart: { labels: string[]; data: number[] };
    summary: { total_transactions: number; success_rate: number };
}

interface Transaction {
    id: string;
    amount: number;
    status: string;
    total_fees: number;
    final_amount: number;
    created_at: string;
    route: string[];
}

export default function PaySuccessPage() {
    const { user, isLoading: authLoading } = useAuth();
    const [chartData, setChartData] = useState<ChartData | null>(null);
    const [transactions, setTransactions] = useState<Transaction[]>([]);
    const [isLoading, setIsLoading] = useState(true);

    const fetchData = useCallback(async () => {
        try {
            const [chartRes, historyRes] = await Promise.all([
                auth.authFetch(`${API_BASE_URL}/api/v1/payments/charts`),
                auth.authFetch(`${API_BASE_URL}/api/v1/payments/history`)
            ]);

            if (chartRes.ok) {
                setChartData(await chartRes.json());
            }
            if (historyRes.ok) {
                const data = await historyRes.json();
                setTransactions(data.transactions || []);
            }
        } catch (err) {
            console.error('Failed to fetch data:', err);
        } finally {
            setIsLoading(false);
        }
    }, []);

    useEffect(() => {
        if (user) {
            fetchData();
        }
    }, [user, fetchData]);

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

    if (!user) {
        return (
            <div className="min-h-screen flex items-center justify-center bg-slate-950">
                <div className="text-center">
                    <p className="text-slate-400 mb-4">Please login to view analytics</p>
                    <Link href="/login" className="px-6 py-3 bg-emerald-500 text-white rounded-lg font-semibold">Login</Link>
                </div>
            </div>
        );
    }

    const totalVolume = transactions.reduce((sum, t) => sum + t.amount, 0);
    const totalFees = transactions.reduce((sum, t) => sum + t.total_fees, 0);
    const successfulTxns = transactions.filter(t => t.status === 'success').length;

    return (
        <div className="min-h-screen bg-slate-950">
            {/* Header */}
            <header className="bg-slate-900/80 backdrop-blur-lg border-b border-white/10 sticky top-0 z-50">
                <div className="max-w-6xl mx-auto px-6 py-4 flex items-center justify-between">
                    <div className="flex items-center gap-4">
                        <Link href="/dashboard" className="text-slate-400 hover:text-white">← Dashboard</Link>
                        <h1 className="text-xl font-bold text-white">📊 Payment Analytics</h1>
                    </div>
                    <Link href="/pay?route=USA,GBR,DEU" className="px-4 py-2 bg-emerald-500 text-white rounded-lg font-semibold text-sm">
                        New Payment
                    </Link>
                </div>
            </header>

            <main className="max-w-6xl mx-auto px-6 py-8">
                {/* Stats Cards */}
                <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-8">
                    <div className="bg-gradient-to-br from-emerald-500/20 to-emerald-600/10 rounded-2xl p-6 border border-emerald-500/30">
                        <div className="text-emerald-400 text-sm mb-2">Total Volume</div>
                        <div className="text-3xl font-bold text-white">${totalVolume.toFixed(2)}</div>
                    </div>
                    <div className="bg-gradient-to-br from-blue-500/20 to-blue-600/10 rounded-2xl p-6 border border-blue-500/30">
                        <div className="text-blue-400 text-sm mb-2">Transactions</div>
                        <div className="text-3xl font-bold text-white">{transactions.length}</div>
                    </div>
                    <div className="bg-gradient-to-br from-purple-500/20 to-purple-600/10 rounded-2xl p-6 border border-purple-500/30">
                        <div className="text-purple-400 text-sm mb-2">Total Fees Paid</div>
                        <div className="text-3xl font-bold text-white">${totalFees.toFixed(2)}</div>
                    </div>
                    <div className="bg-gradient-to-br from-yellow-500/20 to-yellow-600/10 rounded-2xl p-6 border border-yellow-500/30">
                        <div className="text-yellow-400 text-sm mb-2">Success Rate</div>
                        <div className="text-3xl font-bold text-white">
                            {transactions.length > 0 ? ((successfulTxns / transactions.length) * 100).toFixed(1) : 0}%
                        </div>
                    </div>
                </div>

                {/* Charts */}
                {chartData && transactions.length > 0 ? (
                    <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-8">
                        <div className="bg-slate-800/50 rounded-2xl p-6 border border-white/10">
                            <h3 className="text-lg font-semibold text-white mb-4">📈 Transaction Volume</h3>
                            <div className="h-64">
                                <ChartComponent type="line" data={chartData.volume_chart} color="emerald" />
                            </div>
                        </div>
                        <div className="bg-slate-800/50 rounded-2xl p-6 border border-white/10">
                            <h3 className="text-lg font-semibold text-white mb-4">💰 Fees Over Time</h3>
                            <div className="h-64">
                                <ChartComponent type="bar" data={chartData.fees_chart} color="purple" />
                            </div>
                        </div>
                        <div className="bg-slate-800/50 rounded-2xl p-6 border border-white/10 lg:col-span-2">
                            <h3 className="text-lg font-semibold text-white mb-4">📊 Payment Status Distribution</h3>
                            <div className="h-64 flex justify-center">
                                <div className="w-64">
                                    <ChartComponent type="doughnut" data={chartData.status_chart} />
                                </div>
                            </div>
                        </div>
                    </div>
                ) : (
                    <div className="bg-slate-800/50 rounded-2xl p-12 border border-white/10 text-center mb-8">
                        <div className="text-6xl mb-4">📭</div>
                        <h3 className="text-xl text-white mb-2">No Transactions Yet</h3>
                        <p className="text-slate-400 mb-6">Make your first payment to see analytics</p>
                        <Link href="/dashboard" className="px-6 py-3 bg-emerald-500 text-white rounded-lg font-semibold">
                            Go to Dashboard
                        </Link>
                    </div>
                )}

                {/* Recent Transactions */}
                {transactions.length > 0 && (
                    <div className="bg-slate-800/50 rounded-2xl p-6 border border-white/10">
                        <div className="flex items-center justify-between mb-4">
                            <h3 className="text-lg font-semibold text-white">Recent Transactions</h3>
                            <Link href="/transactions" className="text-emerald-400 text-sm hover:text-emerald-300">
                                View all →
                            </Link>
                        </div>
                        <div className="overflow-x-auto">
                            <table className="w-full">
                                <thead>
                                    <tr className="text-left text-slate-500 text-sm border-b border-white/10">
                                        <th className="pb-3">Date</th>
                                        <th className="pb-3">Amount</th>
                                        <th className="pb-3">Fees</th>
                                        <th className="pb-3">Received</th>
                                        <th className="pb-3">Status</th>
                                        <th className="pb-3">Receipt</th>
                                    </tr>
                                </thead>
                                <tbody>
                                    {transactions.slice(0, 5).map((txn) => (
                                        <tr key={txn.id} className="border-b border-white/5">
                                            <td className="py-3 text-slate-400 text-sm">
                                                {new Date(txn.created_at).toLocaleDateString()}
                                            </td>
                                            <td className="py-3 text-white font-medium">${txn.amount.toFixed(2)}</td>
                                            <td className="py-3 text-red-400">${txn.total_fees.toFixed(2)}</td>
                                            <td className="py-3 text-emerald-400 font-medium">${txn.final_amount.toFixed(2)}</td>
                                            <td className="py-3">
                                                <span className={`px-2 py-1 rounded-full text-xs ${txn.status === 'success' ? 'bg-emerald-500/20 text-emerald-400' :
                                                        txn.status === 'failed' ? 'bg-red-500/20 text-red-400' :
                                                            'bg-yellow-500/20 text-yellow-400'
                                                    }`}>
                                                    {txn.status}
                                                </span>
                                            </td>
                                            <td className="py-3">
                                                {txn.status === 'success' && (
                                                    <a
                                                        href={`${API_BASE_URL}/api/v1/receipts/${txn.id}`}
                                                        target="_blank"
                                                        className="text-blue-400 hover:text-blue-300 text-sm"
                                                    >
                                                        📄 PDF
                                                    </a>
                                                )}
                                            </td>
                                        </tr>
                                    ))}
                                </tbody>
                            </table>
                        </div>
                    </div>
                )}
            </main>
        </div>
    );
}
