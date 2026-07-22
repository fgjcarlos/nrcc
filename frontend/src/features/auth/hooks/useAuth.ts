import { useState, useEffect, useCallback } from 'react';
import { authService, type User } from '../services/authService';
import { api, armAuthBootstrap, releaseAuthBootstrap } from '@/shared/lib';

interface AuthState {
  isAuthenticated: boolean;
  isInitialized: boolean;
  isLoading: boolean;
  user: User | null;
}

export function useAuth() {
  const [state, setState] = useState<AuthState>({
    isAuthenticated: false,
    isInitialized: false,
    isLoading: true,
    user: null,
  });

  const checkAuth = useCallback(async () => {
    try {
      if (!authService.getToken()) {
        const refreshRes = await api.post<{ data: { token: string } }>('/auth/refresh', null);
        authService.setToken(refreshRes.data.data.token);
      }

      const user = await authService.getMe();
      setState({
        isAuthenticated: true,
        isInitialized: true,
        isLoading: false,
        user,
      });
    } catch {
      setState({
        isAuthenticated: false,
        isInitialized: true,
        isLoading: false,
        user: null,
      });
    }
  }, []);

  const login = useCallback(async (username: string, password: string) => {
    const response = await authService.login(username, password);
    setState({
      isAuthenticated: true,
      isInitialized: true,
      isLoading: false,
      user: response.user,
    });
    return response;
  }, []);

  const logout = useCallback(async () => {
    await authService.logout();
    setState({
      isAuthenticated: false,
      isInitialized: true,
      isLoading: false,
      user: null,
    });
  }, []);

  useEffect(() => {
    // Each useAuth mount arms the gate (refcount) and releases it
    // when its checkAuth settles. The gate itself is a singleton
    // in the api module: a second mount in the same page load does
    // not create a fresh pending gate — it just bumps the refcount
    // and lets the existing one stay armed until both releases
    // have run.
    armAuthBootstrap();
    checkAuth().finally(releaseAuthBootstrap);
  }, [checkAuth]);

  return {
    ...state,
    login,
    logout,
    checkAuth,
  };
}
