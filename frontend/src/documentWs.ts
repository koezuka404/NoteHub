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
};

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
  accessToken: string,
  handlers: DocumentWsHandlers,
) {
  const url = `${wsBaseUrl()}/ws/documents/${documentId}?access_token=${encodeURIComponent(accessToken)}`;
  const socket = new WebSocket(url);

  socket.addEventListener('open', () => {
    handlers.onOpen?.();
  });

  socket.addEventListener('close', () => {
    handlers.onClose?.();
  });

  socket.addEventListener('message', (event) => {
    let envelope: WsEnvelope;
    try {
      envelope = JSON.parse(String(event.data)) as WsEnvelope;
    } catch {
      handlers.onError?.('WebSocket メッセージの解析に失敗しました');
      return;
    }

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
      case 'error':
        handlers.onError?.(String(envelope.data.message ?? 'WebSocket エラー'));
        break;
      default:
        break;
    }
  });

  return {
    sendEdit(content: string) {
      if (socket.readyState !== WebSocket.OPEN) {
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
      socket.close();
    },
  };
}
