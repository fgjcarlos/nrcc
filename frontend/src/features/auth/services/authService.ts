import { api, registerTokenAccessors } from '@/shared/lib/api';
import type { Schemas } from '@/shared/api';

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

let accessToken: string | null = null;

function setToken(token: string | null): void {
  accessToken = token;
}

async function logout(): Promise<void> {
  const token = accessToken;
  setToken(null);
  try {
    if (token) {
      await fetch('/api/auth/logout', {
        method: 'POST',
        headers: { Authorization: `Bearer ${token}` },
        credentials: 'include',
      });
    }
  } finally {
    setToken(null);
  }
}

export const authService = {
  getStatus: async (): Promise<AuthStatus> => {
    const response = await api.get<{ data: AuthStatus }>('/auth/status');
    return response.data.data;
  },

  setup: async (username: string, password: string, setupToken: string): Promise<AuthResponse> => {
    const response = await api.post<{ data: AuthResponse }>('/auth/setup', {
      username,
      password,
      confirmPassword: password,
    }, {
      headers: { 'X-Setup-Reset-Token': setupToken },
    });
    const { token, user } = response.data.data;
    setToken(token);
    return { token, user };
  },

  login: async (username: string, password: string): Promise<AuthResponse> => {
    const response = await api.post<{ data: AuthResponse }>('/auth/login', {
      username,
      password,
    });
    const { token, user } = response.data.data;
    setToken(token);
    return { token, user };
  },

  logout,

  getToken: (): string | null => {
    return accessToken;
  },

  setToken,

  getMe: async (): Promise<User> => {
    const response = await api.get<{ data: User }>('/auth/me');
    return response.data.data;
  },

  getUsers: async (): Promise<User[]> => {
    const response = await api.get<Schemas['SuccessEnvelope_UserListResponse']>('/auth/users');
    return response.data.data?.users ?? [];
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

  // currentPassword is required by the backend when users change their own password.
  changePassword: async (id: string, password: string, currentPassword?: string): Promise<void> => {
    const response = await api.patch<Schemas['SuccessEnvelope_CredentialChange']>(
      `/auth/users/${id}/password`, currentPassword ? { password, currentPassword } : { password },
    );
    const token = response.data.data?.token;
    if (token) setToken(token);
  },

  updateUserRole: async (id: string, role: 'admin' | 'viewer'): Promise<User> => {
    const response = await api.patch<{ data: User }>(`/auth/users/${id}`, { role });
    return response.data.data;
  },
};

registerTokenAccessors(
  () => accessToken,
  (token: string) => authService.setToken(token),
);
