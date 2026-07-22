import { api, registerTokenAccessors } from '@/shared/lib/api';

export interface User {
  id: string;
  username: string;
  role: 'admin' | 'viewer';
  createdAt: string;
  updatedAt?: string;
}

export interface AuthResponse {
  token: string;
  user: User;
}

export interface AuthStatus {
  initialized: boolean;
}

// sessionStorage key for the access token. The access token lives
// in sessionStorage (cleared on tab close, survives F5 in the same
// tab) so a full-page reload rehydrates the session without having
// to round-trip /auth/refresh. The refresh cookie path stays in
// place as a fallback for the case where the in-tab token has
// expired (token lifetime is short — see service.AccessTokenLifetime).
const ACCESS_TOKEN_KEY = 'nrcc_access_token';

function loadAccessToken(): string | null {
  try {
    return sessionStorage.getItem(ACCESS_TOKEN_KEY);
  } catch {
    // sessionStorage can throw in private-browsing / locked-down
    // environments; fall back to module-only state.
    return null;
  }
}

function saveAccessToken(token: string | null): void {
  try {
    if (token === null) {
      sessionStorage.removeItem(ACCESS_TOKEN_KEY);
    } else {
      sessionStorage.setItem(ACCESS_TOKEN_KEY, token);
    }
  } catch {
    // ignore — token is still in module memory for the current
    // page session.
  }
}

let accessToken: string | null = loadAccessToken();

registerTokenAccessors(
  () => accessToken,
  (token: string) => {
    accessToken = token;
    saveAccessToken(token);
  },
);

export const authService = {
  getStatus: async (): Promise<AuthStatus> => {
    const response = await api.get<{ data: AuthStatus }>('/auth/status');
    return response.data.data;
  },

  setup: async (username: string, password: string): Promise<AuthResponse> => {
    const response = await api.post<{ data: AuthResponse }>('/auth/setup', {
      username,
      password,
      confirmPassword: password,
    });
    const { token, user } = response.data.data;
    accessToken = token;
    saveAccessToken(token);
    return { token, user };
  },

  login: async (username: string, password: string): Promise<AuthResponse> => {
    const response = await api.post<{ data: AuthResponse }>('/auth/login', {
      username,
      password,
    });
    const { token, user } = response.data.data;
    accessToken = token;
    saveAccessToken(token);
    return { token, user };
  },

  logout: async () => {
    accessToken = null;
    saveAccessToken(null);
    try {
      await api.post('/auth/logout');
    } catch {
      // best-effort server-side revocation
    }
  },

  getToken: (): string | null => {
    return accessToken;
  },

  setToken: (token: string) => {
    accessToken = token;
    saveAccessToken(token);
  },

  getMe: async (): Promise<User> => {
    const response = await api.get<{ data: User }>('/auth/me');
    return response.data.data;
  },

  getUsers: async (): Promise<User[]> => {
    const response = await api.get<{ data: User[] }>('/auth/users');
    return response.data.data;
  },

  createUser: async (username: string, password: string, role: 'admin' | 'viewer'): Promise<User> => {
    const response = await api.post<{ data: User }>('/auth/users', {
      username,
      password,
      role,
    });
    return response.data.data;
  },

  deleteUser: async (id: string): Promise<void> => {
    await api.delete(`/auth/users/${id}`);
  },

  changePassword: async (id: string, password: string): Promise<void> => {
    await api.patch(`/auth/users/${id}/password`, { password });
  },

  updateUserRole: async (id: string, role: 'admin' | 'viewer'): Promise<User> => {
    const response = await api.patch<{ data: User }>(`/auth/users/${id}`, { role });
    return response.data.data;
  },
};
