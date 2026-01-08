'use client';

import { createContext, useContext, useEffect, useState, useCallback, ReactNode, useRef } from 'react';

export interface FXRate {
    currency: string;
    rate: number;
    timestamp: number;
}

interface WebSocketContextType {
    isConnected: boolean;
    fxRates: Map<string, FXRate>;
    getFXRate: (currency: string) => number | null;
    lastUpdate: number | null;
}

const WebSocketContext = createContext<WebSocketContextType | null>(null);

export function WebSocketProvider({ children }: { children: ReactNode }) {
    const [isConnected, setIsConnected] = useState(false);
    const [fxRates, setFxRates] = useState<Map<string, FXRate>>(new Map());
    const [lastUpdate, setLastUpdate] = useState<number | null>(null);
    const wsRef = useRef<WebSocket | null>(null);
    const reconnectTimeoutRef = useRef<NodeJS.Timeout | null>(null);

    const connect = useCallback(() => {
        if (wsRef.current?.readyState === WebSocket.OPEN) return;

        try {
            const ws = new WebSocket('ws://localhost:8080/ws');
            wsRef.current = ws;

            ws.onopen = () => {
                setIsConnected(true);
                console.log('🔌 WebSocket connected');

                // Subscribe to FX updates
                ws.send(JSON.stringify({ type: 'subscribe', channel: 'fx_rates' }));
            };

            ws.onmessage = (event) => {
                try {
                    const message = JSON.parse(event.data);

                    if (message.type === 'fx_update' && message.rates) {
                        const newRates = new Map<string, FXRate>();
                        const timestamp = Date.now();

                        Object.entries(message.rates).forEach(([currency, rate]) => {
                            newRates.set(currency, {
                                currency,
                                rate: rate as number,
                                timestamp,
                            });
                        });

                        setFxRates(newRates);
                        setLastUpdate(timestamp);
                    }

                    // Handle initial state with countries
                    if (message.type === 'initial_state' && message.countries) {
                        const newRates = new Map<string, FXRate>();
                        const timestamp = Date.now();

                        message.countries.forEach((country: { currency: string; fx_rate: number }) => {
                            if (country.fx_rate) {
                                newRates.set(country.currency, {
                                    currency: country.currency,
                                    rate: country.fx_rate,
                                    timestamp,
                                });
                            }
                        });

                        setFxRates(newRates);
                        setLastUpdate(timestamp);
                    }
                } catch {
                    // Ignore parse errors
                }
            };

            ws.onclose = () => {
                setIsConnected(false);
                console.log('🔌 WebSocket disconnected');

                // Reconnect after 3 seconds
                reconnectTimeoutRef.current = setTimeout(connect, 3000);
            };

            ws.onerror = () => {
                setIsConnected(false);
            };
        } catch (error) {
            console.error('WebSocket connection error:', error);
        }
    }, []);

    useEffect(() => {
        connect();

        return () => {
            if (reconnectTimeoutRef.current) {
                clearTimeout(reconnectTimeoutRef.current);
            }
            if (wsRef.current) {
                wsRef.current.close();
            }
        };
    }, [connect]);

    const getFXRate = useCallback((currency: string): number | null => {
        return fxRates.get(currency)?.rate ?? null;
    }, [fxRates]);

    return (
        <WebSocketContext.Provider
            value={{
                isConnected,
                fxRates,
                getFXRate,
                lastUpdate,
            }}
        >
            {children}
        </WebSocketContext.Provider>
    );
}

export function useWebSocket() {
    const context = useContext(WebSocketContext);
    if (!context) {
        throw new Error('useWebSocket must be used within WebSocketProvider');
    }
    return context;
}
