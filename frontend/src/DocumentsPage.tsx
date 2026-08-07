import { FormEvent, useCallback, useEffect, useState } from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import {
  createDocument,
  deleteWorkspace,
  getWorkspace,
  listDocuments,
  updateWorkspace,
  type DocumentListItem,
  type WorkspaceDetail,
} from './api';
import { useAuth } from './auth';
import AppLayout from './AppLayout';
import DocumentMonacoEditor from './DocumentMonacoEditor';
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

  const isHost = workspace?.role === 'host';

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
      setWorkspace((current) => (current ? { ...current, name: updated.name } : current));
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
        <div className="panel-header">
          <Link to="/workspaces" className="link-button">
            ← ワークスペース一覧
          </Link>
          <h2>{workspace?.name ?? 'ドキュメント'}</h2>
          <nav className="workspace-nav">
            <span className="workspace-nav-link active">ドキュメント</span>
            <Link to={`/workspaces/${workspaceId}/members`} className="workspace-nav-link">
              メンバー
            </Link>
          </nav>
        </div>
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
            <li key={document.id}>
              <Link
                to={`/workspaces/${workspaceId}/documents/${document.id}`}
                className="list-button list-link"
              >
                <span className="list-title">{document.title}</span>
                <span className="list-meta">更新: {formatDate(document.updatedAt)}</span>
              </Link>
            </li>
          ))}
        </ul>

        {isHost ? (
          <section className="member-section workspace-settings">
            <h3 className="section-title">ワークスペース設定</h3>
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
              <button type="submit" className="button compact-button danger-button" disabled={deletingWorkspace}>
                {deletingWorkspace ? '削除中...' : 'ワークスペースを削除'}
              </button>
            </form>
          </section>
        ) : null}
      </section>
    </AppLayout>
  );
}
