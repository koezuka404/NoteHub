import { cleanup, render, screen, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import App from './App';
import { mockAuthState, mockUser, setMockAuth } from './test/mockAuth';
import * as api from './api';

vi.mock('./auth', () => ({
  useAuth: () => mockAuthState,
  AuthProvider: ({ children }: { children: React.ReactNode }) => children,
}));

vi.mock('./api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('./api')>();
  return {
    ...actual,
    listWorkspaces: vi.fn(),
    getWorkspace: vi.fn(),
    listDocuments: vi.fn(),
    ensureCsrfToken: vi.fn().mockResolvedValue(undefined),
  };
});

describe('App', () => {
  afterEach(() => {
    cleanup();
    window.history.pushState({}, '', '/');
  });

  beforeEach(() => {
    vi.clearAllMocks();
    setMockAuth({ accessToken: null, loading: false, user: null });
    vi.mocked(api.listWorkspaces).mockResolvedValue([]);
    vi.mocked(api.getWorkspace).mockResolvedValue({ id: 'ws-1', name: 'WS', role: 'host' });
    vi.mocked(api.listDocuments).mockResolvedValue([]);
  });

  it('renders login page at /login', () => {
    window.history.pushState({}, '', '/login');
    render(<App />);
    expect(screen.getByLabelText('メールアドレス')).toBeInTheDocument();
  });

  it('renders top page for authenticated users', async () => {
    setMockAuth({ accessToken: 'token', user: mockUser, loading: false });
    window.history.pushState({}, '', '/');
    render(<App />);
    await waitFor(() => {
      expect(screen.getByText('ログイン済みです')).toBeInTheDocument();
    });
  });

  it('redirects unknown paths to home', async () => {
    setMockAuth({ accessToken: 'token', user: mockUser, loading: false });
    window.history.pushState({}, '', '/unknown-path');
    render(<App />);
    await waitFor(() => {
      expect(screen.getByText('ログイン済みです')).toBeInTheDocument();
    });
  });

  it('redirects guests from protected routes to login', async () => {
    window.history.pushState({}, '', '/workspaces');
    render(<App />);
    await waitFor(() => {
      expect(screen.getByLabelText('メールアドレス')).toBeInTheDocument();
    });
  });
});
