import { vi } from 'vitest';
import type { AuthUser } from '../api';

export const mockUser: AuthUser = {
  id: 'u1',
  name: 'Test User',
  email: 'user@example.com',
  status: 'active',
};

export const mockAuthState = {
  user: mockUser,
  accessToken: 'access-token',
  loading: false,
  login: vi.fn(),
  register: vi.fn(),
  logout: vi.fn(),
};

export function setMockAuth(overrides: Partial<typeof mockAuthState> = {}) {
  Object.assign(mockAuthState, {
    user: mockUser,
    accessToken: 'access-token',
    loading: false,
    login: vi.fn(),
    register: vi.fn(),
    logout: vi.fn(),
  }, overrides);
}
