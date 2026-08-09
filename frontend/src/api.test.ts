import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { mockFetch } from './test/mocks/fetch';
import * as api from './api';

const user = { id: 'u1', name: 'User', email: 'user@example.com', status: 'active' };

function clearCsrfCookie() {
  document.cookie = 'notehub_csrf_token=; Max-Age=0; path=/';
}

function resetApiState() {
  api.configureAuthHandlers(null);
  api.stopProactiveRefresh();
  clearCsrfCookie();
  vi.unstubAllGlobals();
  vi.useRealTimers();
}

describe('api helpers', () => {
  beforeEach(resetApiState);
  afterEach(resetApiState);

  it('validatePassword covers all rules', () => {
    expect(api.validatePassword('short1')).toMatch(/8文字以上/);
    expect(api.validatePassword('verylongpassword1')).toMatch(/15文字以下/);
    expect(api.validatePassword('        ')).toBe('パスワードを入力してください');
    expect(api.validatePassword('12345678')).toMatch(/英字と数字/);
    expect(api.validatePassword('abcdefgh')).toMatch(/英字と数字/);
    expect(api.validatePassword('abc12345')).toBeNull();
  });

  it('getCsrfToken reads cookie', () => {
    document.cookie = 'notehub_csrf_token=abc%20123';
    expect(api.getCsrfToken()).toBe('abc 123');
    clearCsrfCookie();
    expect(api.getCsrfToken()).toBe('');
  });

  it('isAccessTokenExpiredOrExpiringSoon handles missing and invalid expiry', () => {
    expect(api.isAccessTokenExpiredOrExpiringSoon()).toBe(true);
    api.configureAuthHandlers({ onAccessTokenRefreshed: vi.fn(), onAuthFailed: vi.fn() });
    mockFetch({ ok: true, data: { accessToken: 't1', tokenType: 'Bearer', expiresAt: 'invalid' } });
    return api.refresh().then(() => {
      expect(api.isAccessTokenExpiredOrExpiringSoon(0)).toBe(true);
    });
  });

  it('subscribeAccessTokenRefresh notifies listeners', async () => {
    const listener = vi.fn();
    const unsubscribe = api.subscribeAccessTokenRefresh(listener);
    mockFetch({ ok: true, data: { accessToken: 't1', tokenType: 'Bearer', expiresAt: new Date(Date.now() + 3600_000).toISOString() } });
    await api.login('user@example.com', 'pass');
    expect(listener).toHaveBeenCalled();
    unsubscribe();
    mockFetch({ ok: true, data: { accessToken: 't2', tokenType: 'Bearer', expiresAt: new Date(Date.now() + 3600_000).toISOString() } });
    await api.refresh();
    expect(listener).toHaveBeenCalledTimes(1);
  });

  it('configureAuthHandlers receives refresh and failure callbacks', async () => {
    const onAccessTokenRefreshed = vi.fn();
    const onAuthFailed = vi.fn();
    api.configureAuthHandlers({ onAccessTokenRefreshed, onAuthFailed });
    mockFetch({ ok: true, data: { user, accessToken: 't1', tokenType: 'Bearer', expiresAt: new Date(Date.now() + 3600_000).toISOString() } });
    await api.login('user@example.com', 'pass');
    expect(onAccessTokenRefreshed).toHaveBeenCalled();
    mockFetch({ ok: false, status: 401, error: { code: 'REFRESH_TOKEN_INVALID', message: 'bad' } });
    await expect(api.ensureFreshAccessToken(null)).resolves.toBeNull();
    expect(onAuthFailed).toHaveBeenCalled();
  });

  it('ensureFreshAccessToken returns current or refreshed token', async () => {
    mockFetch({ ok: true, data: { user, accessToken: 't1', tokenType: 'Bearer', expiresAt: new Date(Date.now() + 3600_000).toISOString() } });
    await api.login('user@example.com', 'pass');
    await expect(api.ensureFreshAccessToken('t1')).resolves.toBe('t1');
    api.stopProactiveRefresh();
    mockFetch({ ok: false, status: 401, error: { code: 'REFRESH_TOKEN_INVALID', message: 'bad' } });
    await expect(api.ensureFreshAccessToken(null)).resolves.toBeNull();
  });

  it('stopProactiveRefresh clears timer', async () => {
    vi.useFakeTimers();
    mockFetch({ ok: true, data: { user, accessToken: 't1', tokenType: 'Bearer', expiresAt: new Date(Date.now() + 120_000).toISOString() } });
    await api.login('user@example.com', 'pass');
    api.stopProactiveRefresh();
    vi.advanceTimersByTime(120_000);
    vi.useRealTimers();
  });

  it('proactive refresh timer triggers refreshAccessToken', async () => {
    vi.useFakeTimers();
    mockFetch([
      {
        ok: true,
        data: {
          user,
          accessToken: 't1',
          tokenType: 'Bearer',
          expiresAt: new Date(Date.now() + 65_000).toISOString(),
        },
      },
      {
        ok: true,
        data: {
          accessToken: 't2',
          tokenType: 'Bearer',
          expiresAt: new Date(Date.now() + 3600_000).toISOString(),
        },
      },
    ]);
    await api.login('user@example.com', 'pass');
    await vi.advanceTimersByTimeAsync(65_000);
    vi.useRealTimers();
  });

  it('proactive refresh timer swallows refresh failures', async () => {
    vi.useFakeTimers();
    mockFetch([
      {
        ok: true,
        data: {
          user,
          accessToken: 't1',
          tokenType: 'Bearer',
          expiresAt: new Date(Date.now() + 65_000).toISOString(),
        },
      },
      {
        ok: false,
        status: 401,
        error: { code: 'REFRESH_TOKEN_INVALID', message: 'bad' },
      },
    ]);
    await api.login('user@example.com', 'pass');
    await vi.advanceTimersByTimeAsync(65_000);
    vi.useRealTimers();
  });

  it('deduplicates concurrent refreshAccessToken calls', async () => {
    mockFetch({
      ok: true,
      data: {
        accessToken: 't2',
        tokenType: 'Bearer',
        expiresAt: new Date(Date.now() + 3600_000).toISOString(),
      },
    });
    const first = api.ensureFreshAccessToken(null);
    const second = api.ensureFreshAccessToken(null);
    await expect(Promise.all([first, second])).resolves.toEqual(['t2', 't2']);
  });

  it('request retries once on ACCESS_TOKEN_EXPIRED', async () => {
    mockFetch([
      { ok: false, status: 401, error: { code: 'ACCESS_TOKEN_EXPIRED', message: 'expired' } },
      { ok: true, data: { accessToken: 't2', tokenType: 'Bearer', expiresAt: new Date(Date.now() + 3600_000).toISOString() } },
      { ok: true, data: { user } },
    ]);
    await expect(api.api('/api/me', { headers: { Authorization: 'Bearer t1' } }, 't1')).resolves.toEqual({ user });
  });

  it('request throws api errors and handles invalid json', async () => {
    mockFetch({ ok: false, status: 500, jsonError: true });
    await expect(api.me('token')).rejects.toMatchObject({ code: 'UNKNOWN_ERROR' });
    mockFetch({ ok: false, status: 403, error: { code: 'FORBIDDEN', message: 'denied' } });
    await expect(api.me('token')).rejects.toMatchObject({ code: 'FORBIDDEN', message: 'denied', status: 403 });
  });
});

describe('auth endpoints', () => {
  beforeEach(resetApiState);
  afterEach(resetApiState);

  it('register login refresh logout me', async () => {
    mockFetch([
      { ok: true, data: { user, createdAt: '2026-01-01T00:00:00Z' } },
      { ok: true, data: { user, accessToken: 't1', tokenType: 'Bearer', expiresAt: new Date(Date.now() + 3600_000).toISOString() } },
      { ok: true, data: { accessToken: 't2', tokenType: 'Bearer', expiresAt: new Date(Date.now() + 3600_000).toISOString() } },
      { ok: true, data: { user } },
      { ok: true, data: { message: 'ok' } },
    ]);
    await expect(api.register('User', 'user@example.com', 'abc12345')).resolves.toMatchObject({ user });
    await expect(api.login('user@example.com', 'abc12345')).resolves.toMatchObject({ accessToken: 't1' });
    await expect(api.refresh()).resolves.toMatchObject({ accessToken: 't2' });
    await expect(api.me('t2')).resolves.toEqual({ user });
    await expect(api.logout('t2')).resolves.toEqual({ message: 'ok' });
  });
});

describe('workspace and document endpoints', () => {
  beforeEach(resetApiState);
  afterEach(resetApiState);

  it('calls workspace endpoints', async () => {
    mockFetch([
      { ok: true, data: [] },
      { ok: true, data: { id: 'w1', name: 'WS' } },
      { ok: true, data: { id: 'w1', name: 'WS', role: 'host' } },
      { ok: true, data: { id: 'w1', name: 'New', updatedAt: '2026-01-01' } },
      { ok: true, data: { workspaceId: 'w1', deletedAt: '2026-01-01' } },
    ]);
    await expect(api.listWorkspaces('token')).resolves.toEqual([]);
    await expect(api.createWorkspace('token', 'WS')).resolves.toMatchObject({ id: 'w1' });
    await expect(api.getWorkspace('token', 'w1')).resolves.toMatchObject({ role: 'host' });
    await expect(api.updateWorkspace('token', 'w1', 'New')).resolves.toMatchObject({ name: 'New' });
    await expect(api.deleteWorkspace('token', 'w1', 'reason')).resolves.toMatchObject({ workspaceId: 'w1' });
  });

  it('calls document endpoints', async () => {
    mockFetch([
      { ok: true, data: { id: 'd1', workspaceId: 'w1', title: 'Doc', content: 'x', updatedBy: 'u1', updatedAt: '2026-01-01' } },
      { ok: true, data: [] },
      { ok: true, data: { id: 'd1', title: 'Doc' } },
      { ok: true, data: { id: 'd1', title: 'New', updatedAt: '2026-01-01' } },
      { ok: true, data: { documentId: 'd1', deletedAt: '2026-01-01' } },
    ]);
    await expect(api.getDocument('token', 'd1')).resolves.toMatchObject({ title: 'Doc' });
    await expect(api.listDocuments('token', 'w1')).resolves.toEqual([]);
    await expect(api.createDocument('token', 'w1', 'Doc', 'body')).resolves.toMatchObject({ id: 'd1' });
    await expect(api.updateDocument('token', 'd1', 'New')).resolves.toMatchObject({ title: 'New' });
    await expect(api.deleteDocument('token', 'd1')).resolves.toMatchObject({ documentId: 'd1' });
  });

  it('calls member endpoints', async () => {
    mockFetch([
      { ok: true, data: { id: 'u2', email: 'a@b.com', name: 'A', status: 'active' } },
      { ok: true, data: [] },
      { ok: true, data: { userId: 'u2', name: 'A', email: 'a@b.com', status: 'active', role: 'member', joinedAt: '2026-01-01' } },
      { ok: true, data: {} },
      { ok: true, data: { userId: 'u2', status: 'suspended', suspendedAt: '2026-01-01' } },
      { ok: true, data: { userId: 'u2', status: 'active' } },
      { ok: true, data: { userId: 'u2', status: 'deleted', deletedAt: '2026-01-01' } },
    ]);
    await expect(api.searchUser('token', 'w1', 'a@b.com')).resolves.toMatchObject({ id: 'u2' });
    await expect(api.listMembers('token', 'w1')).resolves.toEqual([]);
    await expect(api.addMember('token', 'w1', 'u2')).resolves.toMatchObject({ userId: 'u2' });
    await expect(api.removeMember('token', 'w1', 'u2')).resolves.toEqual({});
    await expect(api.suspendMember('token', 'w1', 'u2')).resolves.toMatchObject({ status: 'suspended' });
    await expect(api.reactivateMember('token', 'w1', 'u2')).resolves.toMatchObject({ status: 'active' });
    await expect(api.deleteAccount('token', 'w1', 'u2')).resolves.toMatchObject({ status: 'deleted' });
  });

  it('calls version endpoints', async () => {
    mockFetch([
      { ok: true, data: [] },
      { ok: true, data: { id: 'v1', documentId: 'd1', content: 'x', versionType: 'manual_save', createdBy: 'u1', createdAt: '2026-01-01' } },
      { ok: true, data: { id: 'v2', createdAt: '2026-01-01' } },
      { ok: true, data: { documentId: 'd1', versionId: 'v1', restoredAt: '2026-01-01' } },
    ]);
    await expect(api.listVersions('token', 'd1')).resolves.toEqual([]);
    await expect(api.getVersion('token', 'd1', 'v1')).resolves.toMatchObject({ id: 'v1' });
    await expect(api.saveManualVersion('token', 'd1')).resolves.toMatchObject({ id: 'v2' });
    await expect(api.restoreVersion('token', 'd1', 'v1')).resolves.toMatchObject({ versionId: 'v1' });
  });
});
