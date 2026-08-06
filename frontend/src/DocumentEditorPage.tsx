import { useEffect, useRef, useState } from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import { getDocument, getWorkspace } from './api';
import { useAuth } from './auth';
import { connectDocumentWebSocket, type EditorInfo } from './documentWs';
import AppLayout from './AppLayout';
import { getErrorMessage } from './utils';

export default function DocumentEditorPage() {
  const { workspaceId = '', documentId = '' } = useParams();
  const navigate = useNavigate();
  const { user, accessToken } = useAuth();
  const [title, setTitle] = useState('');
  const [workspaceName, setWorkspaceName] = useState('');
  const [content, setContent] = useState('');
  const [editors, setEditors] = useState<EditorInfo[]>([]);
  const [connected, setConnected] = useState(false);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [wsError, setWsError] = useState('');
  const wsRef = useRef<ReturnType<typeof connectDocumentWebSocket> | null>(null);
  const debounceRef = useRef<number | null>(null);
  const remoteUpdateRef = useRef(false);

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
        setTitle(document.title);
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

    const connection = connectDocumentWebSocket(documentId, accessToken, {
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
        if (updatedBy === user?.id) {
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
    });

    wsRef.current = connection;
    return () => {
      if (debounceRef.current !== null) {
        window.clearTimeout(debounceRef.current);
      }
      connection.close();
      wsRef.current = null;
    };
  }, [accessToken, documentId, error, loading, navigate, user?.id, workspaceId]);

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
          <div className="editor-heading">
            <h2>{title}</h2>
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
    </AppLayout>
  );
}
