import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { connectWorkspaceWebSocket } from './workspaceWs';
import { MockWebSocket, installMockWebSocket } from './test/mocks/websocket';

const tokenRefreshListeners = new Set<() => void>();

vi.mock('./api', () => ({
  subscribeAccessTokenRefresh: vi.fn((listener: () => void) => {
    tokenRefreshListeners.add(listener);
    return () => {
      tokenRefreshListeners.delete(listener);
    };
  }),
}));

function emitTokenRefresh() {
  for (const listener of tokenRefreshListeners) {
    listener();
  }
}

function lastSocket(): MockWebSocket {
  const socket = MockWebSocket.instances.at(-1);
  if (!socket) {
    throw new Error('No WebSocket instance');
  }
  return socket;
}

describe('connectWorkspaceWebSocket', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    installMockWebSocket();
    tokenRefreshListeners.clear();
    vi.stubEnv('VITE_WS_BASE_URL', '');
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.unstubAllEnvs();
  });

  it('connects with default ws base url and handles all message types', () => {
    const handlers = {
      onDocumentCreated: vi.fn(),
      onDocumentUpdated: vi.fn(),
      onDocumentDeleted: vi.fn(),
      onError: vi.fn(),
    };

    connectWorkspaceWebSocket(
      'ws-1',
      { getAccessToken: () => 'token-1' },
      handlers,
    );

    const socket = lastSocket();
    expect(socket.url).toContain('/ws/workspaces/ws-1');
    expect(socket.url).not.toContain('access_token');
    expect(socket.protocols).toEqual(['bearer', 'token-1']);
    expect(socket.url.startsWith('ws://') || socket.url.startsWith('wss://')).toBe(true);
    socket.open();

    socket.message({
      type: 'document_created',
      data: { document_id: 'd1', title: 'Doc 1', updated_by: 'u1', updated_at: '2026-01-01' },
      timestamp: '2026-01-01',
    });
    expect(handlers.onDocumentCreated).toHaveBeenCalledWith({
      id: 'd1',
      title: 'Doc 1',
      updatedBy: 'u1',
      updatedAt: '2026-01-01',
    });

    socket.message({
      type: 'document_updated',
      data: { document_id: 'd1', title: 'Doc 2', updated_by: 'u2', updated_at: '2026-01-02' },
      timestamp: '2026-01-02',
    });
    expect(handlers.onDocumentUpdated).toHaveBeenCalledWith({
      id: 'd1',
      title: 'Doc 2',
      updatedBy: 'u2',
      updatedAt: '2026-01-02',
    });

    socket.message({
      type: 'document_deleted',
      data: { document_id: 'd1' },
      timestamp: '2026-01-02',
    });
    expect(handlers.onDocumentDeleted).toHaveBeenCalledWith('d1');

    socket.message({
      type: 'error',
      data: { code: 'DOCUMENT_NOT_FOUND', message: 'missing' },
      timestamp: '2026-01-02',
    });
    expect(handlers.onError).toHaveBeenCalledWith('ドキュメントが見つかりません');

    socket.message({ type: 'unknown', data: {}, timestamp: '2026-01-02' });
    expect(handlers.onError).toHaveBeenCalledTimes(1);
  });

  it('uses configured VITE_WS_BASE_URL without trailing slash', () => {
    vi.stubEnv('VITE_WS_BASE_URL', 'wss://ws.example.com/');

    connectWorkspaceWebSocket(
      'ws-1',
      { getAccessToken: () => 'token-1' },
      {},
    );

    expect(lastSocket().url).toBe('wss://ws.example.com/ws/workspaces/ws-1');
    expect(lastSocket().protocols).toEqual(['bearer', 'token-1']);
  });

  it('ignores list items missing id or title and invalid json', () => {
    const handlers = {
      onDocumentCreated: vi.fn(),
      onDocumentUpdated: vi.fn(),
      onError: vi.fn(),
    };

    connectWorkspaceWebSocket(
      'ws-1',
      { getAccessToken: () => 'token-1' },
      handlers,
    );

    const socket = lastSocket();
    socket.open();

    socket.message({
      type: 'document_created',
      data: { title: 'No id' },
      timestamp: '2026-01-01',
    });
    socket.message({
      type: 'document_updated',
      data: { document_id: 'd1' },
      timestamp: '2026-01-01',
    });
    expect(handlers.onDocumentCreated).not.toHaveBeenCalled();
    expect(handlers.onDocumentUpdated).not.toHaveBeenCalled();

    socket.message('not-json');
    expect(handlers.onError).toHaveBeenCalledWith('WebSocket メッセージの解析に失敗しました');
  });

  it('refreshes token on reconnect and token refresh subscription', async () => {
    const refreshAccessToken = vi.fn(async () => 'fresh-token');
    const handlers = { onError: vi.fn() };

    const connection = connectWorkspaceWebSocket(
      'ws-1',
      {
        getAccessToken: () => 'old-token',
        refreshAccessToken,
      },
      handlers,
    );

    await Promise.resolve();
    lastSocket().open();
    lastSocket().close(1006, 'disconnect');

    await vi.runAllTimersAsync();
    await Promise.resolve();
    expect(refreshAccessToken).toHaveBeenCalledWith('old-token');
    expect(lastSocket().url).toContain('/ws/workspaces/ws-1');
    expect(lastSocket().protocols).toEqual(['bearer', 'fresh-token']);

    emitTokenRefresh();
    await Promise.resolve();
    expect(MockWebSocket.instances.length).toBeGreaterThan(1);

    connection.close();
  });

  it('reports error when token is unavailable and stops after max reconnect attempts', async () => {
    const handlers = {
      onError: vi.fn(),
      onReconnecting: vi.fn(),
    };

    connectWorkspaceWebSocket(
      'ws-1',
      {
        getAccessToken: () => null,
        refreshAccessToken: async () => null,
      },
      handlers,
    );

    await Promise.resolve();
    expect(handlers.onError).toHaveBeenCalledWith('WebSocket の再接続に失敗しました再度ログインしてください');

    MockWebSocket.reset();
    installMockWebSocket();
    handlers.onError.mockClear();
    handlers.onReconnecting.mockClear();

    connectWorkspaceWebSocket(
      'ws-1',
      { getAccessToken: () => 'token-1' },
      handlers,
    );

    for (let attempt = 0; attempt < 8; attempt += 1) {
      lastSocket().close(1006, 'disconnect');
      await vi.runAllTimersAsync();
    }

    lastSocket().close(1006, 'disconnect');
    await vi.runAllTimersAsync();

    expect(handlers.onReconnecting).toHaveBeenCalled();
    expect(handlers.onError).toHaveBeenCalledWith(
      'WebSocket の再接続に失敗しましたページを再読み込みしてください',
    );
  });

  it('does not reconnect after intentional close or code 1000', async () => {
    const handlers = { onReconnecting: vi.fn() };
    const connection = connectWorkspaceWebSocket(
      'ws-1',
      { getAccessToken: () => 'token-1' },
      handlers,
    );

    const socket = lastSocket();
    socket.open();
    connection.close();
    socket.close(1006, 'disconnect');

    await vi.runAllTimersAsync();
    expect(handlers.onReconnecting).not.toHaveBeenCalled();
    expect(MockWebSocket.instances).toHaveLength(1);
  });

  it('ignores reconnect and token refresh after close', async () => {
    const handlers = { onError: vi.fn(), onReconnecting: vi.fn() };
    const connection = connectWorkspaceWebSocket(
      'ws-1',
      { getAccessToken: () => 'token-1' },
      handlers,
    );

    await Promise.resolve();
    lastSocket().open();
    connection.close();
    emitTokenRefresh();
    lastSocket().close(1006, 'disconnect');

    await vi.runAllTimersAsync();
    expect(handlers.onReconnecting).not.toHaveBeenCalled();
    expect(MockWebSocket.instances).toHaveLength(1);
  });

  it('uses https protocol when page is served over https', () => {
    vi.stubGlobal('location', { ...window.location, protocol: 'https:', host: 'example.com' });

    connectWorkspaceWebSocket(
      'ws-1',
      { getAccessToken: () => 'token-1' },
      {},
    );

    expect(lastSocket().url).toBe('wss://example.com/ws/workspaces/ws-1');
    expect(lastSocket().protocols).toEqual(['bearer', 'token-1']);
  });

  it('does not connect after reconnect timer when already closed', async () => {
    const handlers = { onReconnecting: vi.fn() };
    const connection = connectWorkspaceWebSocket(
      'ws-1',
      { getAccessToken: () => 'token-1' },
      handlers,
    );

    await Promise.resolve();
    lastSocket().open();
    lastSocket().close(1006, 'disconnect');
    connection.close();
    await vi.runAllTimersAsync();

    expect(MockWebSocket.instances).toHaveLength(1);
  });

  it('resets reconnect attempt counter on open', async () => {
    const handlers = { onReconnecting: vi.fn(), onError: vi.fn() };

    connectWorkspaceWebSocket(
      'ws-1',
      { getAccessToken: () => 'token-1' },
      handlers,
    );

    await Promise.resolve();
    const first = lastSocket();
    first.open();
    first.close(1006, 'disconnect');
    await vi.runAllTimersAsync();
    await Promise.resolve();
    const second = lastSocket();
    second.open();
    second.close(1006, 'disconnect');
    await vi.runAllTimersAsync();

    expect(MockWebSocket.instances.length).toBeGreaterThanOrEqual(2);
  });

  it('handles error and deleted envelope defaults', () => {
    const handlers = { onError: vi.fn(), onDocumentDeleted: vi.fn() };

    connectWorkspaceWebSocket('ws-1', { getAccessToken: () => 'token-1' }, handlers);
    const socket = lastSocket();
    socket.open();

    socket.message({ type: 'error', data: {}, timestamp: 't' });
    expect(handlers.onError).toHaveBeenCalled();

    socket.message({ type: 'document_deleted', data: {}, timestamp: 't' });
    expect(handlers.onDocumentDeleted).toHaveBeenCalledWith('');
  });

  it('skips scheduled reconnect when already closed', async () => {
    const handlers = { onReconnecting: vi.fn() };
    const connection = connectWorkspaceWebSocket('ws-1', { getAccessToken: () => 'token-1' }, handlers);

    await Promise.resolve();
    lastSocket().open();
    lastSocket().close(1006, 'disconnect');
    connection.close();
    await vi.runAllTimersAsync();

    expect(MockWebSocket.instances).toHaveLength(1);
  });

  it('skips token refresh reconnect when connection is closed', async () => {
    const handlers = { onReconnecting: vi.fn() };

    const connection = connectWorkspaceWebSocket(
      'ws-1',
      { getAccessToken: () => 'token-1' },
      handlers,
    );

    await Promise.resolve();
    lastSocket().open();
    connection.close();
    emitTokenRefresh();
    await Promise.resolve();
    await vi.runAllTimersAsync();

    expect(MockWebSocket.instances).toHaveLength(1);
  });

  it('refreshes token when socket is already closed', async () => {
    connectWorkspaceWebSocket('ws-1', { getAccessToken: () => 'token-1' }, {});

    await Promise.resolve();
    lastSocket().close(1000, 'done');
    emitTokenRefresh();
    await Promise.resolve();

    expect(MockWebSocket.instances).toHaveLength(2);
  });

  it('aborts in-flight reconnect when connection closes', async () => {
    let resolveRefresh!: (value: string | null) => void;
    let refreshCount = 0;
    const refreshAccessToken = vi.fn(async (token: string | null) => {
      refreshCount += 1;
      if (refreshCount === 1) {
        return token;
      }
      return new Promise<string | null>((resolve) => {
        resolveRefresh = resolve;
      });
    });

    const connection = connectWorkspaceWebSocket(
      'ws-1',
      { getAccessToken: () => 'token-1', refreshAccessToken },
      {},
    );

    await Promise.resolve();
    lastSocket().open();
    lastSocket().close(1006, 'disconnect');
    await vi.advanceTimersByTimeAsync(0);
    connection.close();
    resolveRefresh('fresh-token');
    await Promise.resolve();

    expect(MockWebSocket.instances).toHaveLength(1);
  });

  it('closes open socket on connection close', () => {
    const connection = connectWorkspaceWebSocket(
      'ws-1',
      { getAccessToken: () => 'token-1' },
      {},
    );

    const socket = lastSocket();
    socket.open();
    connection.close();
    expect(socket.readyState).toBe(MockWebSocket.CLOSED);
  });
});
