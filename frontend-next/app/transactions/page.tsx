'use client';

import { useState, useEffect, useCallback } from 'react';
import { useAuth } from '@/lib/auth-context';
import { auth, API_BASE_URL } from '@/lib/auth';
import { getFlagEmoji } from '@/lib/country-data';
import Link from 'next/link';

interface Transaction {
    id: string;
    amount: number;
    currency: string;
    target_currency: string;
    route: string[];
    status: string;
    base_fee: number;
    hop_fees: number;
    halt_fines: number;
    total_fees: number;
    final_amount: number;
    admin_profit: number;
    hops_completed: number;
    failed_at?: string;
    created_at: string;
    completed_at?: string;
    card_last4: string;
}

export default function TransactionsPage() {
    const { user, isLoading: authLoading } = useAuth();
    const [transactions, setTransactions] = useState<Transaction[]>([]);
    const [isLoading, setIsLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);

    const fetchTransactions = useCallback(async () => {
        try {
            const response = await auth.authFetch(`${API_BASE_URL}/api/v1/payments/history`);
            if (response.ok) {
                const data = await response.json();
                setTransactions(data.transactions || []);
            } else {
                setError('Failed to fetch transactions');
            }
        } catch {
            setError('Failed to fetch transactions');
        } finally {
            setIsLoading(false);
        }
    }, []);

    useEffect(() => {
        if (user) {
            fetchTransactions();
        }
    }, [user, fetchTransactions]);

    const getStatusColor = (status: string) => {
        switch (status) {
            case 'success': return 'text-emerald-400 bg-emerald-500/10 border-emerald-500/30';
            case 'failed': return 'text-red-400 bg-red-500/10 border-red-500/30';
            case 'processing': return 'text-yellow-400 bg-yellow-500/10 border-yellow-500/30';
            default: return 'text-slate-400 bg-slate-500/10 border-slate-500/30';
        }
    };

    const getStatusIcon = (status: string) => {
        switch (status) {
            case 'success': return '✅';
            case 'failed': return '❌';
            case 'processing': return '⏳';
            default: return '🔄';
        }
    };

    if (authLoading) {
        return <div className="min-h-screen flex items-center justify-center bg-slate-950 text-slate-400">Loading...</div>;
    }

    if (!user) {
        return (
            <div className="min-h-screen flex items-center justify-center bg-slate-950">
                <div className="text-center">
                    <p className="text-slate-400 mb-4">Please login to view transactions</p>
                    <Link href="/login" className="px-6 py-3 bg-emerald-500 text-white rounded-lg font-semibold">Login</Link>
                </div>
            </div>
        );
    }

    return (
        <div className="min-h-screen bg-slate-950">
            {/* Header */}
            <header className="bg-slate-900/80 backdrop-blur-lg border-b border-white/10 sticky top-0 z-50">
                <div className="max-w-6xl mx-auto px-6 py-4 flex items-center justify-between">
                    <div className="flex items-center gap-4">
                        <Link href="/" className="text-slate-400 hover:text-white">← Back</Link>
                        <h1 className="text-xl font-bold text-white">📜 Transaction History</h1>
                    </div>
                    <Link href="/dashboard" className="px-4 py-2 bg-emerald-500 text-white rounded-lg font-semibold text-sm">
                        New Payment
                    </Link>
                </div>
            </header>

            <main className="max-w-6xl mx-auto px-6 py-8">
                {isLoading ? (
                    <div className="text-center py-16 text-slate-400">
                        <div className="animate-spin text-4xl mb-4">⚙️</div>
                        <p>Loading transactions...</p>
                    </div>
                ) : error ? (
                    <div className="text-center py-16 text-red-400">
                        <p>{error}</p>
                    </div>
                ) : transactions.length === 0 ? (
                    <div className="text-center py-16">
                        <div className="text-6xl mb-4">📭</div>
                        <h2 className="text-xl text-white mb-2">No Transactions Yet</h2>
                        <p className="text-slate-400 mb-6">Make your first payment through the mesh</p>
                        <Link href="/dashboard" className="px-6 py-3 bg-emerald-500 text-white rounded-lg font-semibold">
                            Go to Dashboard
                        </Link>
                    </div>
                ) : (
                    <div className="space-y-4">
                        {transactions.map((txn) => (
                            <div key={txn.id} className="bg-slate-800/50 rounded-xl border border-white/10 p-6">
                                <div className="flex items-start justify-between mb-4">
                                    <div>
                                        <div className="flex items-center gap-3 mb-2">
                                            <span className={`px-3 py-1 rounded-full text-xs font-medium border ${getStatusColor(txn.status)}`}>
                                                {getStatusIcon(txn.status)} {txn.status.toUpperCase()}
                                            </span>
                                            <span className="text-slate-500 text-xs">
                                                {new Date(txn.created_at).toLocaleString()}
                                            </span>
                                        </div>
                                        <p className="text-xs text-slate-500 font-mono">{txn.id}</p>
                                    </div>

                                    <div className="text-right">
                                        <div className="text-2xl font-bold text-white">${txn.amount.toFixed(2)}</div>
                                        <div className="text-sm text-slate-400">{txn.currency} → {txn.target_currency}</div>
                                    </div>
                                </div>

                                {/* Route */}
                                <div className="mb-4">
                                    <label className="text-xs text-slate-500 mb-2 block">Route ({txn.route.length - 1} hops)</label>
                                    <div className="flex flex-wrap gap-1">
                                        {txn.route.map((code, i) => {
                                            const isCompleted = i < txn.hops_completed + 1;
                                            const isFailed = code === txn.failed_at;
                                            return (
                                                <span key={i} className="inline-flex items-center">
                                                    <span className={`px-2 py-1 rounded text-xs ${isFailed ? 'bg-red-500/20 text-red-400 border border-red-500/50' :
                                                            isCompleted ? 'bg-emerald-500/20 text-emerald-400' : 'bg-slate-700 text-slate-400'
                                                        }`}>
                                                        {getFlagEmoji(code)} {code}
                                                    </span>
                                                    {i < txn.route.length - 1 && (
                                                        <span className={`mx-1 ${isCompleted ? 'text-emerald-500' : 'text-slate-600'}`}>→</span>
                                                    )}
                                                </span>
                                            );
                                        })}
                                    </div>
                                </div>

                                {/* Fee Details */}
                                <div className="grid grid-cols-4 gap-4 text-sm mb-4">
                                    <div>
                                        <span className="text-slate-500 block">Platform Fee</span>
                                        <span className="text-red-400">${txn.base_fee.toFixed(2)}</span>
                                    </div>
                                    <div>
                                        <span className="text-slate-500 block">Hop Fees</span>
                                        <span className="text-red-400">${txn.hop_fees.toFixed(2)}</span>
                                    </div>
                                    <div>
                                        <span className="text-slate-500 block">Total Fees</span>
                                        <span className="text-red-400 font-semibold">${txn.total_fees.toFixed(2)}</span>
                                    </div>
                                    <div>
                                        <span className="text-slate-500 block">Final Amount</span>
                                        <span className="text-emerald-400 font-bold">${txn.final_amount.toFixed(2)}</span>
                                    </div>
                                </div>

                                {/* Actions */}
                                <div className="flex items-center justify-between pt-4 border-t border-white/10">
                                    <div className="text-xs text-slate-500">
                                        Card ending in {txn.card_last4}
                                    </div>
                                    {txn.status === 'success' && (
                                        <a
                                            href={`${API_BASE_URL}/api/v1/receipts/${txn.id}`}
                                            target="_blank"
                                            className="px-4 py-2 bg-blue-500/20 text-blue-400 rounded-lg text-sm hover:bg-blue-500/30"
                                        >
                                            📄 Download Receipt
                                        </a>
                                    )}
                                </div>
                            </div>
                        ))}
                    </div>
                )}
            </main>
        </div>
    );
}
