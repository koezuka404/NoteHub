import { FormEvent, useCallback, useEffect, useRef, useState } from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import {
  createDocument,
  deleteDocument,
  deleteWorkspace,
  ensureFreshAccessToken,
  getWorkspace,
  listDocuments,
  updateWorkspace,
  type DocumentListItem,
  type WorkspaceDetail,
} from './api';
import { useAuth } from './auth';
import AppLayout from './AppLayout';
import DocumentMonacoEditor from './DocumentMonacoEditor';
import HostPanel from './HostPanel';
import WorkspaceHeader from './WorkspaceHeader';
import { connectWorkspaceWebSocket } from './workspaceWs';
import { formatDate, getErrorMessage } from './utils';

export default function DocumentsPage() {
  const { workspaceId = '' } = useParams();
  const navigate = useNavigate();
  const { accessToken } = useAuth();
  const [workspace, setWorkspace] = useState<WorkspaceDetail | null>(null);
  const [documents, setDocuments] = useState<DocumentListItem[]>([]);
  const [documentTitle, setDocumentTitle] = useState('');
  const [documentContent, setDocumentContent] = useState('');
  const [creating, setCreating] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [workspaceNameInput, setWorkspaceNameInput] = useState('');
  const [deleteReason, setDeleteReason] = useState('');
  const [updatingWorkspace, setUpdatingWorkspace] = useState(false);
  const [deletingWorkspace, setDeletingWorkspace] = useState(false);
  const [showDeleteWorkspaceForm, setShowDeleteWorkspaceForm] = useState(false);
  const [deletingDocumentId, setDeletingDocumentId] = useState('');
  const accessTokenRef = useRef<string | null>(accessToken);
  const wsRef = useRef<ReturnType<typeof connectWorkspaceWebSocket> | null>(null);

  const isHost = workspace?.role === 'host';

  useEffect(() => {
    accessTokenRef.current = accessToken;
  }, [accessToken]);

  const loadPage = useCallback(async () => {
    if (!accessToken || !workspaceId) {
      return;
    }
    setLoading(true);
    setError('');
    try {
      const [workspaceDetail, items] = await Promise.all([
        getWorkspace(accessToken, workspaceId),
        listDocuments(accessToken, workspaceId),
      ]);
      setWorkspace(workspaceDetail);
      setWorkspaceNameInput(workspaceDetail.name);
      setDocuments(items);
    } catch (err) {
      setError(getErrorMessage(err));
    } finally {
      setLoading(false);
    }
  }, [accessToken, workspaceId]);

  useEffect(() => {
    void loadPage();
  }, [loadPage]);

  useEffect(() => {
    if (!accessToken || !workspaceId || loading) {
      return;
    }

    const connection = connectWorkspaceWebSocket(
      workspaceId,
      {
        getAccessToken: () => accessTokenRef.current,
        refreshAccessToken: ensureFreshAccessToken,
      },
      {
        onDocumentCreated: (document) => {
          setDocuments((current) => {
            if (current.some((item) => item.id === document.id)) {
              return current;
            }
            return [document, ...current];
          });
        },
        onDocumentUpdated: (document) => {
          setDocuments((current) =>
            current.map((item) => (item.id === document.id ? document : item)),
          );
        },
        onDocumentDeleted: (documentId) => {
          setDocuments((current) => current.filter((item) => item.id !== documentId));
        },
      },
    );
    wsRef.current = connection;

    return () => {
      connection.close();
      wsRef.current = null;
    };
  }, [accessToken, workspaceId, loading]);

  async function handleDeleteDocument(documentId: string) {
    if (!accessToken || !window.confirm('このドキュメントを削除しますか？')) {
      return;
    }
    setDeletingDocumentId(documentId);
    setError('');
    try {
      await deleteDocument(accessToken, documentId);
      setDocuments((current) => current.filter((item) => item.id !== documentId));
    } catch (err) {
      setError(getErrorMessage(err));
    } finally {
      setDeletingDocumentId('');
    }
  }

  async function handleCreateDocument(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!accessToken || !workspaceId || !documentTitle.trim()) {
      return;
    }
    setCreating(true);
    setError('');
    try {
      const created = await createDocument(
        accessToken,
        workspaceId,
        documentTitle.trim(),
        documentContent,
      );
      setDocumentTitle('');
      setDocumentContent('');
      void navigate(`/workspaces/${workspaceId}/documents/${created.id}`);
    } catch (err) {
      setError(getErrorMessage(err));
    } finally {
      setCreating(false);
    }
  }

  async function handleUpdateWorkspace(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!accessToken || !workspaceId || !workspaceNameInput.trim()) {
      return;
    }
    setUpdatingWorkspace(true);
    setError('');
    try {
      const updated = await updateWorkspace(accessToken, workspaceId, workspaceNameInput.trim());
      setWorkspace({ ...workspace!, name: updated.name });
      setWorkspaceNameInput(updated.name);
    } catch (err) {
      setError(getErrorMessage(err));
    } finally {
      setUpdatingWorkspace(false);
    }
  }

  async function handleDeleteWorkspace(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!accessToken || !workspaceId || !deleteReason.trim()) {
      return;
    }
    if (!window.confirm('このワークスペースを削除しますか？')) {
      return;
    }
    setDeletingWorkspace(true);
    setError('');
    try {
      await deleteWorkspace(accessToken, workspaceId, deleteReason.trim());
      void navigate('/workspaces', { replace: true });
    } catch (err) {
      setError(getErrorMessage(err));
      setDeletingWorkspace(false);
    }
  }

  return (
    <AppLayout>
      {error ? <div className="error app-error">{error}</div> : null}

      <section className="panel">
        <WorkspaceHeader
          workspaceId={workspaceId}
          workspaceName={workspace?.name ?? 'ドキュメント'}
          role={workspace?.role}
          activeTab="documents"
        />
        <form className="create-document-form" onSubmit={handleCreateDocument}>
          <div className="field">
            <label htmlFor="document-title">タイトル</label>
            <input
              id="document-title"
              value={documentTitle}
              onChange={(event) => setDocumentTitle(event.target.value)}
              placeholder="新しいドキュメント名"
              maxLength={100}
              required
            />
          </div>
          <div className="field">
            <label htmlFor="document-content">本文</label>
            <DocumentMonacoEditor
              value={documentContent}
              onChange={setDocumentContent}
              height="240px"
            />
          </div>
          <button type="submit" className="button compact-button" disabled={creating}>
            {creating ? '作成中...' : '作成'}
          </button>
        </form>
        {loading ? <p className="loading">読み込み中...</p> : null}
        <ul className="item-list">
          {documents.map((document) => (
            <li key={document.id} className="document-list-item">
              <Link
                to={`/workspaces/${workspaceId}/documents/${document.id}`}
                className="list-button list-link document-list-link"
              >
                <span className="list-title">{document.title}</span>
                <span className="list-meta">更新: {formatDate(document.updatedAt)}</span>
              </Link>
              <button
                type="button"
                className="document-list-delete"
                disabled={deletingDocumentId === document.id}
                onClick={() => void handleDeleteDocument(document.id)}
              >
                {deletingDocumentId === document.id ? '削除中...' : '削除'}
              </button>
            </li>
          ))}
        </ul>

        {isHost ? (
          <HostPanel
            title="ワークスペース設定"
            description="名前の変更や削除が行えます"
          >
            <p className="host-panel-link">
              <Link to={`/workspaces/${workspaceId}/members`}>メンバー管理</Link>
            </p>
            <form className="inline-form" onSubmit={handleUpdateWorkspace}>
              <input
                value={workspaceNameInput}
                onChange={(event) => setWorkspaceNameInput(event.target.value)}
                placeholder="ワークスペース名"
                maxLength={100}
                required
              />
              <button type="submit" className="button compact-button secondary-button" disabled={updatingWorkspace}>
                {updatingWorkspace ? '更新中...' : '名前を更新'}
              </button>
            </form>
            {showDeleteWorkspaceForm ? (
              <form className="create-document-form workspace-delete-form" onSubmit={handleDeleteWorkspace}>
                <div className="field">
                  <label htmlFor="workspace-delete-reason">削除理由</label>
                  <textarea
                    id="workspace-delete-reason"
                    className="reason-textarea"
                    value={deleteReason}
                    onChange={(event) => setDeleteReason(event.target.value)}
                    placeholder="削除理由を入力してください"
                    maxLength={500}
                    rows={3}
                    required
                  />
                </div>
                <div className="form-actions">
                  <button type="submit" className="button compact-button danger-button" disabled={deletingWorkspace}>
                    {deletingWorkspace ? '削除中...' : '削除する'}
                  </button>
                  <button
                    type="button"
                    className="button compact-button secondary-button"
                    disabled={deletingWorkspace}
                    onClick={() => {
                      setShowDeleteWorkspaceForm(false);
                      setDeleteReason('');
                    }}
                  >
                    キャンセル
                  </button>
                </div>
              </form>
            ) : (
              <button
                type="button"
                className="button compact-button danger-button workspace-delete-toggle"
                onClick={() => setShowDeleteWorkspaceForm(true)}
              >
                ワークスペースを削除
              </button>
            )}
          </HostPanel>
        ) : null}
      </section>
    </AppLayout>
  );
}
