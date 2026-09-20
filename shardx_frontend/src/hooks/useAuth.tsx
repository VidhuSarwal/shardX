import { createContext, useContext, useState, useEffect, ReactNode } from 'react';
import { getAuthToken, clearAuthToken } from '@/lib/api';

interface AuthContextType {
  isAuthenticated: boolean;
  logout: () => void;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export const AuthProvider = ({ children }: { children: ReactNode }) => {
  const [isAuthenticated, setIsAuthenticated] = useState<boolean>(false);

  useEffect(() => {
    const sync = () => setIsAuthenticated(!!getAuthToken());
    sync();
    // Listen to custom auth change and storage events (multi-tab support)
    window.addEventListener('auth:changed', sync);
    window.addEventListener('storage', sync);
    return () => {
      window.removeEventListener('auth:changed', sync);
      window.removeEventListener('storage', sync);
    };
  }, []);

  const logout = () => {
    clearAuthToken();
    setIsAuthenticated(false);
    window.location.href = '/login';
  };

  return (
    <AuthContext.Provider value={{ isAuthenticated, logout }}>
      {children}
    </AuthContext.Provider>
  );
};

export const useAuth = () => {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error('useAuth must be used within AuthProvider');
  }
  return context;
};
