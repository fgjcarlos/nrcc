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

// Auth-bootstrap gate. The first page load arms a single
// page-scoped promise; every non-auth request awaits it before
// going out, so a TanStack query fired during the first render
// does not race the rehydrate call and produce a spurious 401
// that would otherwise send the user back to /login — see
// issue #517.
//
// The gate is single-shot: arm once, release once. Subsequent
// useAuth mounts in the same page load (e.g. on a protected
// route mounted after login) do NOT re-arm and do NOT touch the
// gate — they just see whatever state the singleton is in. If
// the gate is already resolved, requests pass through. If it is
// still pending, requests wait for the original release.
//
// A safety timeout releases the gate after 2s even if the
// owning useAuth never settles (e.g. a misconfigured mock in
// tests, or a wedged request interceptor). Without this, a
// single failure path could deadlock the page. 2s is well
// above the legitimate rehydrate window (cookie round-trip to
// /auth/refresh + /auth/me) and well below the E2E test
// timeout, so it never trips in working paths.
let bootstrapGate: Promise<void> = Promise.resolve();
let bootstrapGateResolve: (() => void) | null = null;
let bootstrapArmed = false;
let bootstrapTimeoutHandle: ReturnType<typeof setTimeout> | null = null;

const GATE_TIMEOUT_MS = 2000;

export function armAuthBootstrap(): void {
  if (bootstrapArmed) return;
  bootstrapArmed = true;
  bootstrapGate = new Promise<void>((resolve) => {
    bootstrapGateResolve = resolve;
  });
  // ponytail: 2s ceiling on a global gate. Add a per-test mock that
  // resolves releaseAuthBootstrap() before this fires if the test
  // path needs the gate held longer.
  bootstrapTimeoutHandle = setTimeout(() => {
    releaseAuthBootstrap();
  }, GATE_TIMEOUT_MS);
}

export function releaseAuthBootstrap(): void {
  if (!bootstrapArmed) return;
  bootstrapArmed = false;
  if (bootstrapTimeoutHandle !== null) {
    clearTimeout(bootstrapTimeoutHandle);
    bootstrapTimeoutHandle = null;
  }
  const r = bootstrapGateResolve;
  bootstrapGateResolve = null;
  r?.();
}

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
  async (config) => {
    const url = config.url ?? '';
    // Auth bootstrap requests (login/setup/refresh/me/status) and
    // calls that happen before useAuth mounts must not be gated —
    // they are the bootstrap itself. Every other call waits for
    // the gate. Note /auth/me is on the whitelist because checkAuth
    // calls it as the last step of the rehydrate — gating it would
    // deadlock the gate the rehydrate is supposed to release.
    if (
      !url.includes('/auth/login') &&
      !url.includes('/auth/setup') &&
      !url.includes('/auth/refresh') &&
      !url.includes('/auth/me') &&
      !url.includes('/auth/status')
    ) {
      await bootstrapGate;
    }
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
