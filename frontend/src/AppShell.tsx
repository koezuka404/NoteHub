import { FormEvent, useCallback, useEffect, useRef, useState } from 'react';
import {
  createDocument,
  createWorkspace,
  listDocuments,
  listWorkspaces,
  type ApiError,
  type DocumentListItem,
  type WorkspaceListItem,
} from './api';
import { useAuth } from './auth';
import { connectDocumentWebSocket, type EditorInfo } from './documentWs';

type View =
  | { kind: 'workspaces' }
  | { kind: 'documents'; workspace: WorkspaceListItem }
  | { kind: 'editor'; workspace: WorkspaceListItem; document: DocumentListItem };

function getErrorMessage(error: unknown): string {
  if (error && typeof error === 'object' && 'message' in error) {
    return String((error as ApiError).message);
  }
  return 'リクエストに失敗しました';
}

function formatDate(value: string): string {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  return date.toLocaleString('ja-JP');
}

export default function AppShell() {
  const { user, accessToken, logout } = useAuth();
  const [view, setView] = useState<View>({ kind: 'workspaces' });
  const [workspaces, setWorkspaces] = useState<WorkspaceListItem[]>([]);
  const [documents, setDocuments] = useState<DocumentListItem[]>([]);
  const [workspaceName, setWorkspaceName] = useState('');
  const [documentTitle, setDocumentTitle] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  const loadWorkspaces = useCallback(async () => {
    if (!accessToken) {
      return;
    }
    setLoading(true);
    setError('');
    try {
      const items = await listWorkspaces(accessToken);
      setWorkspaces(items);
    } catch (err) {
      setError(getErrorMessage(err));
    } finally {
      setLoading(false);
    }
  }, [accessToken]);

  const loadDocuments = useCallback(
    async (workspaceId: string) => {
      if (!accessToken) {
        return;
      }
      setLoading(true);
      setError('');
      try {
        const items = await listDocuments(accessToken, workspaceId);
        setDocuments(items);
      } catch (err) {
        setError(getErrorMessage(err));
      } finally {
        setLoading(false);
      }
    },
    [accessToken],
  );

  useEffect(() => {
    if (view.kind === 'workspaces') {
      void loadWorkspaces();
    }
  }, [view.kind, loadWorkspaces]);

  useEffect(() => {
    if (view.kind === 'documents') {
      void loadDocuments(view.workspace.id);
    }
  }, [view, loadDocuments]);

  async function handleCreateWorkspace(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!accessToken || !workspaceName.trim()) {
      return;
    }
    setError('');
    try {
      await createWorkspace(accessToken, workspaceName.trim());
      setWorkspaceName('');
      await loadWorkspaces();
    } catch (err) {
      setError(getErrorMessage(err));
    }
  }

  async function handleCreateDocument(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!accessToken || view.kind !== 'documents' || !documentTitle.trim()) {
      return;
    }
    setError('');
    try {
      await createDocument(accessToken, view.workspace.id, documentTitle.trim());
      setDocumentTitle('');
      await loadDocuments(view.workspace.id);
    } catch (err) {
      setError(getErrorMessage(err));
    }
  }

  return (
    <div className="app-shell">
      <header className="app-header">
        <div>
          <h1 className="app-title">NoteHub</h1>
          <p className="app-subtitle">{user?.name ?? 'ユーザー'}</p>
        </div>
        <button type="button" className="button secondary-button header-button" onClick={() => void logout()}>
          ログアウト
        </button>
      </header>

      {error ? <div className="error app-error">{error}</div> : null}

      {view.kind === 'workspaces' ? (
        <section className="panel">
          <div className="panel-header">
            <h2>ワークスペース</h2>
          </div>
          <form className="inline-form" onSubmit={handleCreateWorkspace}>
            <input
              value={workspaceName}
              onChange={(event) => setWorkspaceName(event.target.value)}
              placeholder="新しいワークスペース名"
              maxLength={100}
              required
            />
            <button type="submit" className="button compact-button">
              作成
            </button>
          </form>
          {loading ? <p className="loading">読み込み中...</p> : null}
          <ul className="item-list">
            {workspaces.map((workspace) => (
              <li key={workspace.id}>
                <button
                  type="button"
                  className="list-button"
                  disabled={!workspace.isAvailable}
                  onClick={() => setView({ kind: 'documents', workspace })}
                >
                  <span className="list-title">{workspace.name}</span>
                  <span className="list-meta">
                    {workspace.role} · {workspace.isAvailable ? '利用可能' : workspace.unavailableReason}
                  </span>
                </button>
              </li>
            ))}
          </ul>
        </section>
      ) : null}

      {view.kind === 'documents' ? (
        <section className="panel">
          <div className="panel-header">
            <button type="button" className="link-button" onClick={() => setView({ kind: 'workspaces' })}>
              ← ワークスペース一覧
            </button>
            <h2>{view.workspace.name}</h2>
          </div>
          <form className="inline-form" onSubmit={handleCreateDocument}>
            <input
              value={documentTitle}
              onChange={(event) => setDocumentTitle(event.target.value)}
              placeholder="新しいドキュメント名"
              maxLength={100}
              required
            />
            <button type="submit" className="button compact-button">
              作成
            </button>
          </form>
          {loading ? <p className="loading">読み込み中...</p> : null}
          <ul className="item-list">
            {documents.map((document) => (
              <li key={document.id}>
                <button
                  type="button"
                  className="list-button"
                  onClick={() => setView({ kind: 'editor', workspace: view.workspace, document })}
                >
                  <span className="list-title">{document.title}</span>
                  <span className="list-meta">更新: {formatDate(document.updatedAt)}</span>
                </button>
              </li>
            ))}
          </ul>
        </section>
      ) : null}

      {view.kind === 'editor' && accessToken ? (
        <DocumentEditor
          key={view.document.id}
          accessToken={accessToken}
          workspace={view.workspace}
          document={view.document}
          currentUserId={user?.id ?? ''}
          onBack={() => setView({ kind: 'documents', workspace: view.workspace })}
        />
      ) : null}
    </div>
  );
}

type DocumentEditorProps = {
  accessToken: string;
  workspace: WorkspaceListItem;
  document: DocumentListItem;
  currentUserId: string;
  onBack: () => void;
};

function DocumentEditor({ accessToken, workspace, document, currentUserId, onBack }: DocumentEditorProps) {
  const [content, setContent] = useState('');
  const [editors, setEditors] = useState<EditorInfo[]>([]);
  const [connected, setConnected] = useState(false);
  const [wsError, setWsError] = useState('');
  const wsRef = useRef<ReturnType<typeof connectDocumentWebSocket> | null>(null);
  const debounceRef = useRef<number | null>(null);
  const remoteUpdateRef = useRef(false);

  useEffect(() => {
    const connection = connectDocumentWebSocket(document.id, accessToken, {
      onOpen: () => {
        setConnected(true);
        setWsError('');
      },
      onClose: () => {
        setConnected(false);
      },
      onError: (message) => {
        setWsError(message);
      },
      onSync: (nextContent) => {
        remoteUpdateRef.current = true;
        setContent(nextContent);
      },
      onUpdated: (nextContent, updatedBy) => {
        if (updatedBy === currentUserId) {
          return;
        }
        remoteUpdateRef.current = true;
        setContent(nextContent);
      },
      onEditors: setEditors,
      onEditorJoined: (editor) => {
        setEditors((current) => {
          if (current.some((item) => item.userId === editor.userId)) {
            return current;
          }
          return [...current, editor];
        });
      },
      onEditorLeft: (editor) => {
        setEditors((current) => current.filter((item) => item.userId !== editor.userId));
      },
      onRestored: (nextContent) => {
        remoteUpdateRef.current = true;
        setContent(nextContent);
      },
    });

    wsRef.current = connection;
    return () => {
      if (debounceRef.current !== null) {
        window.clearTimeout(debounceRef.current);
      }
      connection.close();
      wsRef.current = null;
    };
  }, [accessToken, currentUserId, document.id]);

  function handleContentChange(nextContent: string) {
    setContent(nextContent);
    if (remoteUpdateRef.current) {
      remoteUpdateRef.current = false;
      return;
    }
    if (debounceRef.current !== null) {
      window.clearTimeout(debounceRef.current);
    }
    debounceRef.current = window.setTimeout(() => {
      wsRef.current?.sendEdit(nextContent);
    }, 300);
  }

  return (
    <section className="panel editor-panel">
      <div className="panel-header">
        <button type="button" className="link-button" onClick={onBack}>
          ← {workspace.name}
        </button>
        <div className="editor-heading">
          <h2>{document.title}</h2>
          <span className={`status-badge ${connected ? 'online' : 'offline'}`}>
            {connected ? '接続中' : '切断'}
          </span>
        </div>
      </div>

      {wsError ? <div className="error">{wsError}</div> : null}

      <div className="editor-meta">
        <span>編集中: {editors.length > 0 ? editors.map((editor) => editor.name).join(', ') : 'なし'}</span>
      </div>

      <textarea
        className="editor-textarea"
        value={content}
        onChange={(event) => handleContentChange(event.target.value)}
        placeholder="ここに入力すると他のタブにも反映されます"
      />
    </section>
  );
}
