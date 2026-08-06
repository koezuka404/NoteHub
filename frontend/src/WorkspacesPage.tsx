import { FormEvent, useCallback, useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { createWorkspace, listWorkspaces, type WorkspaceListItem } from './api';
import { useAuth } from './auth';
import AppLayout from './AppLayout';
import { getErrorMessage } from './utils';

export default function WorkspacesPage() {
  const { accessToken } = useAuth();
  const [workspaces, setWorkspaces] = useState<WorkspaceListItem[]>([]);
  const [workspaceName, setWorkspaceName] = useState('');
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

  useEffect(() => {
    void loadWorkspaces();
  }, [loadWorkspaces]);

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

  return (
    <AppLayout>
      {error ? <div className="error app-error">{error}</div> : null}

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
              {workspace.isAvailable ? (
                <Link to={`/workspaces/${workspace.id}/documents`} className="list-button list-link">
                  <span className="list-title">{workspace.name}</span>
                  <span className="list-meta">
                    {workspace.role} · 利用可能
                  </span>
                </Link>
              ) : (
                <div className="list-button list-button-static" aria-disabled="true">
                  <span className="list-title">{workspace.name}</span>
                  <span className="list-meta">
                    {workspace.role} · {workspace.unavailableReason}
                  </span>
                </div>
              )}
            </li>
          ))}
        </ul>
      </section>
    </AppLayout>
  );
}
