import { api } from '@/shared/lib';

export interface User {
  id: string;
  username: string;
  role: 'admin' | 'viewer';
  createdAt: string;
}

export interface AuthResponse {
  token: string;
  user: User;
}

export interface AuthStatus {
  initialized: boolean;
}

const AUTH_KEY = 'cc-token';

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
    localStorage.setItem(AUTH_KEY, token);
    return { token, user };
  },

  login: async (username: string, password: string): Promise<AuthResponse> => {
    const response = await api.post<{ data: AuthResponse }>('/auth/login', {
      username,
      password,
    });
    const { token, user } = response.data.data;
    localStorage.setItem(AUTH_KEY, token);
    return { token, user };
  },

  logout: () => {
    localStorage.removeItem(AUTH_KEY);
  },

  getToken: (): string | null => {
    return localStorage.getItem(AUTH_KEY);
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
};
