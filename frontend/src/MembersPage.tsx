import { FormEvent, useCallback, useEffect, useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import {
  addMember,
  deleteAccount,
  getWorkspace,
  listMembers,
  removeMember,
  reactivateMember,
  searchUser,
  suspendMember,
  type MemberListItem,
  type SearchUserResult,
  type WorkspaceDetail,
} from './api';
import { useAuth } from './auth';
import AppLayout from './AppLayout';
import { formatDate, formatRole, formatUserStatus, getErrorMessage } from './utils';

export default function MembersPage() {
  const { workspaceId = '' } = useParams();
  const { accessToken, user } = useAuth();
  const [workspace, setWorkspace] = useState<WorkspaceDetail | null>(null);
  const [members, setMembers] = useState<MemberListItem[]>([]);
  const [searchEmail, setSearchEmail] = useState('');
  const [searchResult, setSearchResult] = useState<SearchUserResult | null>(null);
  const [searchMessage, setSearchMessage] = useState('');
  const [loading, setLoading] = useState(false);
  const [searching, setSearching] = useState(false);
  const [adding, setAdding] = useState(false);
  const [removingUserId, setRemovingUserId] = useState<string | null>(null);
  const [suspendingUserId, setSuspendingUserId] = useState<string | null>(null);
  const [reactivatingUserId, setReactivatingUserId] = useState<string | null>(null);
  const [deletingAccountUserId, setDeletingAccountUserId] = useState<string | null>(null);
  const [error, setError] = useState('');

  const isHost = workspace?.role === 'host';

  const loadMembers = useCallback(async () => {
    if (!accessToken || !workspaceId) {
      return;
    }
    const items = await listMembers(accessToken, workspaceId);
    setMembers(items);
  }, [accessToken, workspaceId]);

  const loadPage = useCallback(async () => {
    if (!accessToken || !workspaceId) {
      return;
    }
    setLoading(true);
    setError('');
    try {
      const workspaceDetail = await getWorkspace(accessToken, workspaceId);
      setWorkspace(workspaceDetail);
      await loadMembers();
    } catch (err) {
      setError(getErrorMessage(err));
    } finally {
      setLoading(false);
    }
  }, [accessToken, workspaceId, loadMembers]);

  useEffect(() => {
    void loadPage();
  }, [loadPage]);

  async function handleSearch(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!accessToken || !workspaceId || !searchEmail.trim()) {
      return;
    }
    setSearching(true);
    setError('');
    setSearchMessage('');
    setSearchResult(null);
    try {
      const result = await searchUser(accessToken, workspaceId, searchEmail.trim());
      setSearchResult(result);
    } catch (err) {
      const message = getErrorMessage(err);
      if (message.includes('見つかりません')) {
        setSearchMessage('条件に一致するユーザーが見つかりませんでした');
      } else {
        setError(message);
      }
    } finally {
      setSearching(false);
    }
  }

  async function handleAddMember() {
    if (!accessToken || !workspaceId || !searchResult) {
      return;
    }
    setAdding(true);
    setError('');
    try {
      await addMember(accessToken, workspaceId, searchResult.id);
      setSearchResult(null);
      setSearchEmail('');
      setSearchMessage('メンバーを追加しました');
      await loadMembers();
    } catch (err) {
      setError(getErrorMessage(err));
    } finally {
      setAdding(false);
    }
  }

  async function handleRemoveMember(targetUserId: string, targetName: string) {
    if (!accessToken || !workspaceId) {
      return;
    }
    if (!window.confirm(`${targetName} をワークスペースから削除しますか？`)) {
      return;
    }
    setRemovingUserId(targetUserId);
    setError('');
    try {
      await removeMember(accessToken, workspaceId, targetUserId);
      await loadMembers();
    } catch (err) {
      setError(getErrorMessage(err));
    } finally {
      setRemovingUserId(null);
    }
  }

  async function handleSuspendMember(targetUserId: string, targetName: string) {
    if (!accessToken || !workspaceId) {
      return;
    }
    if (!window.confirm(`${targetName} のアカウントを停止しますか？\n停止後はログイン・API・編集ができなくなります。`)) {
      return;
    }
    setSuspendingUserId(targetUserId);
    setError('');
    try {
      await suspendMember(accessToken, workspaceId, targetUserId);
      await loadMembers();
    } catch (err) {
      setError(getErrorMessage(err));
    } finally {
      setSuspendingUserId(null);
    }
  }

  async function handleReactivateMember(targetUserId: string, targetName: string) {
    if (!accessToken || !workspaceId) {
      return;
    }
    if (!window.confirm(`${targetName} のアカウントを復帰しますか？\n復帰後は再度ログインが必要です。`)) {
      return;
    }
    setReactivatingUserId(targetUserId);
    setError('');
    try {
      await reactivateMember(accessToken, workspaceId, targetUserId);
      await loadMembers();
    } catch (err) {
      setError(getErrorMessage(err));
    } finally {
      setReactivatingUserId(null);
    }
  }

  async function handleDeleteAccount(targetUserId: string, targetName: string) {
    if (!accessToken || !workspaceId) {
      return;
    }
    if (
      !window.confirm(
        `${targetName} のアカウントを論理削除しますか？\nこの操作は取り消せず、二度とログインできなくなります。`,
      )
    ) {
      return;
    }
    setDeletingAccountUserId(targetUserId);
    setError('');
    try {
      await deleteAccount(accessToken, workspaceId, targetUserId);
      await loadMembers();
    } catch (err) {
      setError(getErrorMessage(err));
    } finally {
      setDeletingAccountUserId(null);
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
          <h2>{workspace?.name ?? 'メンバー'}</h2>
          <nav className="workspace-nav">
            <Link to={`/workspaces/${workspaceId}/documents`} className="workspace-nav-link">
              ドキュメント
            </Link>
            <span className="workspace-nav-link active">メンバー</span>
          </nav>
        </div>

        {isHost ? (
          <div className="member-section">
            <h3 className="section-title">ユーザーを検索して追加</h3>
            <form className="inline-form" onSubmit={handleSearch}>
              <input
                type="email"
                value={searchEmail}
                onChange={(event) => setSearchEmail(event.target.value)}
                placeholder="追加するユーザーのメールアドレス"
                maxLength={255}
                required
              />
              <button type="submit" className="button compact-button" disabled={searching}>
                {searching ? '検索中...' : '検索'}
              </button>
            </form>

            {searchMessage ? <p className="hint-inline">{searchMessage}</p> : null}

            {searchResult ? (
              <div className="search-result">
                <div>
                  <p className="list-title">{searchResult.name}</p>
                  <p className="list-meta">
                    {searchResult.email} · {formatUserStatus(searchResult.status)}
                  </p>
                </div>
                <button
                  type="button"
                  className="button compact-button"
                  disabled={adding}
                  onClick={() => void handleAddMember()}
                >
                  {adding ? '追加中...' : 'メンバーに追加'}
                </button>
              </div>
            ) : null}
          </div>
        ) : null}

        <div className="member-section">
          <h3 className="section-title">メンバー一覧</h3>
          {loading ? <p className="loading">読み込み中...</p> : null}
          <ul className="item-list">
            {members.map((member) => (
              <li key={member.userId}>
                <div className="member-row">
                  <div>
                    <span className="list-title">{member.name}</span>
                    <span className="list-meta">
                      {member.email || '—'} · {formatRole(member.role)} · {formatUserStatus(member.status)} · 参加:{' '}
                      {formatDate(member.joinedAt)}
                    </span>
                  </div>
                  {isHost && member.role !== 'host' ? (
                    <div className="member-actions">
                      {member.status === 'active' ? (
                        <button
                          type="button"
                          className="button compact-button danger-button"
                          disabled={suspendingUserId === member.userId}
                          onClick={() => void handleSuspendMember(member.userId, member.name)}
                        >
                          {suspendingUserId === member.userId ? '停止中...' : '停止'}
                        </button>
                      ) : null}
                      {member.status === 'suspended' ? (
                        <>
                          <button
                            type="button"
                            className="button compact-button"
                            disabled={reactivatingUserId === member.userId}
                            onClick={() => void handleReactivateMember(member.userId, member.name)}
                          >
                            {reactivatingUserId === member.userId ? '復帰中...' : '復帰'}
                          </button>
                          <button
                            type="button"
                            className="button compact-button danger-button"
                            disabled={deletingAccountUserId === member.userId}
                            onClick={() => void handleDeleteAccount(member.userId, member.name)}
                          >
                            {deletingAccountUserId === member.userId ? '削除中...' : 'アカウント削除'}
                          </button>
                        </>
                      ) : null}
                      {member.status !== 'deleted' ? (
                        <button
                          type="button"
                          className="button compact-button danger-button"
                          disabled={removingUserId === member.userId}
                          onClick={() => void handleRemoveMember(member.userId, member.name)}
                        >
                          {removingUserId === member.userId ? '削除中...' : 'WSから除外'}
                        </button>
                      ) : null}
                    </div>
                  ) : null}
                </div>
              </li>
            ))}
          </ul>
          {!loading && members.length === 0 ? <p className="loading">メンバーがいません</p> : null}
        </div>

        {!isHost && user ? (
          <p className="hint-inline">メンバーの追加・停止・復帰・アカウント削除はホストのみ実行できます。</p>
        ) : null}
      </section>
    </AppLayout>
  );
}
