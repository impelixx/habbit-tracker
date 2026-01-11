import { useState, useEffect } from 'react';
import { useTelegram } from './useTelegram';
import { api } from '../services/api';
import type { User } from '../types';

export const useAuth = () => {
  const { initData, isReady } = useTelegram();
  const [user, setUser] = useState<User | null>(null);
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const authenticate = async () => {
      try {
        setIsLoading(true);
        setError(null);

        // Check if we have a stored token
        const existingToken = api.getToken();
        if (existingToken) {
          // Token exists, try to fetch user data
          // For now, we'll just mark as authenticated
          // In a real scenario, you'd validate the token with the backend
          setIsAuthenticated(true);
          setIsLoading(false);
          return;
        }

        // If running in Telegram, use initData to authenticate
        if (initData) {
          const authResponse = await api.verifyAuth(initData);
          setUser(authResponse.user);
          setIsAuthenticated(true);
        } else {
          // Development mode without Telegram
          console.warn('No initData available. Authentication skipped.');
          setIsAuthenticated(false);
        }
      } catch (err) {
        console.error('Authentication error:', err);
        setError(err instanceof Error ? err.message : 'Authentication failed');
        setIsAuthenticated(false);
      } finally {
        setIsLoading(false);
      }
    };

    if (isReady) {
      authenticate();
    }
  }, [initData, isReady]);

  const logout = () => {
    api.clearToken();
    setUser(null);
    setIsAuthenticated(false);
  };

  return {
    user,
    isAuthenticated,
    isLoading,
    error,
    logout,
  };
};
