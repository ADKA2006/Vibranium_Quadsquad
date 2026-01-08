'use client';

import { useEffect, useRef, useState, useCallback } from 'react';
import cytoscape, { Core } from 'cytoscape';
import { useAuth } from '@/lib/auth-context';
import { auth, API_BASE_URL } from '@/lib/auth';
import Link from 'next/link';

interface Country {
    code: string;
    name: string;
    currency: string;
    base_credibility: number;
    success_rate: number;
    gdp_rank: number;
    fx_rate?: number;
}

// Major trade connections between countries (currency pairs)
const TRADE_CONNECTIONS = [
    // USD hub connections
    ['USA', 'GBR'], ['USA', 'EUR'], ['USA', 'JPN'], ['USA', 'CHN'], ['USA', 'CAN'],
    ['USA', 'MEX'], ['USA', 'AUS'], ['USA', 'CHE'], ['USA', 'KOR'], ['USA', 'IND'],
    ['USA', 'BRA'], ['USA', 'SGP'], ['USA', 'HKG'],
    // EUR connections
    ['DEU', 'FRA'], ['DEU', 'ITA'], ['DEU', 'ESP'], ['DEU', 'NLD'], ['DEU', 'BEL'],
    ['DEU', 'AUT'], ['DEU', 'POL'], ['DEU', 'CHE'], ['DEU', 'GBR'],
    ['FRA', 'ITA'], ['FRA', 'ESP'], ['FRA', 'BEL'], ['FRA', 'NLD'],
    // Asian connections
    ['CHN', 'JPN'], ['CHN', 'KOR'], ['CHN', 'HKG'], ['CHN', 'TWN'], ['CHN', 'SGP'],
    ['CHN', 'THA'], ['CHN', 'VNM'], ['CHN', 'MYS'], ['CHN', 'IDN'], ['CHN', 'IND'],
    ['JPN', 'KOR'], ['JPN', 'TWN'], ['JPN', 'SGP'], ['JPN', 'THA'],
    ['SGP', 'MYS'], ['SGP', 'HKG'], ['SGP', 'THA'], ['SGP', 'IDN'],
    // Middle East
    ['SAU', 'ARE'], ['SAU', 'EGY'], ['ARE', 'IND'],
    // South America
    ['BRA', 'ARG'], ['BRA', 'MEX'], ['BRA', 'CHL'], ['BRA', 'COL'],
    ['MEX', 'COL'], ['CHL', 'PER'], ['ARG', 'CHL'],
    // Africa
    ['ZAF', 'NGA'], ['ZAF', 'EGY'],
    // Oceania
    ['AUS', 'NZL'], ['AUS', 'SGP'], ['AUS', 'JPN'], ['AUS', 'CHN'],
    // Nordic
    ['SWE', 'NOR'], ['SWE', 'DNK'], ['SWE', 'FIN'], ['NOR', 'DNK'],
    // Eastern Europe
    ['POL', 'CZE'], ['CZE', 'AUT'], ['ROU', 'POL'],
    // Other major pairs
    ['GBR', 'IRL'], ['GBR', 'CHE'], ['GBR', 'IND'], ['GBR', 'HKG'],
    ['CHE', 'AUT'], ['ISR', 'USA'], ['TUR', 'DEU'],
];

// Map ISO3 to use EUR for eurozone countries
const EUROZONE = ['DEU', 'FRA', 'ITA', 'ESP', 'NLD', 'BEL', 'AUT', 'IRL', 'FIN', 'PRT'];

export default function CountryGraphPage() {
    const { user, isAdmin, isLoading: authLoading } = useAuth();
    const containerRef = useRef<HTMLDivElement>(null);
    const cyRef = useRef<Core | null>(null);
    const [countries, setCountries] = useState<Country[]>([]);
    const [isLoading, setIsLoading] = useState(true);
    const [selectedCountry, setSelectedCountry] = useState<Country | null>(null);
    const [fxRates, setFxRates] = useState<Record<string, number>>({});

    // Fetch countries
    const fetchCountries = useCallback(async () => {
        try {
            const response = await auth.authFetch(`${API_BASE_URL}/api/v1/admin/countries`);
            if (response.ok) {
                const data = await response.json();
                setCountries(data.countries || []);

                // Extract FX rates
                const rates: Record<string, number> = {};
                (data.countries || []).forEach((c: Country) => {
                    if (c.fx_rate) {
                        rates[c.currency] = c.fx_rate;
                    }
                });
                setFxRates(rates);
            }
        } catch (error) {
            console.error('Failed to fetch countries:', error);
        } finally {
            setIsLoading(false);
        }
    }, []);

    useEffect(() => {
        if (user) {
            fetchCountries();
            // Refresh every 30 seconds for live rates
            const interval = setInterval(fetchCountries, 30000);
            return () => clearInterval(interval);
        }
    }, [user, fetchCountries]);

    // Initialize Cytoscape
    useEffect(() => {
        if (!containerRef.current || countries.length === 0) return;

        // Generate nodes from countries
        const nodes = countries.map((country, index) => {
            // Position in a circular layout with some randomization
            const angle = (index / countries.length) * 2 * Math.PI;
            const radius = 350 + (country.gdp_rank <= 10 ? 0 : country.gdp_rank <= 25 ? 50 : 100);

            return {
                data: {
                    id: country.code,
                    label: country.code,
                    name: country.name,
                    currency: country.currency,
                    successRate: country.success_rate,
                    gdpRank: country.gdp_rank,
                    fxRate: country.fx_rate || fxRates[country.currency] || 1,
                },
                position: {
                    x: 450 + radius * Math.cos(angle),
                    y: 400 + radius * Math.sin(angle),
                },
            };
        });

        // Generate edges from trade connections
        const countrySet = new Set(countries.map(c => c.code));
        const edges = TRADE_CONNECTIONS
            .filter(([src, tgt]) => countrySet.has(src) && countrySet.has(tgt))
            .map(([source, target], i) => {
                const srcCountry = countries.find(c => c.code === source);
                const tgtCountry = countries.find(c => c.code === target);

                // Calculate FX rate between the two currencies
                const srcRate = srcCountry?.fx_rate || fxRates[srcCountry?.currency || ''] || 1;
                const tgtRate = tgtCountry?.fx_rate || fxRates[tgtCountry?.currency || ''] || 1;
                const pairRate = srcRate > 0 ? tgtRate / srcRate : 1;

                return {
                    data: {
                        id: `e${i}`,
                        source,
                        target,
                        rate: pairRate.toFixed(4),
                        srcCurrency: srcCountry?.currency,
                        tgtCurrency: tgtCountry?.currency,
                    },
                };
            });

        if (cyRef.current) {
            cyRef.current.destroy();
        }

        cyRef.current = cytoscape({
            container: containerRef.current,
            elements: [...nodes, ...edges],
            style: [
                {
                    selector: 'node',
                    style: {
                        'background-color': (ele: cytoscape.NodeSingular) => {
                            const rank = ele.data('gdpRank');
                            if (rank <= 5) return '#10b981'; // Top 5 - emerald
                            if (rank <= 15) return '#3b82f6'; // Top 15 - blue
                            if (rank <= 30) return '#8b5cf6'; // Top 30 - purple
                            return '#6b7280'; // Rest - gray
                        },
                        'label': 'data(label)',
                        'color': '#fff',
                        'text-valign': 'center',
                        'text-halign': 'center',
                        'font-size': '10px',
                        'font-weight': 'bold',
                        'width': (ele: cytoscape.NodeSingular) => {
                            const rank = ele.data('gdpRank');
                            return Math.max(30, 60 - rank);
                        },
                        'height': (ele: cytoscape.NodeSingular) => {
                            const rank = ele.data('gdpRank');
                            return Math.max(30, 60 - rank);
                        },
                        'border-width': 2,
                        'border-color': '#1e293b',
                    },
                },
                {
                    selector: 'node:selected',
                    style: {
                        'border-width': 4,
                        'border-color': '#fbbf24',
                        'background-color': '#fbbf24',
                    },
                },
                {
                    selector: 'edge',
                    style: {
                        'width': 1.5,
                        'line-color': '#4b5563',
                        'curve-style': 'bezier',
                        'opacity': 0.6,
                    },
                },
                {
                    selector: 'edge:selected',
                    style: {
                        'width': 3,
                        'line-color': '#10b981',
                        'opacity': 1,
                    },
                },
            ],
            layout: {
                name: 'preset',
            },
            minZoom: 0.3,
            maxZoom: 3,
        });

        // Node click handler
        cyRef.current.on('tap', 'node', (evt) => {
            const node = evt.target;
            const countryData = countries.find(c => c.code === node.id());
            if (countryData) {
                setSelectedCountry(countryData);
            }
        });

        // Background click to deselect
        cyRef.current.on('tap', (evt) => {
            if (evt.target === cyRef.current) {
                setSelectedCountry(null);
            }
        });

        return () => {
            if (cyRef.current) {
                cyRef.current.destroy();
            }
        };
    }, [countries, fxRates]);

    if (authLoading) {
        return <div className="min-h-screen flex items-center justify-center text-slate-400">Loading...</div>;
    }

    if (!user) {
        return (
            <div className="min-h-screen flex items-center justify-center">
                <div className="text-center">
                    <p className="text-slate-400 mb-4">Please login to view the FX graph</p>
                    <Link href="/login" className="px-4 py-2 bg-emerald-500 text-white rounded-lg">Login</Link>
                </div>
            </div>
        );
    }

    return (
        <div className="min-h-screen">
            {/* Header */}
            <header className="bg-slate-900/80 backdrop-blur-lg border-b border-white/10 sticky top-0 z-50">
                <div className="max-w-7xl mx-auto px-6 py-4 flex items-center justify-between">
                    <div className="flex items-center gap-4">
                        <Link href="/" className="text-slate-400 hover:text-white transition-colors">← Back</Link>
                        <h1 className="text-xl font-bold text-white">🌍 Global FX Rate Network</h1>
                    </div>
                    <div className="flex items-center gap-4">
                        <span className="text-sm text-slate-400">
                            {countries.length} Countries | {TRADE_CONNECTIONS.length} Trading Pairs
                        </span>
                        {isAdmin && (
                            <Link href="/admin/countries" className="px-3 py-1 bg-red-500/20 text-red-400 rounded text-sm">
                                Manage
                            </Link>
                        )}
                    </div>
                </div>
            </header>

            <main className="flex">
                {/* Graph Container */}
                <div className="flex-1 relative">
                    {isLoading ? (
                        <div className="absolute inset-0 flex items-center justify-center text-slate-400">
                            Loading FX network...
                        </div>
                    ) : (
                        <div
                            ref={containerRef}
                            className="w-full h-[calc(100vh-80px)] bg-slate-950"
                        />
                    )}

                    {/* Legend */}
                    <div className="absolute bottom-4 left-4 bg-slate-800/90 backdrop-blur p-4 rounded-xl border border-white/10">
                        <h3 className="text-xs uppercase text-slate-500 mb-3">GDP Ranking</h3>
                        <div className="space-y-2 text-sm">
                            <div className="flex items-center gap-2">
                                <span className="w-4 h-4 rounded-full bg-emerald-500" />
                                <span className="text-slate-300">Top 5</span>
                            </div>
                            <div className="flex items-center gap-2">
                                <span className="w-4 h-4 rounded-full bg-blue-500" />
                                <span className="text-slate-300">Top 6-15</span>
                            </div>
                            <div className="flex items-center gap-2">
                                <span className="w-4 h-4 rounded-full bg-purple-500" />
                                <span className="text-slate-300">Top 16-30</span>
                            </div>
                            <div className="flex items-center gap-2">
                                <span className="w-4 h-4 rounded-full bg-gray-500" />
                                <span className="text-slate-300">31-50</span>
                            </div>
                        </div>
                    </div>
                </div>

                {/* Side Panel */}
                <div className="w-80 bg-slate-900 border-l border-white/10 p-4 overflow-y-auto h-[calc(100vh-80px)]">
                    {selectedCountry ? (
                        <div className="space-y-4">
                            <div className="flex items-center justify-between">
                                <h2 className="text-xl font-bold text-white">{selectedCountry.name}</h2>
                                <span className="text-2xl">{getFlagEmoji(selectedCountry.code)}</span>
                            </div>

                            <div className="bg-slate-800/50 rounded-xl p-4 space-y-3">
                                <div className="flex justify-between">
                                    <span className="text-slate-400">Code</span>
                                    <span className="font-mono text-cyan-400">{selectedCountry.code}</span>
                                </div>
                                <div className="flex justify-between">
                                    <span className="text-slate-400">Currency</span>
                                    <span className="font-mono text-white">{selectedCountry.currency}</span>
                                </div>
                                <div className="flex justify-between">
                                    <span className="text-slate-400">GDP Rank</span>
                                    <span className="text-white">#{selectedCountry.gdp_rank}</span>
                                </div>
                                <div className="flex justify-between">
                                    <span className="text-slate-400">FX Rate (to USD)</span>
                                    <span className={`font-mono ${selectedCountry.fx_rate ? 'text-emerald-400' : 'text-slate-500'}`}>
                                        {selectedCountry.fx_rate?.toFixed(4) || 'N/A'}
                                    </span>
                                </div>
                                <div className="flex justify-between">
                                    <span className="text-slate-400">Success Rate</span>
                                    <span className={`font-semibold ${selectedCountry.success_rate >= 0.9 ? 'text-emerald-400' :
                                        selectedCountry.success_rate >= 0.8 ? 'text-amber-400' : 'text-red-400'
                                        }`}>
                                        {(selectedCountry.success_rate * 100).toFixed(0)}%
                                    </span>
                                </div>
                                <div className="flex justify-between">
                                    <span className="text-slate-400">Base Credibility</span>
                                    <span className="text-white">{selectedCountry.base_credibility}</span>
                                </div>
                            </div>

                            {/* Trading Partners */}
                            <div>
                                <h3 className="text-sm uppercase text-slate-500 mb-2">Trading Partners</h3>
                                <div className="flex flex-wrap gap-2">
                                    {TRADE_CONNECTIONS
                                        .filter(([a, b]) => a === selectedCountry.code || b === selectedCountry.code)
                                        .map(([a, b]) => {
                                            const partner = a === selectedCountry.code ? b : a;
                                            return (
                                                <span
                                                    key={partner}
                                                    className="px-2 py-1 bg-slate-700 rounded text-xs text-slate-300"
                                                >
                                                    {partner}
                                                </span>
                                            );
                                        })}
                                </div>
                            </div>
                        </div>
                    ) : (
                        <div className="text-center text-slate-500 mt-20">
                            <p className="text-4xl mb-4">🌐</p>
                            <p>Click on a country node to see details</p>
                        </div>
                    )}

                    {/* FX Rates List */}
                    <div className="mt-6">
                        <h3 className="text-sm uppercase text-slate-500 mb-3">Live FX Rates (USD Base)</h3>
                        <div className="space-y-1 max-h-60 overflow-y-auto">
                            {countries
                                .filter(c => c.fx_rate && c.currency !== 'USD')
                                .sort((a, b) => (a.fx_rate || 0) - (b.fx_rate || 0))
                                .slice(0, 15)
                                .map(c => (
                                    <div key={c.code} className="flex justify-between text-sm py-1 border-b border-white/5">
                                        <span className="text-slate-400">{c.currency}</span>
                                        <span className="font-mono text-emerald-400">{c.fx_rate?.toFixed(4)}</span>
                                    </div>
                                ))}
                            {countries.filter(c => c.fx_rate).length === 0 && (
                                <p className="text-slate-500 text-xs">
                                    FX rates will appear when API key is configured
                                </p>
                            )}
                        </div>
                    </div>
                </div>
            </main>
        </div>
    );
}

// Helper to get flag emoji from country code
function getFlagEmoji(countryCode: string): string {
    const codeMap: Record<string, string> = {
        USA: 'US', CHN: 'CN', DEU: 'DE', JPN: 'JP', IND: 'IN', GBR: 'GB', FRA: 'FR',
        ITA: 'IT', BRA: 'BR', CAN: 'CA', RUS: 'RU', KOR: 'KR', AUS: 'AU', MEX: 'MX',
        ESP: 'ES', IDN: 'ID', NLD: 'NL', SAU: 'SA', TUR: 'TR', CHE: 'CH', POL: 'PL',
        TWN: 'TW', BEL: 'BE', SWE: 'SE', IRL: 'IE', AUT: 'AT', THA: 'TH', ISR: 'IL',
        NGA: 'NG', ARE: 'AE', ARG: 'AR', NOR: 'NO', EGY: 'EG', VNM: 'VN', BGD: 'BD',
        ZAF: 'ZA', PHL: 'PH', DNK: 'DK', MYS: 'MY', SGP: 'SG', HKG: 'HK', PAK: 'PK',
        CHL: 'CL', COL: 'CO', FIN: 'FI', CZE: 'CZ', ROU: 'RO', PRT: 'PT', NZL: 'NZ',
        PER: 'PE',
    };
    const code = codeMap[countryCode] || countryCode.slice(0, 2);
    const codePoints = [...code.toUpperCase()].map(c => 0x1F1E6 + c.charCodeAt(0) - 65);
    return String.fromCodePoint(...codePoints);
}
