'use client';

import { useState, useEffect, useCallback, Suspense } from 'react';
import { useSearchParams, useRouter } from 'next/navigation';
import { useAuth } from '@/lib/auth-context';
import { auth, API_BASE_URL } from '@/lib/auth';
import { getFlagEmoji } from '@/lib/country-data';
import Link from 'next/link';

interface FeeBreakdown {
    base_fee: number;
    base_fee_rate: string;
    hop_fees: number;
    hop_fee_rate: string;
    hop_count: number;
    halt_fines: number;
    halt_count: number;
    total_fees: number;
    final_amount: number;
}

interface Transaction {
    id: string;
    amount: number;
    currency: string;
    target_currency: string;
    route: string[];
    status: string;
    total_fees: number;
    final_amount: number;
    admin_profit: number;
    hop_results?: HopResult[];
}

interface HopResult {
    from_country: string;
    to_country: string;
    success: boolean;
    latency_ms: number;
    amount_in: number;
    amount_out: number;
}

interface StripeInitResponse {
    transaction_id: string;
    stripe_client_secret: string;
    stripe_payment_id: string;
    transaction: Transaction;
    fee_breakdown: FeeBreakdown;
    is_mock_mode: boolean;
}

export default function PayPage() {
    return (
        <Suspense fallback={<div className="min-h-screen flex items-center justify-center">Loading...</div>}>
            <PayPageContent />
        </Suspense>
    );
}

function PayPageContent() {
    const { user, isLoading: authLoading } = useAuth();
    const searchParams = useSearchParams();
    const router = useRouter();

    const routeParam = searchParams.get('route');
    const route = routeParam ? routeParam.split(',') : [];

    const [amount, setAmount] = useState('1000');
    const [currency] = useState('USD');
    const [targetCurrency] = useState('USD');

    // Stripe flow state
    const [stripeData, setStripeData] = useState<StripeInitResponse | null>(null);
    const [isInitiating, setIsInitiating] = useState(false);
    const [isProcessing, setIsProcessing] = useState(false);
    const [currentHop, setCurrentHop] = useState(-1);
    const [error, setError] = useState<string | null>(null);
    const [success, setSuccess] = useState(false);
    const [finalTransaction, setFinalTransaction] = useState<Transaction | null>(null);

    // Step 1: Initiate payment at Endpoint A
    const initiatePayment = async () => {
        if (route.length < 2) return;

        setIsInitiating(true);
        setError(null);

        try {
            const response = await auth.authFetch(`${API_BASE_URL}/api/v1/stripe/initiate`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    amount: parseFloat(amount),
                    currency,
                    target_currency: targetCurrency,
                    route
                })
            });

            if (!response.ok) {
                const err = await response.json();
                throw new Error(err.error || 'Failed to initiate payment');
            }

            const data: StripeInitResponse = await response.json();
            setStripeData(data);
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Failed to initiate payment');
        } finally {
            setIsInitiating(false);
        }
    };

    // Step 2: Pay Now - Complete payment at Endpoint B
    const payNow = async () => {
        if (!stripeData) return;

        setIsProcessing(true);
        setError(null);

        // Simulate mesh hops with animation
        for (let i = 0; i < route.length - 1; i++) {
            setCurrentHop(i);
            await new Promise(r => setTimeout(r, 600)); // 600ms per hop animation
        }

        try {
            const response = await auth.authFetch(`${API_BASE_URL}/api/v1/stripe/complete`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    transaction_id: stripeData.transaction_id,
                    stripe_payment_id: stripeData.stripe_payment_id
                })
            });

            const data = await response.json();

            if (data.success) {
                setSuccess(true);
                setFinalTransaction(data.transaction);
            } else {
                setError(data.message || 'Payment failed during mesh processing');
            }
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Payment processing failed');
        } finally {
            setIsProcessing(false);
            setCurrentHop(-1);
        }
    };

    // Initiate when user logs in or route changes
    useEffect(() => {
        if (route.length >= 2 && parseFloat(amount) > 0 && user) {
            initiatePayment();
        }
    }, [user, route.join(',')]);

    // Auto-reinitiate with debounce when amount changes
    useEffect(() => {
        if (!user || route.length < 2 || parseFloat(amount) <= 0) return;

        setStripeData(null); // Clear old data
        const timer = setTimeout(() => {
            initiatePayment();
        }, 800);

        return () => clearTimeout(timer);
    }, [amount]);

    const reinitiate = () => {
        setStripeData(null);
        setError(null);
        initiatePayment();
    };

    if (authLoading) {
        return <div className="min-h-screen flex items-center justify-center bg-slate-950 text-slate-400">Loading...</div>;
    }

    if (!user) {
        return (
            <div className="min-h-screen flex items-center justify-center bg-slate-950">
                <div className="text-center">
                    <p className="text-slate-400 mb-4">Please login to make payments</p>
                    <Link href="/login" className="px-6 py-3 bg-emerald-500 text-white rounded-lg font-semibold">Login</Link>
                </div>
            </div>
        );
    }

    if (route.length < 2) {
        return (
            <div className="min-h-screen flex items-center justify-center bg-slate-950">
                <div className="text-center">
                    <div className="text-6xl mb-4">🗺️</div>
                    <h2 className="text-xl text-white mb-4">No Route Selected</h2>
                    <p className="text-slate-400 mb-6">Select start and end countries from the dashboard</p>
                    <Link href="/dashboard" className="px-6 py-3 bg-emerald-500 text-white rounded-lg font-semibold">
                        Go to Dashboard
                    </Link>
                </div>
            </div>
        );
    }

    // Success Screen
    if (success && finalTransaction) {
        return (
            <div className="min-h-screen bg-slate-950 py-8">
                <div className="max-w-2xl mx-auto px-6">
                    <div className="bg-gradient-to-br from-emerald-500/20 to-cyan-500/20 border border-emerald-500/30 rounded-3xl p-8 text-center">
                        <div className="text-7xl mb-4 animate-bounce">✅</div>
                        <h1 className="text-3xl font-bold text-white mb-2">Payment Successful!</h1>
                        <p className="text-emerald-400 mb-8">Your funds have been transferred through the mesh</p>

                        {/* Transaction Summary */}
                        <div className="bg-slate-800/80 rounded-2xl p-6 text-left mb-6">
                            <h3 className="text-sm text-slate-500 uppercase mb-4">Transaction Summary</h3>

                            <div className="grid grid-cols-2 gap-4 mb-4">
                                <div>
                                    <span className="text-slate-500 text-sm">Amount Sent</span>
                                    <p className="text-2xl font-bold text-white">${finalTransaction.amount.toFixed(2)}</p>
                                </div>
                                <div>
                                    <span className="text-slate-500 text-sm">Amount Received</span>
                                    <p className="text-2xl font-bold text-emerald-400">${finalTransaction.final_amount.toFixed(2)}</p>
                                </div>
                            </div>

                            <div className="border-t border-white/10 pt-4 mb-4">
                                <span className="text-slate-500 text-sm">Route</span>
                                <div className="flex flex-wrap gap-2 mt-2">
                                    {finalTransaction.route.map((code, i) => (
                                        <span key={i} className="inline-flex items-center">
                                            <span className="px-2 py-1 bg-emerald-500/20 text-emerald-400 rounded text-sm">
                                                {getFlagEmoji(code)} {code}
                                            </span>
                                            {i < finalTransaction.route.length - 1 && <span className="text-emerald-500 mx-1">✓→</span>}
                                        </span>
                                    ))}
                                </div>
                            </div>

                            <div className="grid grid-cols-2 gap-4 text-sm">
                                <div>
                                    <span className="text-slate-500">Total Fees</span>
                                    <p className="text-red-400">${finalTransaction.total_fees.toFixed(2)}</p>
                                </div>
                                <div>
                                    <span className="text-slate-500">Transaction ID</span>
                                    <p className="text-slate-300 font-mono text-xs">{finalTransaction.id}</p>
                                </div>
                            </div>
                        </div>

                        {/* Actions */}
                        <div className="flex gap-4 justify-center">
                            <a
                                href={`${API_BASE_URL}/api/v1/receipts/${finalTransaction.id}`}
                                target="_blank"
                                className="px-6 py-3 bg-blue-500 text-white rounded-xl font-semibold hover:bg-blue-600 transition-colors"
                            >
                                📄 Download Receipt
                            </a>
                            <Link
                                href="/pay/success"
                                className="px-6 py-3 bg-purple-500 text-white rounded-xl font-semibold hover:bg-purple-600 transition-colors"
                            >
                                📊 View Analytics
                            </Link>
                        </div>
                    </div>

                    <div className="text-center mt-6">
                        <Link href="/dashboard" className="text-slate-400 hover:text-white">
                            ← Back to Dashboard
                        </Link>
                    </div>
                </div>
            </div>
        );
    }

    // Payment Form
    return (
        <div className="min-h-screen bg-slate-950">
            <header className="bg-slate-900/80 backdrop-blur-lg border-b border-white/10 sticky top-0 z-50">
                <div className="max-w-4xl mx-auto px-6 py-4 flex items-center justify-between">
                    <div className="flex items-center gap-4">
                        <Link href="/dashboard" className="text-slate-400 hover:text-white">← Back</Link>
                        <h1 className="text-xl font-bold text-white">💳 Pay Now</h1>
                    </div>
                    {stripeData?.is_mock_mode && (
                        <span className="px-3 py-1 bg-yellow-500/20 text-yellow-400 rounded-full text-xs">Demo Mode</span>
                    )}
                </div>
            </header>

            <main className="max-w-4xl mx-auto px-6 py-8">
                <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
                    {/* Left: Payment Form */}
                    <div className="bg-slate-800/50 rounded-2xl border border-white/10 p-6">
                        <h2 className="text-lg font-semibold text-white mb-6">Endpoint A: Enter Amount</h2>

                        {/* Route Display */}
                        <div className="mb-6 p-4 bg-slate-900/50 rounded-xl">
                            <label className="text-xs text-slate-500 uppercase">Selected Route</label>
                            <div className="flex flex-wrap gap-2 mt-2">
                                {route.map((code, i) => {
                                    const isActive = isProcessing && i <= currentHop;
                                    const isCurrent = isProcessing && i === currentHop;
                                    return (
                                        <span key={i} className="inline-flex items-center">
                                            <span className={`px-2 py-1 rounded text-sm transition-all ${isCurrent ? 'bg-yellow-500 text-black animate-pulse scale-110' :
                                                isActive ? 'bg-emerald-500/50 text-emerald-300' :
                                                    'bg-slate-700 text-white'
                                                }`}>
                                                {getFlagEmoji(code)} {code}
                                            </span>
                                            {i < route.length - 1 && (
                                                <span className={`mx-1 ${isActive && i < currentHop ? 'text-emerald-400' : 'text-slate-600'}`}>
                                                    {isActive && i < currentHop ? '✓' : '→'}
                                                </span>
                                            )}
                                        </span>
                                    );
                                })}
                            </div>
                        </div>

                        {/* Amount Input */}
                        <div className="mb-6">
                            <label className="block text-sm text-slate-400 mb-2">Amount ({currency}) - Max $999,999</label>
                            <div className="relative">
                                <span className="absolute left-4 top-1/2 -translate-y-1/2 text-slate-400 text-xl">$</span>
                                <input
                                    type="number"
                                    value={amount}
                                    onChange={(e) => {
                                        const val = Math.min(999999, Math.max(0, parseFloat(e.target.value) || 0));
                                        setAmount(String(val));
                                        setStripeData(null);
                                    }}
                                    className="w-full bg-slate-900 border border-white/20 rounded-xl px-10 py-4 text-white text-2xl font-bold focus:outline-none focus:border-emerald-500"
                                    min="1"
                                    max="999999"
                                    step="0.01"
                                    disabled={isProcessing}
                                />
                            </div>
                            <div className="flex justify-between mt-2">
                                <button
                                    onClick={reinitiate}
                                    disabled={isInitiating}
                                    className="text-sm text-emerald-400 hover:text-emerald-300"
                                >
                                    {isInitiating ? 'Calculating...' : '🔄 Recalculate fees'}
                                </button>
                                <div className="flex gap-2">
                                    {[100, 500, 1000, 5000, 10000].map(preset => (
                                        <button
                                            key={preset}
                                            onClick={() => { setAmount(String(preset)); setStripeData(null); }}
                                            className="px-2 py-1 text-xs bg-slate-700 rounded text-slate-300 hover:bg-slate-600"
                                        >
                                            ${preset >= 1000 ? `${preset / 1000}k` : preset}
                                        </button>
                                    ))}
                                </div>
                            </div>
                        </div>

                        {error && (
                            <div className="mb-4 p-3 bg-red-500/10 border border-red-500/30 rounded-lg text-red-400 text-sm">
                                {error}
                            </div>
                        )}

                        {/* PAY NOW Button */}
                        <button
                            onClick={payNow}
                            disabled={!stripeData || isProcessing || isInitiating}
                            className={`w-full py-5 rounded-xl font-bold text-xl transition-all shadow-2xl ${stripeData && !isProcessing
                                ? 'bg-gradient-to-r from-emerald-500 via-cyan-500 to-blue-500 text-white hover:scale-[1.02] shadow-emerald-500/30'
                                : 'bg-slate-700 text-slate-500 cursor-not-allowed'
                                }`}
                        >
                            {isProcessing ? (
                                <span className="flex items-center justify-center gap-3">
                                    <span className="animate-spin text-2xl">⚡</span>
                                    Processing through mesh... ({currentHop + 1}/{route.length - 1})
                                </span>
                            ) : isInitiating ? (
                                'Preparing payment...'
                            ) : (
                                <>💳 PAY NOW - ${amount}</>
                            )}
                        </button>

                        <p className="mt-4 text-center text-slate-500 text-xs">
                            Secured by Stripe • {stripeData?.is_mock_mode ? 'Demo Mode' : 'Live Payment'}
                        </p>
                    </div>

                    {/* Right: Fee Breakdown */}
                    <div className="bg-slate-800/50 rounded-2xl border border-white/10 p-6">
                        <h2 className="text-lg font-semibold text-white mb-6">Endpoint B: Fee Preview</h2>

                        {stripeData?.fee_breakdown ? (
                            <div className="space-y-4">
                                <div className="flex justify-between py-3 border-b border-white/10">
                                    <span className="text-slate-400">Original Amount</span>
                                    <span className="text-white font-bold text-xl">${parseFloat(amount).toFixed(2)}</span>
                                </div>

                                <div className="space-y-3 py-3 border-b border-white/10">
                                    <div className="flex justify-between text-sm">
                                        <span className="text-slate-400">Platform Fee ({stripeData.fee_breakdown.base_fee_rate})</span>
                                        <span className="text-red-400">-${stripeData.fee_breakdown.base_fee.toFixed(2)}</span>
                                    </div>
                                    <div className="flex justify-between text-sm">
                                        <span className="text-slate-400">Hop Fees ({stripeData.fee_breakdown.hop_fee_rate} × {stripeData.fee_breakdown.hop_count})</span>
                                        <span className="text-red-400">-${stripeData.fee_breakdown.hop_fees.toFixed(2)}</span>
                                    </div>
                                    {stripeData.fee_breakdown.halt_fines > 0 && (
                                        <div className="flex justify-between text-sm">
                                            <span className="text-slate-400">Halt Fines</span>
                                            <span className="text-red-400">-${stripeData.fee_breakdown.halt_fines.toFixed(2)}</span>
                                        </div>
                                    )}
                                </div>

                                <div className="pt-2">
                                    <div className="flex justify-between mb-2">
                                        <span className="text-slate-400">Total Fees</span>
                                        <span className="text-red-400 font-semibold">${stripeData.fee_breakdown.total_fees.toFixed(2)}</span>
                                    </div>
                                    <div className="flex justify-between items-center py-4 bg-emerald-500/10 rounded-xl px-4 -mx-4">
                                        <span className="text-emerald-400 font-semibold">You Receive</span>
                                        <span className="text-emerald-400 font-bold text-3xl">${stripeData.fee_breakdown.final_amount.toFixed(2)}</span>
                                    </div>
                                </div>

                                {/* Admin Profit */}
                                <div className="mt-6 p-4 bg-purple-500/10 border border-purple-500/30 rounded-xl">
                                    <div className="flex items-center gap-2 mb-2">
                                        <span className="text-purple-400 font-semibold">💰 Platform Revenue</span>
                                    </div>
                                    <p className="text-purple-300 text-2xl font-bold">${stripeData.fee_breakdown.total_fees.toFixed(2)}</p>
                                </div>
                            </div>
                        ) : (
                            <div className="text-center py-12 text-slate-500">
                                <div className="animate-pulse text-4xl mb-4">💳</div>
                                <p>Enter amount to see fees</p>
                            </div>
                        )}
                    </div>
                </div>
            </main>
        </div>
    );
}
