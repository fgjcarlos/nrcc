import axios, { type AxiosError, type InternalAxiosRequestConfig } from 'axios';
import type { ApiResponse } from 'shared/types';

const api = axios.create({
  baseURL: '/api',
  timeout: 10000,
  headers: {
    'Content-Type': 'application/json',
  },
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
    const response = await axios.post<{ data: { token: string } }>('/api/auth/refresh', null, {
      withCredentials: true,
    });
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
        window.location.href = '/login';
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
