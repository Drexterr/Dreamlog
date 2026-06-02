import React, { createContext, useContext, useState, useEffect } from 'react';
import { api } from '../services/api';
import type { Therapist } from '../types';

interface AuthContextProps {
  isAuthenticated: boolean;
  token: string | null;
  therapist: Therapist | null;
  login: (email: string, password: string) => Promise<void>;
  register: (email: string, name: string, credentials?: string) => Promise<void>;
  logout: () => void;
  loading: boolean;
}

const AuthContext = createContext<AuthContextProps | undefined>(undefined);

export const AuthProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [token, setToken] = useState<string | null>(localStorage.getItem('dreamlog_therapist_token'));
  const [therapist, setTherapist] = useState<Therapist | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    // If token exists, load profile (Mocked in our mock mode)
    if (token) {
      const storedProfile = localStorage.getItem('dreamlog_therapist_profile');
      if (storedProfile) {
        setTherapist(JSON.parse(storedProfile));
      } else {
        // Fallback default mock profile
        const mockProfile: Therapist = {
          id: 'mock-therapist-uuid-1',
          userId: 'mock-user-uuid-1',
          name: 'Dr. Sarah Jenkins',
          email: 'sjenkins@clinic.org',
          credentials: 'Clinical Psychologist, PsyD',
          createdAt: new Date().toISOString(),
        };
        setTherapist(mockProfile);
        localStorage.setItem('dreamlog_therapist_profile', JSON.stringify(mockProfile));
      }
    }
    setLoading(false);
  }, [token]);

  const login = async (email: string, password: string) => {
    setLoading(true);
    try {
      const result = await api.login(email, password);
      localStorage.setItem('dreamlog_therapist_token', result.token);
      setToken(result.token);
      
      const mockProfile: Therapist = {
        id: 'mock-therapist-uuid-1',
        userId: 'mock-user-uuid-1',
        name: 'Dr. Sarah Jenkins',
        email: email,
        credentials: 'Clinical Psychologist, PsyD',
        createdAt: new Date().toISOString(),
      };
      setTherapist(mockProfile);
      localStorage.setItem('dreamlog_therapist_profile', JSON.stringify(mockProfile));
    } finally {
      setLoading(false);
    }
  };

  const register = async (email: string, name: string, credentials?: string) => {
    setLoading(true);
    try {
      const t = await api.register(email, name, credentials);
      // Simulate auto-login on register
      const result = await api.login(email, 'default_password');
      localStorage.setItem('dreamlog_therapist_token', result.token);
      localStorage.setItem('dreamlog_therapist_profile', JSON.stringify(t));
      setToken(result.token);
      setTherapist(t);
    } finally {
      setLoading(false);
    }
  };

  const logout = () => {
    localStorage.removeItem('dreamlog_therapist_token');
    localStorage.removeItem('dreamlog_therapist_profile');
    setToken(null);
    setTherapist(null);
  };

  const isAuthenticated = !!token;

  return (
    <AuthContext.Provider value={{ isAuthenticated, token, therapist, login, register, logout, loading }}>
      {children}
    </AuthContext.Provider>
  );
};

export const useAuth = () => {
  const context = useContext(AuthContext);
  if (!context) throw new Error('useAuth must be used within an AuthProvider');
  return context;
};
