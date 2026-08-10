import { useEffect, useRef, useState } from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import { deleteDocument, ensureFreshAccessToken, getDocument, getWorkspace, saveManualVersion, updateDocument } from './api';
import { useAuth } from './auth';
import { connectDocumentWebSocket, type EditorInfo } from './documentWs';
import AppLayout from './AppLayout';
import DocumentMonacoEditor from './DocumentMonacoEditor';
import HostBadge from './HostBadge';
import { getErrorMessage } from './utils';

export default function DocumentEditorPage() {
  const { workspaceId = '', documentId = '' } = useParams();
  const navigate = useNavigate();
  const { user, accessToken } = useAuth();
  const [title, setTitle] = useState('');
  const [workspaceName, setWorkspaceName] = useState('');
  const [workspaceRole, setWorkspaceRole] = useState('');
  const [content, setContent] = useState('');
  const [editors, setEditors] = useState<EditorInfo[]>([]);
  const [connected, setConnected] = useState(false);
  const [reconnecting, setReconnecting] = useState(false);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [wsError, setWsError] = useState('');
  const [saveMessage, setSaveMessage] = useState('');
  const [saving, setSaving] = useState(false);
  const [titleSaving, setTitleSaving] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const wsRef = useRef<ReturnType<typeof connectDocumentWebSocket> | null>(null);
  const accessTokenRef = useRef<string | null>(accessToken);
  const debounceRef = useRef<number | null>(null);
  const remoteUpdateRef = useRef(false);

  useEffect(() => {
    accessTokenRef.current = accessToken;
  }, [accessToken]);

  useEffect(() => {
    if (!accessToken || !workspaceId || !documentId) {
      return;
    }

    let active = true;
    (async () => {
      setLoading(true);
      setError('');
      try {
        const [workspace, document] = await Promise.all([
          getWorkspace(accessToken, workspaceId),
          getDocument(accessToken, documentId),
        ]);
        if (!active) {
          return;
        }
        if (document.workspaceId !== workspaceId) {
          setError('ドキュメントがワークスペースに属していません');
          return;
        }
        setWorkspaceName(workspace.name);
        setWorkspaceRole(workspace.role);
        setTitle(document.title);
        setContent(document.content);
      } catch (err) {
        if (active) {
          setError(getErrorMessage(err));
        }
      } finally {
        if (active) {
          setLoading(false);
        }
      }
    })();

    return () => {
      active = false;
    };
  }, [accessToken, workspaceId, documentId]);

  useEffect(() => {
    if (!accessToken || !documentId || loading || error) {
      return;
    }

    const connection = connectDocumentWebSocket(
      documentId,
      {
        getAccessToken: () => accessTokenRef.current,
        refreshAccessToken: ensureFreshAccessToken,
      },
      {
      onOpen: () => {
        setConnected(true);
        setReconnecting(false);
        setWsError('');
      },
      onClose: () => {
        setConnected(false);
      },
      onReconnecting: () => {
        setConnected(false);
        setReconnecting(true);
      },
      onError: (message) => {
        setWsError(message);
        setReconnecting(false);
      },
      onSync: (nextContent) => {
        remoteUpdateRef.current = true;
        setContent(nextContent);
      },
      onUpdated: (update) => {
        if (update.updatedBy === user?.id) {
          return;
        }
        if (update.title) {
          setTitle(update.title);
        }
        if (update.content !== undefined) {
          remoteUpdateRef.current = true;
          setContent(update.content);
        }
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
      onDeleted: () => {
        setWsError('このドキュメントは削除されました');
        setConnected(false);
        void navigate(`/workspaces/${workspaceId}/documents`, { replace: true });
      },
      onWorkspaceDeleted: () => {
        setWsError('ワークスペースが削除されました');
        setConnected(false);
        void navigate('/workspaces', { replace: true });
      },
      onWorkspaceHostSuspended: () => {
        setWsError('ホストが停止されているため、このワークスペースは利用できません');
        setConnected(false);
        void navigate('/workspaces', { replace: true });
      },
      onWorkspaceHostDeleted: () => {
        setWsError('ホストが削除されているため、このワークスペースは利用できません');
        setConnected(false);
        void navigate('/workspaces', { replace: true });
      },
      onAccountSuspended: () => {
        setWsError('アカウントが停止されました');
        setConnected(false);
        void navigate('/login', { replace: true });
      },
      onAccountDeleted: () => {
        setWsError('アカウントが削除されました');
        setConnected(false);
        void navigate('/login', { replace: true });
      },
      onMemberRemoved: () => {
        setWsError('ワークスペースから除外されました');
        setConnected(false);
        setReconnecting(false);
        void navigate('/workspaces', { replace: true });
      },
    },
    );

    wsRef.current = connection;
    return () => {
      if (debounceRef.current !== null) {
        window.clearTimeout(debounceRef.current);
      }
      connection.close();
      wsRef.current = null;
      setReconnecting(false);
    };
  }, [documentId, error, loading, navigate, user?.id, workspaceId]);

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

  async function handleManualSave() {
    if (!accessToken || !documentId) {
      return;
    }
    if (debounceRef.current !== null) {
      window.clearTimeout(debounceRef.current);
      debounceRef.current = null;
      wsRef.current?.sendEdit(content);
    }
    setSaving(true);
    setSaveMessage('');
    setWsError('');
    try {
      await saveManualVersion(accessToken, documentId);
      setSaveMessage('保存しました');
    } catch (err) {
      setWsError(getErrorMessage(err));
    } finally {
      setSaving(false);
    }
  }

  async function handleTitleSave() {
    if (!accessToken || !documentId || !title.trim()) {
      return;
    }
    setTitleSaving(true);
    setWsError('');
    try {
      const updated = await updateDocument(accessToken, documentId, title.trim());
      setTitle(updated.title);
    } catch (err) {
      setWsError(getErrorMessage(err));
    } finally {
      setTitleSaving(false);
    }
  }

  async function handleDeleteDocument() {
    if (!accessToken || !documentId) {
      return;
    }
    if (!window.confirm('このドキュメントを削除しますか？')) {
      return;
    }
    setDeleting(true);
    setWsError('');
    try {
      await deleteDocument(accessToken, documentId);
      void navigate(`/workspaces/${workspaceId}/documents`, { replace: true });
    } catch (err) {
      setWsError(getErrorMessage(err));
      setDeleting(false);
    }
  }

  if (loading) {
    return (
      <AppLayout>
        <p className="loading panel-loading">読み込み中...</p>
      </AppLayout>
    );
  }

  if (error) {
    return (
      <AppLayout>
        <div className="error app-error">{error}</div>
        <section className="panel">
          <Link to={`/workspaces/${workspaceId}/documents`} className="link-button">
            ← ドキュメント一覧
          </Link>
        </section>
      </AppLayout>
    );
  }

  return (
    <AppLayout>
      <section className="panel editor-panel">
        <div className="panel-header">
          <Link to={`/workspaces/${workspaceId}/documents`} className="link-button">
            ← {workspaceName}
          </Link>
          <div className="workspace-nav-section">
            <div className="workspace-nav-row">
              <nav className="workspace-nav" aria-label="ワークスペースメニュー">
                <span className="workspace-nav-link active">ドキュメント</span>
                <Link to={`/workspaces/${workspaceId}/members`} className="workspace-nav-link">
                  メンバー
                </Link>
              </nav>
              {workspaceRole ? <HostBadge role={workspaceRole} /> : null}
            </div>
            {workspaceRole === 'host' ? (
              <p className="host-notice">ワークスペースの設定変更ができます</p>
            ) : null}
          </div>
          <div className="editor-heading">
            <div className="editor-title-form">
              <input
                className="editor-title-input"
                value={title}
                onChange={(event) => setTitle(event.target.value)}
                maxLength={100}
                aria-label="ドキュメントタイトル"
              />
              <button
                type="button"
                className="button compact-button secondary-button"
                onClick={() => void handleTitleSave()}
                disabled={titleSaving || !title.trim()}
              >
                {titleSaving ? '保存中...' : 'タイトル保存'}
              </button>
            </div>
            <span className={`status-badge ${connected ? 'online' : reconnecting ? 'reconnecting' : 'offline'}`}>
              {connected ? '接続中' : reconnecting ? '再接続中...' : '切断'}
            </span>
          </div>
        </div>

        {wsError ? <div className="error">{wsError}</div> : null}

        <div className="editor-meta">
          <span>編集中: {editors.length > 0 ? editors.map((editor) => editor.name).join(', ') : 'なし'}</span>
        </div>

        <div className="editor-toolbar">
          <div className="editor-toolbar-primary">
            <button
              type="button"
              className="button compact-button"
              onClick={() => void handleManualSave()}
              disabled={saving}
            >
              {saving ? '保存中...' : '保存'}
            </button>
            <Link
              to={`/workspaces/${workspaceId}/documents/${documentId}/versions`}
              className="button compact-button secondary-button link-as-button"
            >
              編集履歴
            </Link>
          </div>
          <button
            type="button"
            className="editor-delete-button"
            onClick={() => void handleDeleteDocument()}
            disabled={deleting}
          >
            {deleting ? '削除中...' : '削除'}
          </button>
        </div>

        {saveMessage ? <div className="success">{saveMessage}</div> : null}

        <DocumentMonacoEditor value={content} onChange={handleContentChange} />
      </section>
    </AppLayout>
  );
}
