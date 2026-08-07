import { subscribeAccessTokenRefresh } from './api';
import { getErrorMessage } from './utils';

export type EditorInfo = {
  userId: string;
  name: string;
};

type WsEnvelope = {
  type: string;
  data: Record<string, unknown>;
  timestamp: string;
};

type DocumentWsHandlers = {
  onOpen?: () => void;
  onClose?: () => void;
  onError?: (message: string) => void;
  onSync?: (content: string) => void;
  onUpdated?: (content: string, updatedBy: string) => void;
  onEditors?: (editors: EditorInfo[]) => void;
  onEditorJoined?: (editor: EditorInfo) => void;
  onEditorLeft?: (editor: EditorInfo) => void;
  onRestored?: (content: string) => void;
  onDeleted?: () => void;
  onWorkspaceDeleted?: () => void;
  onWorkspaceHostSuspended?: () => void;
  onWorkspaceHostDeleted?: () => void;
  onAccountSuspended?: () => void;
  onAccountDeleted?: () => void;
  onMemberRemoved?: () => void;
  onReconnecting?: () => void;
};

type DocumentWsOptions = {
  getAccessToken: () => string | null;
  refreshAccessToken?: (currentToken: string | null) => Promise<string | null>;
};

const MAX_RECONNECT_ATTEMPTS = 8;
const RECONNECT_BASE_MS = 1000;
const TERMINAL_CLOSE_CODES = new Set([1000, 4000, 4001, 4002, 4003]);

function wsBaseUrl(): string {
  const configured = import.meta.env.VITE_WS_BASE_URL as string | undefined;
  if (configured) {
    return configured.replace(/\/$/, '');
  }
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  return `${protocol}//${window.location.host}`;
}

function parseEditors(raw: unknown): EditorInfo[] {
  if (!raw || typeof raw !== 'object' || !('editors' in raw)) {
    return [];
  }
  const editors = (raw as { editors?: unknown }).editors;
  if (!Array.isArray(editors)) {
    return [];
  }
  return editors
    .map((item) => {
      if (!item || typeof item !== 'object') {
        return null;
      }
      const editor = item as { user_id?: string; name?: string };
      if (!editor.user_id || !editor.name) {
        return null;
      }
      return { userId: editor.user_id, name: editor.name };
    })
    .filter((item): item is EditorInfo => item !== null);
}

function parseEditor(raw: unknown): EditorInfo | null {
  if (!raw || typeof raw !== 'object') {
    return null;
  }
  const editor = raw as { user_id?: string; name?: string };
  if (!editor.user_id || !editor.name) {
    return null;
  }
  return { userId: editor.user_id, name: editor.name };
}

export function connectDocumentWebSocket(
  documentId: string,
  options: DocumentWsOptions,
  handlers: DocumentWsHandlers,
) {
  let closed = false;
  let suppressReconnect = false;
  let socket: WebSocket | null = null;
  let reconnectTimer: number | null = null;
  let reconnectAttempt = 0;

  function markTerminalEvent() {
    closed = true;
    clearReconnectTimer();
  }

  function handleEnvelope(envelope: WsEnvelope) {
    switch (envelope.type) {
      case 'document_sync':
        handlers.onSync?.(String(envelope.data.content ?? ''));
        break;
      case 'document_updated':
        handlers.onUpdated?.(
          String(envelope.data.content ?? ''),
          String(envelope.data.updated_by ?? ''),
        );
        break;
      case 'editors_sync':
        handlers.onEditors?.(parseEditors(envelope.data));
        break;
      case 'editor_joined': {
        const editor = parseEditor(envelope.data);
        if (editor) {
          handlers.onEditorJoined?.(editor);
        }
        break;
      }
      case 'editor_left': {
        const editor = parseEditor(envelope.data);
        if (editor) {
          handlers.onEditorLeft?.(editor);
        }
        break;
      }
      case 'document_restored':
        handlers.onRestored?.(String(envelope.data.content ?? ''));
        break;
      case 'document_deleted':
        markTerminalEvent();
        handlers.onDeleted?.();
        break;
      case 'workspace_deleted':
        markTerminalEvent();
        handlers.onWorkspaceDeleted?.();
        break;
      case 'workspace_host_suspended':
        markTerminalEvent();
        handlers.onWorkspaceHostSuspended?.();
        break;
      case 'workspace_host_deleted':
        markTerminalEvent();
        handlers.onWorkspaceHostDeleted?.();
        break;
      case 'account_suspended':
        markTerminalEvent();
        handlers.onAccountSuspended?.();
        break;
      case 'account_deleted':
        markTerminalEvent();
        handlers.onAccountDeleted?.();
        break;
      case 'member_removed':
        markTerminalEvent();
        handlers.onMemberRemoved?.();
        break;
      case 'error': {
        const code = String(envelope.data.code ?? '');
        const message = String(envelope.data.message ?? '');
        handlers.onError?.(getErrorMessage({ code, message }));
        break;
      }
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

  function shouldStopReconnect(code: number, reason: string): boolean {
    if (closed || suppressReconnect) {
      return true;
    }
    if (TERMINAL_CLOSE_CODES.has(code)) {
      return true;
    }
    if (reason === 'client close' || reason === 'token refreshed') {
      return true;
    }
    return false;
  }

  function scheduleReconnect() {
    if (closed || suppressReconnect) {
      return;
    }
    if (reconnectAttempt >= MAX_RECONNECT_ATTEMPTS) {
      handlers.onError?.('WebSocket の再接続に失敗しました。ページを再読み込みしてください。');
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
      handlers.onOpen?.();
    });

    ws.addEventListener('close', (event) => {
      handlers.onClose?.();
      if (socket === ws) {
        socket = null;
      }
      if (shouldStopReconnect(event.code, event.reason)) {
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
    if (closed) {
      return;
    }

    let token = options.getAccessToken();
    if (options.refreshAccessToken) {
      token = await options.refreshAccessToken(token);
    }
    if (!token) {
      handlers.onError?.('WebSocket の再接続に失敗しました。再度ログインしてください。');
      return;
    }

    const url = `${wsBaseUrl()}/ws/documents/${documentId}?access_token=${encodeURIComponent(token)}`;
    const ws = new WebSocket(url);
    socket = ws;
    bindSocket(ws);
  }

  function reconnectForTokenRefresh() {
    if (closed) {
      return;
    }
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
    sendEdit(content: string) {
      if (!socket || socket.readyState !== WebSocket.OPEN) {
        return;
      }
      socket.send(
        JSON.stringify({
          type: 'document_edit',
          data: { content },
        }),
      );
    },
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
