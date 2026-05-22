import { vi } from 'vitest';
import type { useAuth } from '../hooks/useAuth';
import type { AuthResponse, User } from '../services/authService';

type AuthContextValue = ReturnType<typeof useAuth>;

export function buildUserMock(overrides: Partial<User> = {}): User {
  return {
    id: 'mock-user-id',
    username: 'mock-user',
    role: 'admin',
    createdAt: '2024-01-01T00:00:00.000Z',
    ...overrides,
  };
}

export function buildAuthResponseMock(overrides: Partial<AuthResponse> = {}): AuthResponse {
  return {
    token: 'mock-token',
    user: buildUserMock(),
    ...overrides,
  };
}

export function buildAuthMock(overrides: Partial<AuthContextValue> = {}): AuthContextValue {
  return {
    isAuthenticated: false,
    isInitialized: false,
    isLoading: false,
    user: null,
    login: vi.fn().mockResolvedValue(buildAuthResponseMock()),
    logout: vi.fn(),
    checkAuth: vi.fn().mockResolvedValue(undefined),
    ...overrides,
  };
}
