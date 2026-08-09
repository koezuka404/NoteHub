import { cleanup, render, screen, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { AuthProvider, useAuth } from './auth';
import * as api from './api';

vi.mock('./api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('./api')>();
  return {
    ...actual,
    refresh: vi.fn(),
    me: vi.fn(),
    login: vi.fn(),
    register: vi.fn(),
    logout: vi.fn(),
    stopProactiveRefresh: vi.fn(),
    configureAuthHandlers: actual.configureAuthHandlers,
  };
});

function AuthConsumer() {
  const auth = useAuth();
  return (
    <div>
      <span data-testid="auth-loading">{String(auth.loading)}</span>
      <span data-testid="auth-user">{auth.user?.email ?? 'none'}</span>
      <button type="button" onClick={() => void auth.login('user@example.com', 'pass')}>
        login
      </button>
      <button type="button" onClick={() => void auth.register('User', 'user@example.com', 'abc12345')}>
        register
      </button>
      <button type="button" onClick={() => void auth.logout()}>
        logout
      </button>
    </div>
  );
}

describe('AuthProvider', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    cleanup();
  });

  it('loads session on mount', async () => {
    vi.mocked(api.refresh).mockResolvedValue({
      accessToken: 'token',
      tokenType: 'Bearer',
      expiresAt: new Date(Date.now() + 3600_000).toISOString(),
    });
    vi.mocked(api.me).mockResolvedValue({ user: { id: 'u1', name: 'User', email: 'user@example.com', status: 'active' } });

    render(
      <AuthProvider>
        <AuthConsumer />
      </AuthProvider>,
    );

    await waitFor(() => expect(screen.getByTestId('auth-loading')).toHaveTextContent('false'));
    expect(screen.getByTestId('auth-user')).toHaveTextContent('user@example.com');
  });

  it('clears session when refresh fails', async () => {
    vi.mocked(api.refresh).mockRejectedValue(new Error('fail'));
    render(
      <AuthProvider>
        <AuthConsumer />
      </AuthProvider>,
    );
    await waitFor(() => expect(screen.getByTestId('auth-loading')).toHaveTextContent('false'));
    expect(screen.getByTestId('auth-user')).toHaveTextContent('none');
  });

  it('login register and logout update state', async () => {
    vi.mocked(api.refresh).mockRejectedValue(new Error('no session'));
    vi.mocked(api.login).mockResolvedValue({
      user: { id: 'u1', name: 'User', email: 'user@example.com', status: 'active' },
      accessToken: 'token',
      tokenType: 'Bearer',
      expiresAt: new Date(Date.now() + 3600_000).toISOString(),
    });
    vi.mocked(api.register).mockResolvedValue({
      user: { id: 'u1', name: 'User', email: 'user@example.com', status: 'active' },
      createdAt: '2026-01-01',
    });
    vi.mocked(api.logout).mockResolvedValue({ message: 'ok' });

    render(
      <AuthProvider>
        <AuthConsumer />
      </AuthProvider>,
    );
    await waitFor(() => expect(screen.getByTestId('auth-loading')).toHaveTextContent('false'));

    screen.getByRole('button', { name: 'login' }).click();
    await waitFor(() => expect(screen.getByTestId('auth-user')).toHaveTextContent('user@example.com'));

    screen.getByRole('button', { name: 'register' }).click();
    await waitFor(() => expect(api.register).toHaveBeenCalled());

    screen.getByRole('button', { name: 'logout' }).click();
    await waitFor(() => expect(screen.getByTestId('auth-user')).toHaveTextContent('none'));
  });

  it('logout succeeds even when api fails', async () => {
    vi.mocked(api.refresh).mockResolvedValue({
      accessToken: 'token',
      tokenType: 'Bearer',
      expiresAt: new Date(Date.now() + 3600_000).toISOString(),
    });
    vi.mocked(api.me).mockResolvedValue({ user: { id: 'u1', name: 'User', email: 'user@example.com', status: 'active' } });
    vi.mocked(api.logout).mockRejectedValue(new Error('fail'));

    render(
      <AuthProvider>
        <AuthConsumer />
      </AuthProvider>,
    );
    await waitFor(() => expect(screen.getByTestId('auth-user')).toHaveTextContent('user@example.com'));
    screen.getByRole('button', { name: 'logout' }).click();
    await waitFor(() => expect(screen.getByTestId('auth-user')).toHaveTextContent('none'));
  });

  it('useAuth throws outside provider', () => {
    const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
    function BadConsumer() {
      useAuth();
      return null;
    }
    expect(() => render(<BadConsumer />)).toThrow('useAuth must be used within AuthProvider');
    consoleSpy.mockRestore();
  });

  it('handles configured auth callbacks and unmount cleanup', async () => {
    const configureSpy = vi.spyOn(api, 'configureAuthHandlers');
    vi.mocked(api.refresh).mockRejectedValue(new Error('no session'));
    const { unmount } = render(
      <AuthProvider>
        <AuthConsumer />
      </AuthProvider>,
    );
    await waitFor(() => expect(screen.getByTestId('auth-loading')).toHaveTextContent('false'));

    const handlers = configureSpy.mock.calls[0]?.[0];
    expect(handlers).toBeTruthy();
    handlers!.onAccessTokenRefreshed('refreshed-token', new Date(Date.now() + 3600_000).toISOString());
    handlers!.onAuthFailed();
    await waitFor(() => expect(screen.getByTestId('auth-user')).toHaveTextContent('none'));

    unmount();
    expect(configureSpy).toHaveBeenCalledWith(null);
    expect(api.stopProactiveRefresh).toHaveBeenCalled();
    configureSpy.mockRestore();
  });

  it('ignores refresh result after unmount', async () => {
    let resolveRefresh: (value: Awaited<ReturnType<typeof api.refresh>>) => void = () => {};
    vi.mocked(api.refresh).mockImplementation(
      () =>
        new Promise((resolve) => {
          resolveRefresh = resolve;
        }),
    );
    vi.mocked(api.me).mockResolvedValue({
      user: { id: 'u1', name: 'User', email: 'user@example.com', status: 'active' },
    });

    const { unmount } = render(
      <AuthProvider>
        <AuthConsumer />
      </AuthProvider>,
    );
    unmount();
    resolveRefresh({
      accessToken: 'token',
      tokenType: 'Bearer',
      expiresAt: new Date(Date.now() + 3600_000).toISOString(),
    });
    await Promise.resolve();
  });

  it('ignores me result after unmount', async () => {
    let resolveMe: (value: Awaited<ReturnType<typeof api.me>>) => void = () => {};
    vi.mocked(api.refresh).mockResolvedValue({
      accessToken: 'token',
      tokenType: 'Bearer',
      expiresAt: new Date(Date.now() + 3600_000).toISOString(),
    });
    vi.mocked(api.me).mockImplementation(
      () =>
        new Promise((resolve) => {
          resolveMe = resolve;
        }),
    );

    const { unmount } = render(
      <AuthProvider>
        <AuthConsumer />
      </AuthProvider>,
    );
    await waitFor(() => expect(api.refresh).toHaveBeenCalled());
    unmount();
    resolveMe({ user: { id: 'u1', name: 'User', email: 'user@example.com', status: 'active' } });
    await Promise.resolve();
  });

  it('ignores refresh failure after unmount', async () => {
    let rejectRefresh: (reason?: unknown) => void = () => {};
    vi.mocked(api.refresh).mockImplementation(
      () =>
        new Promise((_resolve, reject) => {
          rejectRefresh = reject;
        }),
    );

    const { unmount } = render(
      <AuthProvider>
        <AuthConsumer />
      </AuthProvider>,
    );
    unmount();
    rejectRefresh(new Error('fail'));
    await Promise.resolve();
  });

  it('logout without token is a no-op', async () => {
    vi.mocked(api.refresh).mockRejectedValue(new Error('no session'));
    render(
      <AuthProvider>
        <AuthConsumer />
      </AuthProvider>,
    );
    await waitFor(() => expect(screen.getByTestId('auth-loading')).toHaveTextContent('false'));
    screen.getByRole('button', { name: 'logout' }).click();
    await waitFor(() => expect(screen.getByTestId('auth-user')).toHaveTextContent('none'));
    expect(api.logout).not.toHaveBeenCalled();
  });
});
