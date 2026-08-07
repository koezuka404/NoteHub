import { FormEvent, useCallback, useEffect, useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import {
  createDocument,
  getWorkspace,
  listDocuments,
  type DocumentListItem,
  type WorkspaceDetail,
} from './api';
import { useAuth } from './auth';
import AppLayout from './AppLayout';
import { formatDate, getErrorMessage } from './utils';

export default function DocumentsPage() {
  const { workspaceId = '' } = useParams();
  const { accessToken } = useAuth();
  const [workspace, setWorkspace] = useState<WorkspaceDetail | null>(null);
  const [documents, setDocuments] = useState<DocumentListItem[]>([]);
  const [documentTitle, setDocumentTitle] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

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
    setError('');
    try {
      await createDocument(accessToken, workspaceId, documentTitle.trim());
      setDocumentTitle('');
      const items = await listDocuments(accessToken, workspaceId);
      setDocuments(items);
    } catch (err) {
      setError(getErrorMessage(err));
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
      </section>
    </AppLayout>
  );
}
