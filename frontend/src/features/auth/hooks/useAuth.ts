import { useState, useEffect, useCallback } from 'react';
import { authService, type User } from '../services/authService';

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
      const token = authService.getToken();
      if (!token) {
        setState({
          isAuthenticated: false,
          isInitialized: true,
          isLoading: false,
          user: null,
        });
        return;
      }

      const user = await authService.getMe();
      setState({
        isAuthenticated: true,
        isInitialized: true,
        isLoading: false,
        user,
      });
    } catch {
      authService.logout();
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

  const logout = useCallback(() => {
    authService.logout();
    setState({
      isAuthenticated: false,
      isInitialized: true,
      isLoading: false,
      user: null,
    });
  }, []);

  useEffect(() => {
    checkAuth();
  }, [checkAuth]);

  return {
    ...state,
    login,
    logout,
    checkAuth,
  };
}
