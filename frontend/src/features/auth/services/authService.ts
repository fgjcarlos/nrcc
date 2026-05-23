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

let accessToken: string | null = null;

registerTokenAccessors(
  () => accessToken,
  (token: string) => { accessToken = token; },
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
    return { token, user };
  },

  login: async (username: string, password: string): Promise<AuthResponse> => {
    const response = await api.post<{ data: AuthResponse }>('/auth/login', {
      username,
      password,
    });
    const { token, user } = response.data.data;
    accessToken = token;
    return { token, user };
  },

  logout: async () => {
    accessToken = null;
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
