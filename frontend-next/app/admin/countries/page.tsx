'use client';

import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import Link from 'next/link';
import { useAuth } from '@/lib/auth-context';
import { auth, API_BASE_URL } from '@/lib/auth';

interface Country {
    code: string;
    name: string;
    currency: string;
    base_credibility: number;
    success_rate: number;
    gdp_rank: number;
    fx_rate?: number;
}

export default function AdminCountriesPage() {
    const { user, isAdmin, isLoading: authLoading } = useAuth();
    const router = useRouter();
    const [countries, setCountries] = useState<Country[]>([]);
    const [isLoading, setIsLoading] = useState(true);
    const [error, setError] = useState('');
    const [successMessage, setSuccessMessage] = useState('');

    // Form state for adding new country
    const [newCountry, setNewCountry] = useState({
        code: '',
        name: '',
        currency: '',
        success_rate: '0.85',
    });

    useEffect(() => {
        if (!authLoading && (!user || !isAdmin)) {
            router.push('/login');
        }
    }, [user, isAdmin, authLoading, router]);

    useEffect(() => {
        if (isAdmin) {
            fetchCountries();
        }
    }, [isAdmin]);

    const fetchCountries = async () => {
        try {
            const response = await auth.authFetch(`${API_BASE_URL}/api/v1/admin/countries`);
            if (response.ok) {
                const data = await response.json();
                setCountries(data.countries || []);
            }
        } catch {
            setError('Failed to fetch countries');
        } finally {
            setIsLoading(false);
        }
    };

    const handleAddCountry = async (e: React.FormEvent) => {
        e.preventDefault();
        setError('');
        setSuccessMessage('');

        try {
            const response = await auth.authFetch(`${API_BASE_URL}/api/v1/admin/countries`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    code: newCountry.code.toUpperCase(),
                    name: newCountry.name,
                    currency: newCountry.currency.toUpperCase(),
                    base_credibility: 0.85,
                    success_rate: parseFloat(newCountry.success_rate),
                }),
            });

            if (response.ok) {
                setSuccessMessage(`Country ${newCountry.code} added successfully!`);
                setNewCountry({ code: '', name: '', currency: '', success_rate: '0.85' });
                fetchCountries();
            } else {
                const data = await response.json();
                setError(data.error || 'Failed to add country');
            }
        } catch {
            setError('Failed to add country');
        }
    };

    const handleDeleteCountry = async (code: string) => {
        if (!confirm(`Are you sure you want to delete ${code}?`)) return;

        try {
            const response = await auth.authFetch(`${API_BASE_URL}/api/v1/admin/countries/${code}`, {
                method: 'DELETE',
            });

            if (response.ok) {
                setSuccessMessage(`Country ${code} deleted successfully!`);
                fetchCountries();
            } else {
                const data = await response.json();
                setError(data.error || 'Failed to delete country');
            }
        } catch {
            setError('Failed to delete country');
        }
    };

    if (authLoading || !isAdmin) {
        return (
            <div className="min-h-screen flex items-center justify-center">
                <div className="text-slate-400">Loading...</div>
            </div>
        );
    }

    return (
        <div className="min-h-screen">
            {/* Header */}
            <header className="bg-slate-900/80 backdrop-blur-lg border-b border-white/10 sticky top-0 z-50">
                <div className="max-w-7xl mx-auto px-6 py-4 flex items-center justify-between">
                    <div className="flex items-center gap-4">
                        <Link href="/" className="text-slate-400 hover:text-white transition-colors">
                            ← Back
                        </Link>
                        <h1 className="text-xl font-bold text-white">🌍 Country Nodes Manager</h1>
                    </div>
                    <span className="px-3 py-1 bg-red-500 text-white text-sm font-bold rounded">ADMIN</span>
                </div>
            </header>

            <main className="max-w-7xl mx-auto px-6 py-8">
                <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
                    {/* Add Country Form */}
                    <div className="lg:col-span-1">
                        <div className="bg-slate-800/50 rounded-2xl border border-white/10 p-6 sticky top-24">
                            <h2 className="text-lg font-semibold text-white mb-6">Add Country Node</h2>

                            <form onSubmit={handleAddCountry} className="space-y-4">
                                <div>
                                    <label className="block text-sm text-slate-400 mb-1">Code (ISO 3166-1 alpha-3)</label>
                                    <input
                                        type="text"
                                        value={newCountry.code}
                                        onChange={(e) => setNewCountry({ ...newCountry, code: e.target.value })}
                                        placeholder="USA"
                                        maxLength={3}
                                        required
                                        className="w-full px-3 py-2 bg-black/30 border border-white/10 rounded-lg text-white placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-emerald-500"
                                    />
                                </div>

                                <div>
                                    <label className="block text-sm text-slate-400 mb-1">Name</label>
                                    <input
                                        type="text"
                                        value={newCountry.name}
                                        onChange={(e) => setNewCountry({ ...newCountry, name: e.target.value })}
                                        placeholder="United States"
                                        required
                                        className="w-full px-3 py-2 bg-black/30 border border-white/10 rounded-lg text-white placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-emerald-500"
                                    />
                                </div>

                                <div>
                                    <label className="block text-sm text-slate-400 mb-1">Currency (ISO 4217)</label>
                                    <input
                                        type="text"
                                        value={newCountry.currency}
                                        onChange={(e) => setNewCountry({ ...newCountry, currency: e.target.value })}
                                        placeholder="USD"
                                        maxLength={3}
                                        required
                                        className="w-full px-3 py-2 bg-black/30 border border-white/10 rounded-lg text-white placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-emerald-500"
                                    />
                                </div>

                                <div>
                                    <label className="block text-sm text-slate-400 mb-1">Success Rate (0-1)</label>
                                    <input
                                        type="number"
                                        step="0.01"
                                        min="0"
                                        max="1"
                                        value={newCountry.success_rate}
                                        onChange={(e) => setNewCountry({ ...newCountry, success_rate: e.target.value })}
                                        required
                                        className="w-full px-3 py-2 bg-black/30 border border-white/10 rounded-lg text-white placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-emerald-500"
                                    />
                                </div>

                                <div className="text-xs text-slate-500">
                                    BaseCredibility is fixed at 0.85
                                </div>

                                <button
                                    type="submit"
                                    className="w-full py-3 bg-gradient-to-r from-emerald-500 to-cyan-500 text-slate-900 font-semibold rounded-lg hover:opacity-90 transition-opacity"
                                >
                                    + Add Country
                                </button>
                            </form>

                            {error && (
                                <div className="mt-4 p-3 bg-red-500/10 border border-red-500/20 rounded-lg text-red-400 text-sm">
                                    {error}
                                </div>
                            )}

                            {successMessage && (
                                <div className="mt-4 p-3 bg-emerald-500/10 border border-emerald-500/20 rounded-lg text-emerald-400 text-sm">
                                    {successMessage}
                                </div>
                            )}
                        </div>
                    </div>

                    {/* Countries Table */}
                    <div className="lg:col-span-2">
                        <div className="bg-slate-800/50 rounded-2xl border border-white/10 overflow-hidden">
                            <div className="p-6 border-b border-white/10">
                                <h2 className="text-lg font-semibold text-white">
                                    Country Nodes ({countries.length})
                                </h2>
                            </div>

                            {isLoading ? (
                                <div className="p-12 text-center text-slate-400">Loading countries...</div>
                            ) : countries.length === 0 ? (
                                <div className="p-12 text-center text-slate-400">
                                    No countries found. Add your first country or bootstrap from Neo4j.
                                </div>
                            ) : (
                                <div className="overflow-x-auto">
                                    <table className="w-full">
                                        <thead>
                                            <tr className="bg-black/20">
                                                <th className="px-4 py-3 text-left text-xs font-semibold text-slate-400 uppercase">Rank</th>
                                                <th className="px-4 py-3 text-left text-xs font-semibold text-slate-400 uppercase">Code</th>
                                                <th className="px-4 py-3 text-left text-xs font-semibold text-slate-400 uppercase">Name</th>
                                                <th className="px-4 py-3 text-left text-xs font-semibold text-slate-400 uppercase">Currency</th>
                                                <th className="px-4 py-3 text-left text-xs font-semibold text-slate-400 uppercase">Success Rate</th>
                                                <th className="px-4 py-3 text-left text-xs font-semibold text-slate-400 uppercase">FX Rate</th>
                                                <th className="px-4 py-3 text-left text-xs font-semibold text-slate-400 uppercase">Actions</th>
                                            </tr>
                                        </thead>
                                        <tbody className="divide-y divide-white/5">
                                            {countries.map((country) => (
                                                <tr key={country.code} className="hover:bg-white/5">
                                                    <td className="px-4 py-3 text-slate-300">{country.gdp_rank || '-'}</td>
                                                    <td className="px-4 py-3 font-mono text-cyan-400">{country.code}</td>
                                                    <td className="px-4 py-3 text-white">{country.name}</td>
                                                    <td className="px-4 py-3 font-mono text-slate-300">{country.currency}</td>
                                                    <td className="px-4 py-3">
                                                        <span className={`${country.success_rate >= 0.9 ? 'text-emerald-400' : country.success_rate >= 0.8 ? 'text-amber-400' : 'text-red-400'}`}>
                                                            {(country.success_rate * 100).toFixed(0)}%
                                                        </span>
                                                    </td>
                                                    <td className="px-4 py-3 text-slate-400">
                                                        {country.fx_rate ? country.fx_rate.toFixed(4) : '-'}
                                                    </td>
                                                    <td className="px-4 py-3">
                                                        <button
                                                            onClick={() => handleDeleteCountry(country.code)}
                                                            className="px-3 py-1 text-sm bg-red-500/20 text-red-400 rounded hover:bg-red-500/30 transition-colors"
                                                        >
                                                            Delete
                                                        </button>
                                                    </td>
                                                </tr>
                                            ))}
                                        </tbody>
                                    </table>
                                </div>
                            )}
                        </div>
                    </div>
                </div>
            </main>
        </div>
    );
}
