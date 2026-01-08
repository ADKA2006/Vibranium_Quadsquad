'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import Link from 'next/link';
import { useAuth } from '@/lib/auth-context';

export default function LoginPage() {
    const [email, setEmail] = useState('');
    const [password, setPassword] = useState('');
    const [error, setError] = useState('');
    const [isLoading, setIsLoading] = useState(false);
    const { login } = useAuth();
    const router = useRouter();

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        setError('');
        setIsLoading(true);

        try {
            await login(email, password);
            router.push('/');
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Login failed');
        } finally {
            setIsLoading(false);
        }
    };

    return (
        <div className="min-h-screen flex items-center justify-center px-4">
            <div className="w-full max-w-md">
                <div className="bg-slate-800/50 backdrop-blur-lg rounded-2xl border border-white/10 p-8 shadow-2xl">
                    <div className="text-center mb-8">
                        <div className="text-4xl mb-4">🔐</div>
                        <h1 className="text-2xl font-bold text-white">Login to PLM</h1>
                        <p className="text-slate-400 mt-2">Access your dashboard</p>
                    </div>

                    <form onSubmit={handleSubmit} className="space-y-6">
                        <div>
                            <label className="block text-sm text-slate-400 mb-2">Email</label>
                            <input
                                type="email"
                                value={email}
                                onChange={(e) => setEmail(e.target.value)}
                                placeholder="admin@plm.local"
                                required
                                className="w-full px-4 py-3 bg-black/30 border border-white/10 rounded-lg text-white placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-emerald-500 focus:border-transparent transition-all"
                            />
                        </div>

                        <div>
                            <label className="block text-sm text-slate-400 mb-2">Password</label>
                            <input
                                type="password"
                                value={password}
                                onChange={(e) => setPassword(e.target.value)}
                                placeholder="Enter password"
                                required
                                className="w-full px-4 py-3 bg-black/30 border border-white/10 rounded-lg text-white placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-emerald-500 focus:border-transparent transition-all"
                            />
                        </div>

                        {error && (
                            <div className="p-3 bg-red-500/10 border border-red-500/20 rounded-lg text-red-400 text-sm">
                                {error}
                            </div>
                        )}

                        <button
                            type="submit"
                            disabled={isLoading}
                            className="w-full py-3 bg-gradient-to-r from-emerald-500 to-cyan-500 text-slate-900 font-semibold rounded-lg hover:opacity-90 transition-opacity disabled:opacity-50 disabled:cursor-not-allowed"
                        >
                            {isLoading ? 'Logging in...' : 'Login'}
                        </button>
                    </form>

                    <div className="mt-8 pt-6 border-t border-white/10">
                        <p className="text-xs text-slate-500 mb-3">Demo accounts:</p>
                        <div className="space-y-1 text-sm">
                            <p className="text-slate-400">
                                <code className="bg-black/30 px-2 py-0.5 rounded">admin@plm.local</code> / <code className="bg-black/30 px-2 py-0.5 rounded">admin123</code>
                                <span className="text-red-400 ml-2">(Admin)</span>
                            </p>
                            <p className="text-slate-400">
                                <code className="bg-black/30 px-2 py-0.5 rounded">user@plm.local</code> / <code className="bg-black/30 px-2 py-0.5 rounded">user123</code>
                                <span className="text-cyan-400 ml-2">(User)</span>
                            </p>
                        </div>
                    </div>

                    <div className="mt-6 text-center">
                        <p className="text-slate-400 text-sm">
                            Don&apos;t have an account?{' '}
                            <Link href="/register" className="text-emerald-400 hover:underline">
                                Register
                            </Link>
                        </p>
                    </div>
                </div>
            </div>
        </div>
    );
}
