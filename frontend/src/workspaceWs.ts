import { subscribeAccessTokenRefresh } from './api';
import type { DocumentListItem } from './api';
import { getErrorMessage } from './utils';

type WsEnvelope = {
  type: string;
  data: Record<string, unknown>;
  timestamp: string;
};

type WorkspaceWsHandlers = {
  onDocumentCreated?: (document: DocumentListItem) => void;
  onDocumentUpdated?: (document: DocumentListItem) => void;
  onDocumentDeleted?: (documentId: string) => void;
  onError?: (message: string) => void;
  onReconnecting?: () => void;
};

type WorkspaceWsOptions = {
  getAccessToken: () => string | null;
  refreshAccessToken?: (currentToken: string | null) => Promise<string | null>;
};

const MAX_RECONNECT_ATTEMPTS = 8;
const RECONNECT_BASE_MS = 1000;

function wsBaseUrl(): string {
  const configured = import.meta.env.VITE_WS_BASE_URL as string | undefined;
  if (configured) {
    return configured.replace(/\/$/, '');
  }
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  return `${protocol}//${window.location.host}`;
}

function parseListItem(data: Record<string, unknown>): DocumentListItem | null {
  const id = String(data.document_id ?? '');
  const title = String(data.title ?? '');
  const updatedBy = String(data.updated_by ?? '');
  const updatedAt = String(data.updated_at ?? '');
  if (!id || !title) {
    return null;
  }
  return { id, title, updatedBy, updatedAt };
}

export function connectWorkspaceWebSocket(
  workspaceId: string,
  options: WorkspaceWsOptions,
  handlers: WorkspaceWsHandlers,
) {
  let closed = false;
  let suppressReconnect = false;
  let socket: WebSocket | null = null;
  let reconnectTimer: number | null = null;
  let reconnectAttempt = 0;

  function handleEnvelope(envelope: WsEnvelope) {
    switch (envelope.type) {
      case 'document_created': {
        const item = parseListItem(envelope.data);
        if (item) {
          handlers.onDocumentCreated?.(item);
        }
        break;
      }
      case 'document_updated': {
        const item = parseListItem(envelope.data);
        if (item) {
          handlers.onDocumentUpdated?.(item);
        }
        break;
      }
      case 'document_deleted':
        handlers.onDocumentDeleted?.(String(envelope.data.document_id ?? ''));
        break;
      case 'error':
        handlers.onError?.(
          getErrorMessage({
            code: String(envelope.data.code ?? ''),
            message: String(envelope.data.message ?? ''),
          }),
        );
        break;
      default:
        break;
    }
  }

  function clearReconnectTimer() {
    if (reconnectTimer !== null) {
      window.clearTimeout(reconnectTimer);
      reconnectTimer = null;
    }
  }

  function scheduleReconnect() {
    if (reconnectAttempt >= MAX_RECONNECT_ATTEMPTS) {
      handlers.onError?.('WebSocket の再接続に失敗しましたページを再読み込みしてください');
      return;
    }
    clearReconnectTimer();
    handlers.onReconnecting?.();
    const delay =
      reconnectAttempt === 0 ? 0 : Math.min(RECONNECT_BASE_MS * 2 ** (reconnectAttempt - 1), 10_000);
    reconnectAttempt += 1;
    reconnectTimer = window.setTimeout(() => {
      reconnectTimer = null;
      void connectWithFreshToken();
    }, delay);
  }

  function bindSocket(ws: WebSocket) {
    ws.addEventListener('open', () => {
      reconnectAttempt = 0;
    });
    ws.addEventListener('close', (event) => {
      if (socket === ws) {
        socket = null;
      }
      if (closed || suppressReconnect || event.code === 1000) {
        return;
      }
      scheduleReconnect();
    });
    ws.addEventListener('message', (event) => {
      let envelope: WsEnvelope;
      try {
        envelope = JSON.parse(String(event.data)) as WsEnvelope;
      } catch {
        handlers.onError?.('WebSocket メッセージの解析に失敗しました');
        return;
      }
      handleEnvelope(envelope);
    });
  }

  async function connectWithFreshToken() {
    let token = options.getAccessToken();
    if (options.refreshAccessToken) {
      token = await options.refreshAccessToken(token);
    }
    if (closed) {
      return;
    }
    if (!token) {
      handlers.onError?.('WebSocket の再接続に失敗しました再度ログインしてください');
      return;
    }
    const url = `${wsBaseUrl()}/ws/workspaces/${workspaceId}`;
    const ws = new WebSocket(url, ['bearer', token]);
    socket = ws;
    bindSocket(ws);
  }

  function reconnectForTokenRefresh() {
    clearReconnectTimer();
    reconnectAttempt = 0;
    suppressReconnect = true;
    if (socket && socket.readyState !== WebSocket.CLOSED) {
      socket.close(4000, 'token refreshed');
    }
    socket = null;
    suppressReconnect = false;
    void connectWithFreshToken();
  }

  const unsubscribe = subscribeAccessTokenRefresh(() => {
    reconnectForTokenRefresh();
  });

  void connectWithFreshToken();

  return {
    close() {
      closed = true;
      clearReconnectTimer();
      unsubscribe();
      suppressReconnect = true;
      if (socket && socket.readyState !== WebSocket.CLOSED) {
        socket.close(1000, 'client close');
      }
      socket = null;
    },
  };
}
