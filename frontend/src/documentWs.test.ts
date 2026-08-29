import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { connectDocumentWebSocket } from './documentWs';
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

describe('connectDocumentWebSocket', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    installMockWebSocket();
    tokenRefreshListeners.clear();
    vi.stubEnv('VITE_WS_BASE_URL', 'wss://ws.example.com');
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.unstubAllEnvs();
  });

  it('handles all envelope types and editor parsing edge cases', () => {
    const handlers = {
      onOpen: vi.fn(),
      onClose: vi.fn(),
      onSync: vi.fn(),
      onUpdated: vi.fn(),
      onEditors: vi.fn(),
      onEditorJoined: vi.fn(),
      onEditorLeft: vi.fn(),
      onRestored: vi.fn(),
      onError: vi.fn(),
    };

    const connection = connectDocumentWebSocket(
      'doc-1',
      { getAccessToken: () => 'token-1' },
      handlers,
    );

    const socket = lastSocket();
    expect(socket.url).toBe('wss://ws.example.com/ws/documents/doc-1');
    expect(socket.protocols).toEqual(['bearer', 'token-1']);
    socket.open();
    expect(handlers.onOpen).toHaveBeenCalled();

    socket.message({ type: 'document_sync', data: { content: 'hello' }, timestamp: 't' });
    expect(handlers.onSync).toHaveBeenCalledWith('hello');

    socket.message({
      type: 'document_updated',
      data: { content: 'next', title: 'Title', updated_by: 'u1' },
      timestamp: 't',
    });
    expect(handlers.onUpdated).toHaveBeenCalledWith({
      content: 'next',
      title: 'Title',
      updatedBy: 'u1',
    });

    socket.message({
      type: 'document_updated',
      data: { updated_by: 'u2' },
      timestamp: 't',
    });
    expect(handlers.onUpdated).toHaveBeenCalledWith({
      content: undefined,
      title: undefined,
      updatedBy: 'u2',
    });

    socket.message({
      type: 'editors_sync',
      data: { editors: [{ user_id: 'u1', name: 'Alice' }, { user_id: '', name: 'Bad' }, null] },
      timestamp: 't',
    });
    expect(handlers.onEditors).toHaveBeenCalledWith([{ userId: 'u1', name: 'Alice' }]);

    socket.message({ type: 'editors_sync', data: {}, timestamp: 't' });
    socket.message({ type: 'editors_sync', data: { editors: 'bad' }, timestamp: 't' });
    expect(handlers.onEditors).toHaveBeenCalledWith([]);

    socket.message({ type: 'editor_joined', data: { user_id: 'u2', name: 'Bob' }, timestamp: 't' });
    expect(handlers.onEditorJoined).toHaveBeenCalledWith({ userId: 'u2', name: 'Bob' });

    socket.message({ type: 'editor_joined', data: { user_id: 'u2' }, timestamp: 't' });
    socket.message({ type: 'editor_left', data: null, timestamp: 't' });
    expect(handlers.onEditorJoined).toHaveBeenCalledTimes(1);

    socket.message({ type: 'editor_left', data: { user_id: 'u2', name: 'Bob' }, timestamp: 't' });
    expect(handlers.onEditorLeft).toHaveBeenCalledWith({ userId: 'u2', name: 'Bob' });

    socket.message({ type: 'document_restored', data: { content: 'restored' }, timestamp: 't' });
    expect(handlers.onRestored).toHaveBeenCalledWith('restored');

    socket.message({
      type: 'error',
      data: { code: 'DOCUMENT_NOT_FOUND', message: 'missing' },
      timestamp: 't',
    });
    expect(handlers.onError).toHaveBeenCalledWith('ドキュメントが見つかりません');

    socket.message({ type: 'unknown', data: {}, timestamp: 't' });
    socket.message('bad-json');
    expect(handlers.onError).toHaveBeenCalledWith('WebSocket メッセージの解析に失敗しました');

    connection.sendEdit('local edit');
    expect(socket.sent).toEqual([
      JSON.stringify({ type: 'document_edit', data: { content: 'local edit' } }),
    ]);

    connection.close();
  });

  it('sendEdit is ignored when socket is not open', () => {
    const connection = connectDocumentWebSocket(
      'doc-1',
      { getAccessToken: () => 'token-1' },
      {},
    );

    connection.sendEdit('ignored');
    expect(lastSocket().sent).toEqual([]);

    lastSocket().open();
    lastSocket().close(1000, 'client close');
    connection.sendEdit('ignored again');
    expect(lastSocket().sent).toEqual([]);
  });

  it('handles terminal events without reconnecting', () => {
    const terminalHandlers = [
      'onDeleted',
      'onWorkspaceDeleted',
      'onWorkspaceHostSuspended',
      'onWorkspaceHostDeleted',
      'onAccountSuspended',
      'onAccountDeleted',
      'onMemberRemoved',
    ] as const;

    for (const handlerName of terminalHandlers) {
      MockWebSocket.reset();
      installMockWebSocket();

      const handlers = {
        onReconnecting: vi.fn(),
        [handlerName]: vi.fn(),
      };

      connectDocumentWebSocket(
        'doc-1',
        { getAccessToken: () => 'token-1' },
        handlers,
      );

      const socket = lastSocket();
      socket.open();

      const typeMap: Record<(typeof terminalHandlers)[number], string> = {
        onDeleted: 'document_deleted',
        onWorkspaceDeleted: 'workspace_deleted',
        onWorkspaceHostSuspended: 'workspace_host_suspended',
        onWorkspaceHostDeleted: 'workspace_host_deleted',
        onAccountSuspended: 'account_suspended',
        onAccountDeleted: 'account_deleted',
        onMemberRemoved: 'member_removed',
      };

      socket.message({ type: typeMap[handlerName], data: {}, timestamp: 't' });
      expect(handlers[handlerName]).toHaveBeenCalled();

      socket.close(1006, 'disconnect');
      vi.runAllTimers();
      expect(handlers.onReconnecting).not.toHaveBeenCalled();
    }
  });

  it('reconnects on abnormal close and handles token refresh', async () => {
    const handlers = {
      onReconnecting: vi.fn(),
      onClose: vi.fn(),
      onError: vi.fn(),
    };
    const refreshAccessToken = vi.fn(async () => 'fresh-token');

    const connection = connectDocumentWebSocket(
      'doc-1',
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
    expect(handlers.onReconnecting).toHaveBeenCalled();
    expect(refreshAccessToken).toHaveBeenCalledWith('old-token');

    emitTokenRefresh();
    await Promise.resolve();
    expect(MockWebSocket.instances.length).toBeGreaterThan(2);

    connection.close();
    expect(handlers.onClose).toHaveBeenCalled();
  });

  it('stops reconnecting at max attempts or missing token', async () => {
    const handlers = { onError: vi.fn(), onReconnecting: vi.fn() };

    connectDocumentWebSocket(
      'doc-1',
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

    connectDocumentWebSocket(
      'doc-1',
      { getAccessToken: () => 'token-1' },
      handlers,
    );

    for (let attempt = 0; attempt < 8; attempt += 1) {
      lastSocket().close(1006, 'disconnect');
      await vi.runAllTimersAsync();
    }
    lastSocket().close(1006, 'disconnect');
    await vi.runAllTimersAsync();

    expect(handlers.onError).toHaveBeenCalledWith(
      'WebSocket の再接続に失敗しましたページを再読み込みしてください',
    );
  });

  it('does not reconnect for terminal close codes and token refresh close reason', async () => {
    const handlers = { onReconnecting: vi.fn() };

    connectDocumentWebSocket(
      'doc-1',
      { getAccessToken: () => 'token-1' },
      handlers,
    );

    lastSocket().open();

    for (const code of [1000, 4000, 4001, 4002, 4003]) {
      lastSocket().close(code, 'terminal');
      await vi.runAllTimersAsync();
    }

    emitTokenRefresh();
    await Promise.resolve();
    await vi.runAllTimersAsync();

    expect(handlers.onReconnecting).not.toHaveBeenCalled();
  });

  it('ignores reconnect and token refresh after terminal close', async () => {
    const handlers = { onReconnecting: vi.fn(), onDeleted: vi.fn() };
    const connection = connectDocumentWebSocket(
      'doc-1',
      { getAccessToken: () => 'token-1' },
      handlers,
    );

    lastSocket().open();
    lastSocket().message({ type: 'document_deleted', data: {}, timestamp: 't' });
    emitTokenRefresh();
    connection.close();
    lastSocket().close(1006, 'disconnect');

    await vi.runAllTimersAsync();
    expect(handlers.onReconnecting).not.toHaveBeenCalled();
  });

  it('uses default ws url when VITE_WS_BASE_URL is unset', () => {
    vi.stubEnv('VITE_WS_BASE_URL', '');

    connectDocumentWebSocket(
      'doc-1',
      { getAccessToken: () => 'token-1' },
      {},
    );

    expect(lastSocket().url).toContain('/ws/documents/doc-1');
    expect(lastSocket().url).not.toContain('access_token');
    expect(lastSocket().protocols).toEqual(['bearer', 'token-1']);
  });

  it('uses ws protocol on http pages and handles envelope defaults', () => {
    vi.stubEnv('VITE_WS_BASE_URL', '');
    vi.stubGlobal('location', { ...window.location, protocol: 'http:', host: 'example.com' });

    const handlers = {
      onSync: vi.fn(),
      onUpdated: vi.fn(),
      onRestored: vi.fn(),
      onError: vi.fn(),
    };

    connectDocumentWebSocket('doc-1', { getAccessToken: () => 'token-1' }, handlers);
    const socket = lastSocket();
    expect(socket.url).toBe('ws://example.com/ws/documents/doc-1');
    expect(socket.protocols).toEqual(['bearer', 'token-1']);
    socket.open();

    socket.message({ type: 'document_sync', data: {}, timestamp: 't' });
    expect(handlers.onSync).toHaveBeenCalledWith('');

    socket.message({ type: 'document_updated', data: { content: 'x' }, timestamp: 't' });
    expect(handlers.onUpdated).toHaveBeenCalledWith({
      content: 'x',
      title: undefined,
      updatedBy: '',
    });

    socket.message({ type: 'document_restored', data: {}, timestamp: 't' });
    expect(handlers.onRestored).toHaveBeenCalledWith('');

    socket.message({ type: 'error', data: {}, timestamp: 't' });
    expect(handlers.onError).toHaveBeenCalled();
  });

  it('clears reconnect timer and skips reconnect for client close reason', async () => {
    const handlers = { onReconnecting: vi.fn() };

    connectDocumentWebSocket('doc-1', { getAccessToken: () => 'token-1' }, handlers);
    lastSocket().open();
    lastSocket().close(1006, 'client close');

    await vi.runAllTimersAsync();
    expect(handlers.onReconnecting).not.toHaveBeenCalled();
  });

  it('skips reconnect when close reason is token refreshed', async () => {
    const handlers = { onReconnecting: vi.fn() };

    connectDocumentWebSocket('doc-1', { getAccessToken: () => 'token-1' }, handlers);
    lastSocket().open();
    lastSocket().close(1006, 'token refreshed');

    await vi.runAllTimersAsync();
    expect(handlers.onReconnecting).not.toHaveBeenCalled();
  });

  it('clears pending reconnect timer on close', async () => {
    const handlers = { onReconnecting: vi.fn() };
    const connection = connectDocumentWebSocket('doc-1', { getAccessToken: () => 'token-1' }, handlers);

    await Promise.resolve();
    lastSocket().open();
    lastSocket().close(1006, 'disconnect');
    connection.close();
    await vi.runAllTimersAsync();

    expect(handlers.onReconnecting).toHaveBeenCalledTimes(1);
    expect(MockWebSocket.instances).toHaveLength(1);
  });

  it('uses wss protocol on https pages when ws base url is unset', () => {
    vi.stubEnv('VITE_WS_BASE_URL', '');
    vi.stubGlobal('location', { ...window.location, protocol: 'https:', host: 'example.com' });

    connectDocumentWebSocket('doc-1', { getAccessToken: () => 'token-1' }, {});
    expect(lastSocket().url).toBe('wss://example.com/ws/documents/doc-1');
    expect(lastSocket().protocols).toEqual(['bearer', 'token-1']);
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

    const connection = connectDocumentWebSocket(
      'doc-1',
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
    const connection = connectDocumentWebSocket(
      'doc-1',
      { getAccessToken: () => 'token-1' },
      {},
    );

    const socket = lastSocket();
    socket.open();
    connection.close();
    expect(socket.readyState).toBe(MockWebSocket.CLOSED);
  });
});
