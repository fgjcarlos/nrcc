import axios, { type AxiosError, type InternalAxiosRequestConfig } from 'axios';
import type { ApiResponse } from 'shared/types';
import { redirectToLogin } from './navigation';

// All requests go out with `withCredentials: true` so the httpOnly
// `nrcc_refresh` cookie (set on /auth/login and /auth/setup) is sent on
// every call, including the page-load rehydrate via /auth/refresh from
// useAuth. Without this, a full page reload (F5) on a protected route
// drops the session because the access token lives only in memory and
// the refresh cookie never reaches the server — see issue #362.
const api = axios.create({
  baseURL: '/api',
  timeout: 10000,
  headers: {
    'Content-Type': 'application/json',
  },
  withCredentials: true,
});

let tokenGetter: (() => string | null) | null = null;
let tokenSetter: ((token: string) => void) | null = null;
let refreshPromise: Promise<string | null> | null = null;

export function registerTokenAccessors(
  getter: () => string | null,
  setter: (token: string) => void,
) {
  tokenGetter = getter;
  tokenSetter = setter;
}

async function refreshAccessToken(): Promise<string | null> {
  try {
    // Use the shared `api` instance so the request picks up the global
    // withCredentials + baseURL config. A previous version used a bare
    // `axios.post` with explicit options; consolidating on `api` keeps
    // the cookie/transport behaviour consistent across refresh paths.
    const response = await api.post<{ data: { token: string } }>('/auth/refresh', null);
    const token = response.data.data.token;
    tokenSetter?.(token);
    return token;
  } catch {
    return null;
  }
}

api.interceptors.request.use(
  (config) => {
    const token = tokenGetter?.();
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  },
  (error) => Promise.reject(error)
);

api.interceptors.response.use(
  (response) => response,
  async (error: AxiosError<ApiResponse<unknown>>) => {
    const originalRequest = error.config as InternalAxiosRequestConfig & { _retry?: boolean };

    if (error.response?.status === 401 && !originalRequest._retry) {
      const url = originalRequest.url ?? '';
      if (url.includes('/auth/login') || url.includes('/auth/setup') || url.includes('/auth/refresh')) {
        return Promise.reject(error);
      }

      originalRequest._retry = true;

      if (!refreshPromise) {
        refreshPromise = refreshAccessToken().finally(() => {
          refreshPromise = null;
        });
      }

      const newToken = await refreshPromise;
      if (newToken) {
        originalRequest.headers.Authorization = `Bearer ${newToken}`;
        return api(originalRequest);
      }

      if (!window.location.pathname.includes('/login') &&
          !window.location.pathname.includes('/setup')) {
        // Navigate via React Router (registered by the app) instead of a full
        // document reload, preserving SPA state/history.
        redirectToLogin();
      }
    }

    if (error.response) {
      console.error('API Error:', error.response.data);
    } else if (error.request) {
      console.error('Network Error:', error.message);
    } else {
      console.error('Error:', error.message);
    }
    return Promise.reject(error);
  }
);

export { api };
export default api;
