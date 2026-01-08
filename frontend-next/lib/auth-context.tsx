'use client';

import React, { createContext, useContext, useState, useEffect, ReactNode } from 'react';
import { auth, User, AuthResponse } from './auth';

interface AuthContextType {
    user: User | null;
    isLoading: boolean;
    login: (email: string, password: string) => Promise<AuthResponse>;
    register: (email: string, password: string, username: string) => Promise<AuthResponse>;
    logout: () => void;
    isAdmin: boolean;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export function AuthProvider({ children }: { children: ReactNode }) {
    const [user, setUser] = useState<User | null>(null);
    const [isLoading, setIsLoading] = useState(true);

    useEffect(() => {
        const storedUser = auth.getUser();
        setUser(storedUser);
        setIsLoading(false);
    }, []);

    const login = async (email: string, password: string): Promise<AuthResponse> => {
        const response = await auth.login(email, password);
        setUser(response.user);
        return response;
    };

    const register = async (email: string, password: string, username: string): Promise<AuthResponse> => {
        const response = await auth.register(email, password, username);
        setUser(response.user);
        return response;
    };

    const logout = () => {
        auth.logout();
        setUser(null);
    };

    return (
        <AuthContext.Provider
            value={{
                user,
                isLoading,
                login,
                register,
                logout,
                isAdmin: user?.role === 'ADMIN',
            }}
        >
            {children}
        </AuthContext.Provider>
    );
}

export function useAuth() {
    const context = useContext(AuthContext);
    if (context === undefined) {
        throw new Error('useAuth must be used within an AuthProvider');
    }
    return context;
}
