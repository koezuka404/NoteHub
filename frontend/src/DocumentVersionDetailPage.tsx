import { useCallback, useEffect, useState } from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import { getDocument, getVersion, getWorkspace, listMembers, restoreVersion } from './api';
import { useAuth } from './auth';
import AppLayout from './AppLayout';
import DocumentMonacoEditor from './DocumentMonacoEditor';
import { formatDate, formatVersionType, getErrorMessage, resolveMemberName } from './utils';

export default function DocumentVersionDetailPage() {
  const { workspaceId = '', documentId = '', versionId = '' } = useParams();
  const navigate = useNavigate();
  const { accessToken } = useAuth();
  const [workspaceName, setWorkspaceName] = useState('');
  const [documentTitle, setDocumentTitle] = useState('');
  const [versionType, setVersionType] = useState('');
  const [createdByName, setCreatedByName] = useState('不明');
  const [createdAt, setCreatedAt] = useState('');
  const [content, setContent] = useState('');
  const [loading, setLoading] = useState(true);
  const [restoring, setRestoring] = useState(false);
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');

  const loadPage = useCallback(async () => {
    if (!accessToken || !workspaceId || !documentId || !versionId) {
      return;
    }
    setLoading(true);
    setError('');
    setSuccess('');
    try {
      const [workspace, document, version, members] = await Promise.all([
        getWorkspace(accessToken, workspaceId),
        getDocument(accessToken, documentId),
        getVersion(accessToken, documentId, versionId),
        listMembers(accessToken, workspaceId),
      ]);
      if (document.workspaceId !== workspaceId) {
        setError('ドキュメントがワークスペースに属していません');
        return;
      }
      const memberNames = new Map(members.map((member) => [member.userId, member.name]));
      setWorkspaceName(workspace.name);
      setDocumentTitle(document.title);
      setVersionType(version.versionType);
      setCreatedByName(resolveMemberName(version.createdBy, memberNames));
      setCreatedAt(version.createdAt);
      setContent(version.content);
    } catch (err) {
      setError(getErrorMessage(err));
    } finally {
      setLoading(false);
    }
  }, [accessToken, workspaceId, documentId, versionId]);

  useEffect(() => {
    void loadPage();
  }, [loadPage]);

  async function handleRestore() {
    if (!accessToken || !documentId || !versionId) {
      return;
    }
    if (!window.confirm('この履歴の内容でドキュメントを復元しますか？')) {
      return;
    }
    setRestoring(true);
    setError('');
    setSuccess('');
    try {
      await restoreVersion(accessToken, documentId, versionId);
      setSuccess('復元しましたエディタに戻ります');
      window.setTimeout(() => {
        void navigate(`/workspaces/${workspaceId}/documents/${documentId}`, { replace: true });
      }, 800);
    } catch (err) {
      setError(getErrorMessage(err));
    } finally {
      setRestoring(false);
    }
  }

  if (loading) {
    return (
      <AppLayout>
        <p className="loading panel-loading">読み込み中...</p>
      </AppLayout>
    );
  }

  return (
    <AppLayout>
      {error ? <div className="error app-error">{error}</div> : null}
      {success ? <div className="success app-error">{success}</div> : null}

      <section className="panel editor-panel">
        <div className="panel-header">
          <Link
            to={`/workspaces/${workspaceId}/documents/${documentId}/versions`}
            className="link-button"
          >
            ← 編集履歴一覧
          </Link>
          <h2>{formatVersionType(versionType)}</h2>
          <p className="hint-inline">
            {documentTitle} / {workspaceName} / 保存日時: {formatDate(createdAt)} / 操作者:{' '}
            {createdByName}
          </p>
        </div>

        <DocumentMonacoEditor value={content} readOnly height="420px" />

        <div className="version-actions">
          <button
            type="button"
            className="button compact-button"
            onClick={() => void handleRestore()}
            disabled={restoring}
          >
            {restoring ? '復元中...' : 'この履歴で復元'}
          </button>
          <Link
            to={`/workspaces/${workspaceId}/documents/${documentId}`}
            className="link-button"
          >
            ← {documentTitle || '編集画面'}
          </Link>
        </div>
      </section>
    </AppLayout>
  );
}
