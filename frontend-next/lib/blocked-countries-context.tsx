'use client';

import { createContext, useContext, useState, useCallback, useEffect, ReactNode } from 'react';

const MAX_BLOCKED_COUNTRIES = 10;
const STORAGE_KEY = 'plm_blocked_countries';

interface BlockedCountriesContextType {
    blockedCountries: Set<string>;
    isBlocked: (code: string) => boolean;
    toggleBlocked: (code: string) => void;
    addBlocked: (code: string) => boolean;
    removeBlocked: (code: string) => void;
    clearBlocked: () => void;
    canAddMore: boolean;
    blockedCount: number;
}

const BlockedCountriesContext = createContext<BlockedCountriesContextType | null>(null);

export function BlockedCountriesProvider({ children }: { children: ReactNode }) {
    const [blockedCountries, setBlockedCountries] = useState<Set<string>>(new Set());

    // Load from localStorage on mount
    useEffect(() => {
        try {
            const stored = localStorage.getItem(STORAGE_KEY);
            if (stored) {
                const parsed = JSON.parse(stored);
                if (Array.isArray(parsed)) {
                    setBlockedCountries(new Set(parsed.slice(0, MAX_BLOCKED_COUNTRIES)));
                }
            }
        } catch {
            // Ignore errors
        }
    }, []);

    // Save to localStorage on change
    useEffect(() => {
        try {
            localStorage.setItem(STORAGE_KEY, JSON.stringify([...blockedCountries]));
        } catch {
            // Ignore errors
        }
    }, [blockedCountries]);

    const isBlocked = useCallback((code: string) => {
        return blockedCountries.has(code);
    }, [blockedCountries]);

    const addBlocked = useCallback((code: string): boolean => {
        if (blockedCountries.size >= MAX_BLOCKED_COUNTRIES) {
            return false;
        }
        setBlockedCountries(prev => new Set([...prev, code]));
        return true;
    }, [blockedCountries.size]);

    const removeBlocked = useCallback((code: string) => {
        setBlockedCountries(prev => {
            const next = new Set(prev);
            next.delete(code);
            return next;
        });
    }, []);

    const toggleBlocked = useCallback((code: string) => {
        if (blockedCountries.has(code)) {
            removeBlocked(code);
        } else {
            addBlocked(code);
        }
    }, [blockedCountries, addBlocked, removeBlocked]);

    const clearBlocked = useCallback(() => {
        setBlockedCountries(new Set());
    }, []);

    return (
        <BlockedCountriesContext.Provider
            value={{
                blockedCountries,
                isBlocked,
                toggleBlocked,
                addBlocked,
                removeBlocked,
                clearBlocked,
                canAddMore: blockedCountries.size < MAX_BLOCKED_COUNTRIES,
                blockedCount: blockedCountries.size,
            }}
        >
            {children}
        </BlockedCountriesContext.Provider>
    );
}

export function useBlockedCountries() {
    const context = useContext(BlockedCountriesContext);
    if (!context) {
        throw new Error('useBlockedCountries must be used within BlockedCountriesProvider');
    }
    return context;
}

export { MAX_BLOCKED_COUNTRIES };
