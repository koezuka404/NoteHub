import { Link } from 'react-router-dom';
import HostBadge from './HostBadge';

type WorkspaceTab = 'documents' | 'members';

type WorkspaceHeaderProps = {
  workspaceId: string;
  workspaceName: string;
  role?: string;
  activeTab: WorkspaceTab;
};

export default function WorkspaceHeader({
  workspaceId,
  workspaceName,
  role,
  activeTab,
}: WorkspaceHeaderProps) {
  const isHost = role === 'host';

  return (
    <div className="panel-header">
      <Link to="/workspaces" className="link-button">
        ← ワークスペース一覧
      </Link>
      <div className="workspace-title-row">
        <h2>{workspaceName}</h2>
        {role ? <HostBadge role={role} /> : null}
      </div>
      <div className="workspace-nav-section">
        <nav className="workspace-nav" aria-label="ワークスペースメニュー">
          {activeTab === 'documents' ? (
            <span className="workspace-nav-link active">ドキュメント</span>
          ) : (
            <Link to={`/workspaces/${workspaceId}/documents`} className="workspace-nav-link">
              ドキュメント
            </Link>
          )}
          {activeTab === 'members' ? (
            <span className="workspace-nav-link active">メンバー</span>
          ) : (
            <Link to={`/workspaces/${workspaceId}/members`} className="workspace-nav-link">
              メンバー
            </Link>
          )}
        </nav>
        {isHost ? (
          <p className="host-notice">
            {activeTab === 'documents'
              ? 'ワークスペースの設定変更ができます'
              : 'メンバーの招待を行えます'}
          </p>
        ) : null}
      </div>
    </div>
  );
}
