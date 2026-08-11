import { useCallback, useEffect, useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import { getDocument, getWorkspace, listMembers, listVersions, type VersionListItem } from './api';
import { useAuth } from './auth';
import AppLayout from './AppLayout';
import { formatDate, formatVersionType, getErrorMessage, resolveMemberName } from './utils';

export default function DocumentVersionsPage() {
  const { workspaceId = '', documentId = '' } = useParams();
  const { accessToken } = useAuth();
  const [workspaceName, setWorkspaceName] = useState('');
  const [documentTitle, setDocumentTitle] = useState('');
  const [versions, setVersions] = useState<VersionListItem[]>([]);
  const [memberNames, setMemberNames] = useState<Map<string, string>>(() => new Map());
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const loadPage = useCallback(async () => {
    if (!accessToken || !workspaceId || !documentId) {
      return;
    }
    setLoading(true);
    setError('');
    try {
      const [workspace, document, items, members] = await Promise.all([
        getWorkspace(accessToken, workspaceId),
        getDocument(accessToken, documentId),
        listVersions(accessToken, documentId),
        listMembers(accessToken, workspaceId),
      ]);
      if (document.workspaceId !== workspaceId) {
        setError('ドキュメントがワークスペースに属していません');
        return;
      }
      setWorkspaceName(workspace.name);
      setDocumentTitle(document.title);
      setVersions(items);
      setMemberNames(new Map(members.map((member) => [member.userId, member.name])));
    } catch (err) {
      setError(getErrorMessage(err));
    } finally {
      setLoading(false);
    }
  }, [accessToken, workspaceId, documentId]);

  useEffect(() => {
    void loadPage();
  }, [loadPage]);

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

      <section className="panel">
        <div className="panel-header">
          <Link to={`/workspaces/${workspaceId}/documents/${documentId}`} className="link-button">
            ← {documentTitle || 'エディタ'}
          </Link>
          <h2>編集履歴</h2>
          <p className="hint-inline">{workspaceName}</p>
        </div>

        {versions.length === 0 ? (
          <p className="hint-inline">保存された履歴はまだありません</p>
        ) : (
          <ul className="item-list">
            {versions.map((version) => (
              <li key={version.id}>
                <Link
                  to={`/workspaces/${workspaceId}/documents/${documentId}/versions/${version.id}`}
                  className="list-button list-link"
                >
                  <span className="list-title">{formatVersionType(version.versionType)}</span>
                  <span className="list-meta">
                    保存日時: {formatDate(version.createdAt)} · 操作者:{' '}
                    {resolveMemberName(version.createdBy, memberNames)}
                  </span>
                </Link>
              </li>
            ))}
          </ul>
        )}
      </section>
    </AppLayout>
  );
}
