import React, { createContext, useContext, useState, useEffect } from 'react';
import type { User } from '../api/types';
import { authApi } from '../api/client';

interface AuthContextType {
  user: User | null;
  token: string | null;
  roles: string[];
  isAuthenticated: boolean;
  isLoading: boolean;
  login: (email: string, password: string) => Promise<void>;
  logout: () => void;
  hasRole: (...requiredRoles: string[]) => boolean;
  isAdmin: () => boolean;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export const AuthProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [user, setUser] = useState<User | null>(null);
  const [token, setToken] = useState<string | null>(null);
  const [roles, setRoles] = useState<string[]>([]);
  const [isLoading, setIsLoading] = useState(true);

  // Restore authenticated session from localStorage
  useEffect(() => {
    try {
      const storedToken = localStorage.getItem('auth_token');
      const storedUser = localStorage.getItem('auth_user');
      const storedRoles = localStorage.getItem('auth_roles');

      if (storedToken && storedUser) {
        setToken(storedToken);
        setUser(JSON.parse(storedUser));
        setRoles(storedRoles ? JSON.parse(storedRoles) : []);
      }
    } catch (err) {
      console.error('Failed to restore session from localStorage:', err);
      localStorage.removeItem('auth_token');
      localStorage.removeItem('auth_user');
      localStorage.removeItem('auth_roles');
    } finally {
      setIsLoading(false);
    }
  }, []);

  const login = async (email: string, password: string) => {
    const res = await authApi.login(email, password);
    setToken(res.token);
    setUser(res.user);
    setRoles(res.roles || []);

    localStorage.setItem('auth_token', res.token);
    localStorage.setItem('auth_user', JSON.stringify(res.user));
    localStorage.setItem('auth_roles', JSON.stringify(res.roles || []));
  };

  const logout = () => {
    setToken(null);
    setUser(null);
    setRoles([]);
    localStorage.removeItem('auth_token');
    localStorage.removeItem('auth_user');
    localStorage.removeItem('auth_roles');
  };

  const hasRole = (...requiredRoles: string[]): boolean => {
    if (!user || roles.length === 0) return false;
    // Super-admin bypasses restrictions
    if (roles.some((r) => r.toLowerCase() === 'admin')) return true;

    return requiredRoles.some((req) =>
      roles.some((userRole) => userRole.toLowerCase() === req.toLowerCase())
    );
  };

  const isAdmin = (): boolean => {
    return roles.some((r) => r.toLowerCase() === 'admin');
  };

  return (
    <AuthContext.Provider
      value={{
        user,
        token,
        roles,
        isAuthenticated: !!token && !!user,
        isLoading,
        login,
        logout,
        hasRole,
        isAdmin,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
};

export const useAuth = (): AuthContextType => {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
};
