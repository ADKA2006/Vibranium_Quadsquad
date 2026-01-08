'use client';

import { useEffect, useState, useCallback, useMemo, useRef } from 'react';
import dynamic from 'next/dynamic';
import { useAuth } from '@/lib/auth-context';
import { useBlockedCountries } from '@/lib/blocked-countries-context';
import { useWebSocket } from '@/lib/websocket-context';
import { COUNTRY_COORDINATES, TRADE_CONNECTIONS, getFlagEmoji, CountryGeo } from '@/lib/country-data';
import { auth, API_BASE_URL } from '@/lib/auth';
import Link from 'next/link';

// Dynamic import for Leaflet map (SSR disabled)
const MapContainer = dynamic(
    () => import('react-leaflet').then(mod => mod.MapContainer),
    { ssr: false, loading: () => <div className="w-full h-full bg-slate-900 flex items-center justify-center text-slate-400">Loading map...</div> }
);
const TileLayer = dynamic(
    () => import('react-leaflet').then(mod => mod.TileLayer),
    { ssr: false }
);
const CircleMarker = dynamic(
    () => import('react-leaflet').then(mod => mod.CircleMarker),
    { ssr: false }
);
const Popup = dynamic(
    () => import('react-leaflet').then(mod => mod.Popup),
    { ssr: false }
);
const Polyline = dynamic(
    () => import('react-leaflet').then(mod => mod.Polyline),
    { ssr: false }
);

interface Country {
    code: string;
    name: string;
    currency: string;
    fx_rate?: number;
    success_rate: number;
    gdp_rank: number;
}

interface RoutePathInfo {
    rank: number;
    nodes: string[];
    hop_count: number;
    total_weight: number;
    total_fee_percent: number;
    final_amount: number;
}

type SelectionMode = 'none' | 'start' | 'end';

export default function Dashboard() {
    const { user, isLoading: authLoading } = useAuth();
    const { blockedCountries, isBlocked, toggleBlocked, blockedCount, canAddMore, clearBlocked } = useBlockedCountries();
    const { isConnected, getFXRate } = useWebSocket();
    const [mapReady, setMapReady] = useState(false);

    const [countries, setCountries] = useState<Country[]>([]);
    const [isLoading, setIsLoading] = useState(true);

    // Route selection state
    const [startNode, setStartNode] = useState<string | null>(null);
    const [endNode, setEndNode] = useState<string | null>(null);
    const [selectionMode, setSelectionMode] = useState<SelectionMode>('none');

    // Route calculation state
    const [routes, setRoutes] = useState<RoutePathInfo[]>([]);
    const [selectedRouteIndex, setSelectedRouteIndex] = useState(0);
    const [isCalculating, setIsCalculating] = useState(false);
    const [routeError, setRouteError] = useState<string | null>(null);
    const routeWsRef = useRef<WebSocket | null>(null);

    // Load Leaflet CSS on client side
    useEffect(() => {
        const loadCSS = async () => {
            await import('leaflet/dist/leaflet.css');
            setMapReady(true);
        };
        loadCSS();
    }, []);

    // Connect to route WebSocket
    useEffect(() => {
        const wsUrl = API_BASE_URL.replace('http', 'ws') + '/ws/route';
        const ws = new WebSocket(wsUrl);

        ws.onopen = () => {
            console.log('Route WebSocket connected');
        };

        ws.onmessage = (event) => {
            try {
                const data = JSON.parse(event.data);
                if (data.type === 'route_response') {
                    setIsCalculating(false);
                    if (data.success) {
                        setRoutes(data.paths || []);
                        setSelectedRouteIndex(0);
                        setRouteError(null);
                    } else {
                        setRouteError(data.error || 'Route calculation failed');
                        setRoutes([]);
                    }
                }
            } catch {
                console.error('Failed to parse route response');
            }
        };

        ws.onerror = () => {
            console.error('Route WebSocket error');
        };

        ws.onclose = () => {
            console.log('Route WebSocket disconnected');
        };

        routeWsRef.current = ws;

        return () => {
            ws.close();
        };
    }, []);

    // Calculate route when start/end change
    useEffect(() => {
        if (startNode && endNode && routeWsRef.current?.readyState === WebSocket.OPEN) {
            setIsCalculating(true);
            setRouteError(null);

            const request = {
                type: 'route_request',
                source: startNode,
                target: endNode,
                blocked_codes: [...blockedCountries],
                amount: 1000 // Example amount
            };

            routeWsRef.current.send(JSON.stringify(request));
        } else {
            setRoutes([]);
        }
    }, [startNode, endNode, blockedCountries]);

    // Fetch countries from API
    const fetchCountries = useCallback(async () => {
        try {
            const response = await auth.authFetch(`${API_BASE_URL}/api/v1/admin/countries`);
            if (response.ok) {
                const data = await response.json();
                setCountries(data.countries || []);
            }
        } catch {
            console.error('Failed to fetch countries');
        } finally {
            setIsLoading(false);
        }
    }, []);

    useEffect(() => {
        if (user) {
            fetchCountries();
        }
    }, [user, fetchCountries]);

    // Get FX rate for a country
    const getCountryFXRate = useCallback((currency: string): number | null => {
        const wsRate = getFXRate(currency);
        if (wsRate) return wsRate;
        const country = countries.find(c => c.currency === currency);
        return country?.fx_rate ?? null;
    }, [getFXRate, countries]);

    // Filter connections based on blocked countries
    const activeConnections = useMemo(() => {
        return TRADE_CONNECTIONS.filter(([a, b]) => !isBlocked(a) && !isBlocked(b));
    }, [isBlocked]);

    // Get country data
    const getCountryData = useCallback((code: string): Country | undefined => {
        return countries.find(c => c.code === code);
    }, [countries]);

    // Handle country selection for routing
    const handleCountryClick = (code: string) => {
        if (selectionMode === 'start') {
            setStartNode(code);
            setSelectionMode('none');
        } else if (selectionMode === 'end') {
            setEndNode(code);
            setSelectionMode('none');
        }
    };

    // Get marker color based on selection state
    const getMarkerColor = (code: string) => {
        if (code === startNode) return '#22c55e';
        if (code === endNode) return '#ef4444';
        if (isBlocked(code)) return '#6b7280';
        const data = getCountryData(code);
        if (!data) return '#8b5cf6';
        if (data.gdp_rank <= 5) return '#10b981';
        if (data.gdp_rank <= 15) return '#3b82f6';
        return '#8b5cf6';
    };

    const getMarkerBorder = (code: string) => {
        if (code === startNode) return '#16a34a';
        if (code === endNode) return '#dc2626';
        return '#1e293b';
    };

    // Get route path coordinates for the selected route
    const selectedRoutePath = useMemo(() => {
        if (routes.length === 0 || selectedRouteIndex >= routes.length) return [];

        const route = routes[selectedRouteIndex];
        const coords: [number, number][] = [];

        for (const code of route.nodes) {
            const geo = COUNTRY_COORDINATES.find(c => c.code === code);
            if (geo) {
                coords.push([geo.lat, geo.lng]);
            }
        }

        return coords;
    }, [routes, selectedRouteIndex]);

    if (authLoading) {
        return <div className="min-h-screen flex items-center justify-center bg-slate-950 text-slate-400">Loading...</div>;
    }

    if (!user) {
        return (
            <div className="min-h-screen flex items-center justify-center bg-slate-950">
                <div className="text-center">
                    <p className="text-slate-400 mb-4">Please login to access the dashboard</p>
                    <Link href="/login" className="px-6 py-3 bg-emerald-500 text-white rounded-lg font-semibold">Login</Link>
                </div>
            </div>
        );
    }

    return (
        <div className="min-h-screen bg-slate-950">
            {/* Header */}
            <header className="bg-slate-900/80 backdrop-blur-lg border-b border-white/10 sticky top-0 z-[1000]">
                <div className="max-w-full mx-auto px-6 py-3 flex items-center justify-between">
                    <div className="flex items-center gap-4">
                        <Link href="/" className="text-slate-400 hover:text-white">← Back</Link>
                        <h1 className="text-xl font-bold text-white">🌍 Global FX Dashboard</h1>
                    </div>

                    {/* Route Selection Controls */}
                    <div className="flex items-center gap-3">
                        <button
                            onClick={() => setSelectionMode(selectionMode === 'start' ? 'none' : 'start')}
                            className={`px-3 py-1.5 rounded-lg text-sm font-medium transition-colors ${selectionMode === 'start' ? 'bg-green-500 text-white animate-pulse' :
                                    startNode ? 'bg-green-500/20 text-green-400 border border-green-500/50' : 'bg-slate-700 text-slate-300'
                                }`}
                        >
                            {startNode ? `Start: ${startNode}` : '🟢 Select Start'}
                        </button>
                        <button
                            onClick={() => setSelectionMode(selectionMode === 'end' ? 'none' : 'end')}
                            className={`px-3 py-1.5 rounded-lg text-sm font-medium transition-colors ${selectionMode === 'end' ? 'bg-red-500 text-white animate-pulse' :
                                    endNode ? 'bg-red-500/20 text-red-400 border border-red-500/50' : 'bg-slate-700 text-slate-300'
                                }`}
                        >
                            {endNode ? `End: ${endNode}` : '🔴 Select End'}
                        </button>
                        {(startNode || endNode) && (
                            <button
                                onClick={() => { setStartNode(null); setEndNode(null); setRoutes([]); }}
                                className="px-2 py-1 text-slate-400 hover:text-white text-sm"
                            >
                                Clear
                            </button>
                        )}
                    </div>

                    {/* Status */}
                    <div className="flex items-center gap-4">
                        <div className="flex items-center gap-2 text-sm">
                            <span className={`w-2.5 h-2.5 rounded-full ${isConnected ? 'bg-emerald-400 animate-pulse' : 'bg-red-400'}`} />
                            <span className="text-slate-400">{isConnected ? 'Live' : 'Offline'}</span>
                        </div>
                    </div>
                </div>
            </header>

            {/* Selection Mode Banner */}
            {selectionMode !== 'none' && (
                <div className={`py-2 text-center text-sm font-medium ${selectionMode === 'start' ? 'bg-green-500/20 text-green-400' : 'bg-red-500/20 text-red-400'
                    }`}>
                    Click on a country to set as {selectionMode} node
                </div>
            )}

            <div className="flex h-[calc(100vh-60px)]">
                {/* Main Map View */}
                <div className="flex-1 relative">
                    {mapReady ? (
                        <MapContainer
                            center={[20, 0]}
                            zoom={2}
                            style={{ width: '100%', height: '100%', background: '#0f172a' }}
                            minZoom={1}
                            maxZoom={8}
                        >
                            <TileLayer
                                attribution='&copy; CARTO'
                                url="https://{s}.basemaps.cartocdn.com/dark_all/{z}/{x}/{y}{r}.png"
                            />

                            {/* Trade Connection Lines (background) */}
                            {activeConnections.map(([src, tgt], i) => {
                                const srcGeo = COUNTRY_COORDINATES.find(c => c.code === src);
                                const tgtGeo = COUNTRY_COORDINATES.find(c => c.code === tgt);
                                if (!srcGeo || !tgtGeo) return null;
                                return (
                                    <Polyline
                                        key={`line-${i}`}
                                        positions={[[srcGeo.lat, srcGeo.lng], [tgtGeo.lat, tgtGeo.lng]]}
                                        pathOptions={{ color: '#3b82f6', weight: 1, opacity: 0.15 }}
                                    />
                                );
                            })}

                            {/* Calculated Route Path */}
                            {selectedRoutePath.length > 0 && (
                                <Polyline
                                    positions={selectedRoutePath}
                                    pathOptions={{
                                        color: selectedRouteIndex === 0 ? '#22c55e' : selectedRouteIndex === 1 ? '#eab308' : '#f97316',
                                        weight: 4,
                                        opacity: 1,
                                        dashArray: '10, 5'
                                    }}
                                />
                            )}

                            {/* Country Markers */}
                            {COUNTRY_COORDINATES.map((country) => {
                                const blocked = isBlocked(country.code);
                                const fxRate = getCountryFXRate(country.currency);
                                const data = getCountryData(country.code);
                                const isInRoute = routes[selectedRouteIndex]?.nodes.includes(country.code);

                                return (
                                    <CircleMarker
                                        key={country.code}
                                        center={[country.lat, country.lng]}
                                        radius={isInRoute ? 10 : blocked ? 5 : data?.gdp_rank && data.gdp_rank <= 10 ? 8 : 6}
                                        pathOptions={{
                                            color: getMarkerBorder(country.code),
                                            fillColor: getMarkerColor(country.code),
                                            fillOpacity: blocked ? 0.3 : 0.9,
                                            weight: isInRoute ? 3 : 2,
                                        }}
                                        eventHandlers={{
                                            click: () => handleCountryClick(country.code),
                                        }}
                                    >
                                        <Popup>
                                            <div className="bg-slate-800 text-white p-3 rounded-lg min-w-[180px] -m-3">
                                                <div className="flex items-center justify-between mb-2">
                                                    <span className="text-base font-bold">{country.name}</span>
                                                    <span className="text-xl">{getFlagEmoji(country.code)}</span>
                                                </div>
                                                <div className="space-y-1 text-xs">
                                                    <div className="flex justify-between">
                                                        <span className="text-slate-400">Code</span>
                                                        <span className="font-mono text-cyan-400">{country.code}</span>
                                                    </div>
                                                    <div className="flex justify-between">
                                                        <span className="text-slate-400">Currency</span>
                                                        <span className="font-mono">{country.currency}</span>
                                                    </div>
                                                    <div className="flex justify-between">
                                                        <span className="text-slate-400">FX Rate</span>
                                                        <span className="font-mono text-emerald-400">{fxRate?.toFixed(4) || 'N/A'}</span>
                                                    </div>
                                                </div>
                                                <div className="flex gap-2 mt-2">
                                                    <button onClick={() => setStartNode(country.code)} className="flex-1 py-1 bg-green-500/30 text-green-400 rounded text-xs">Set Start</button>
                                                    <button onClick={() => setEndNode(country.code)} className="flex-1 py-1 bg-red-500/30 text-red-400 rounded text-xs">Set End</button>
                                                </div>
                                                <button
                                                    onClick={() => toggleBlocked(country.code)}
                                                    className={`w-full mt-2 py-1 rounded text-xs ${blocked ? 'bg-emerald-500 text-white' : 'bg-slate-700 text-slate-300'}`}
                                                >
                                                    {blocked ? '✓ Unblock' : '🚫 Block'}
                                                </button>
                                            </div>
                                        </Popup>
                                    </CircleMarker>
                                );
                            })}
                        </MapContainer>
                    ) : (
                        <div className="w-full h-full bg-slate-900 flex items-center justify-center">
                            <div className="text-center text-slate-400">
                                <div className="text-4xl mb-4">🌍</div>
                                <p>Loading map...</p>
                            </div>
                        </div>
                    )}

                    {/* Legend */}
                    <div className="absolute bottom-4 left-4 bg-slate-800/90 backdrop-blur p-3 rounded-lg border border-white/10 z-[500]">
                        <h4 className="text-xs text-slate-500 uppercase mb-2">Legend</h4>
                        <div className="space-y-1 text-xs text-slate-300">
                            <div className="flex items-center gap-2"><span className="w-3 h-3 rounded-full bg-green-500"></span> Start</div>
                            <div className="flex items-center gap-2"><span className="w-3 h-3 rounded-full bg-red-500"></span> End</div>
                            <div className="flex items-center gap-2"><span className="w-3 h-3 rounded-full bg-gray-500"></span> Blocked</div>
                        </div>
                    </div>
                </div>

                {/* Sidebar */}
                <div className="w-96 bg-slate-900 border-l border-white/10 p-4 overflow-y-auto">
                    {/* Top 3 Routes */}
                    <div className="mb-6">
                        <h2 className="text-lg font-bold text-white mb-3">🛤️ Top 3 Routes</h2>

                        {isCalculating && (
                            <div className="text-center py-8 text-slate-400">
                                <div className="animate-spin text-2xl mb-2">⚙️</div>
                                <p className="text-sm">Calculating optimal paths...</p>
                            </div>
                        )}

                        {routeError && (
                            <div className="bg-red-500/10 border border-red-500/30 rounded-lg p-3 text-red-400 text-sm">
                                {routeError}
                            </div>
                        )}

                        {!isCalculating && routes.length === 0 && !routeError && (
                            <div className="text-center py-8 text-slate-500">
                                <p className="text-4xl mb-2">🗺️</p>
                                <p className="text-sm">Select start and end countries</p>
                                <p className="text-xs mt-1">to calculate optimal routes</p>
                            </div>
                        )}

                        {routes.length > 0 && (
                            <div className="space-y-3">
                                {routes.map((route, index) => (
                                    <button
                                        key={index}
                                        onClick={() => setSelectedRouteIndex(index)}
                                        className={`w-full text-left p-3 rounded-xl border transition-all ${selectedRouteIndex === index
                                                ? 'bg-emerald-500/20 border-emerald-500/50'
                                                : 'bg-slate-800/50 border-white/10 hover:border-white/30'
                                            }`}
                                    >
                                        <div className="flex items-center justify-between mb-2">
                                            <span className={`text-sm font-bold ${index === 0 ? 'text-emerald-400' : index === 1 ? 'text-yellow-400' : 'text-orange-400'
                                                }`}>
                                                {index === 0 ? '🥇 Best' : index === 1 ? '🥈 2nd' : '🥉 3rd'} Route
                                            </span>
                                            <span className="text-xs text-slate-500">{route.hop_count} hops</span>
                                        </div>

                                        <div className="flex flex-wrap gap-1 mb-2">
                                            {route.nodes.map((code, i) => (
                                                <span key={i} className="inline-flex items-center">
                                                    <span className="px-1.5 py-0.5 bg-slate-700 rounded text-xs text-white">
                                                        {getFlagEmoji(code)} {code}
                                                    </span>
                                                    {i < route.nodes.length - 1 && <span className="text-slate-600 mx-1">→</span>}
                                                </span>
                                            ))}
                                        </div>

                                        <div className="flex justify-between text-xs">
                                            <span className="text-slate-400">Fee: <span className="text-red-400">{route.total_fee_percent.toFixed(4)}%</span></span>
                                            <span className="text-slate-400">After fees: <span className="text-emerald-400">{(route.final_amount * 100).toFixed(4)}%</span></span>
                                        </div>
                                    </button>
                                ))}
                            </div>
                        )}
                    </div>

                    {/* Blocked Countries */}
                    <div className="mb-6 border-t border-white/10 pt-4">
                        <div className="flex items-center justify-between mb-3">
                            <h2 className="text-sm font-bold text-white">🚫 Blocked ({blockedCount}/10)</h2>
                            {blockedCount > 0 && (
                                <button onClick={clearBlocked} className="text-xs text-slate-400 hover:text-white">Clear</button>
                            )}
                        </div>

                        {blockedCount > 0 ? (
                            <div className="flex flex-wrap gap-2">
                                {[...blockedCountries].map(code => (
                                    <button
                                        key={code}
                                        onClick={() => toggleBlocked(code)}
                                        className="flex items-center gap-1 px-2 py-1 bg-red-500/10 border border-red-500/30 rounded text-xs text-red-400 hover:bg-red-500/20"
                                    >
                                        {getFlagEmoji(code)} {code} ✕
                                    </button>
                                ))}
                            </div>
                        ) : (
                            <p className="text-slate-500 text-xs">No countries blocked</p>
                        )}
                    </div>

                    {/* Network Stats */}
                    <div className="border-t border-white/10 pt-4">
                        <h3 className="text-sm uppercase text-slate-500 mb-3">Network</h3>
                        <div className="grid grid-cols-2 gap-2 text-sm">
                            <div className="bg-slate-800/50 rounded-lg p-2 text-center">
                                <div className="text-white font-bold">{countries.length}</div>
                                <div className="text-slate-500 text-xs">Countries</div>
                            </div>
                            <div className="bg-slate-800/50 rounded-lg p-2 text-center">
                                <div className="text-emerald-400 font-bold">{activeConnections.length}</div>
                                <div className="text-slate-500 text-xs">Active Routes</div>
                            </div>
                        </div>
                    </div>

                    {/* Quick Links */}
                    <div className="border-t border-white/10 pt-4 mt-4">
                        <Link href="/graph" className="block w-full py-2 text-center bg-blue-500/20 text-blue-400 rounded-lg text-sm hover:bg-blue-500/30">
                            🔗 Open Logic View
                        </Link>
                    </div>
                </div>
            </div>
        </div>
    );
}
