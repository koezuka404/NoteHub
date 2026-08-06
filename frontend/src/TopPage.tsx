import { Link } from 'react-router-dom';
import { useAuth } from './auth';

export default function TopPage() {
  const { user, logout } = useAuth();

  return (
    <main className="page">
      <section className="card">
        <h1 className="title">NoteHub</h1>
        <p className="subtitle">ログイン済みです</p>

        <div className="user-card">
          <dl>
            <div>
              <dt>ユーザー名</dt>
              <dd>{user?.name ?? '—'}</dd>
            </div>
            <div>
              <dt>メールアドレス</dt>
              <dd>{user?.email ?? '—'}</dd>
            </div>
          </dl>

          <Link to="/workspaces" className="button link-as-button">
            ワークスペースへ
          </Link>
          <button type="button" className="button secondary-button" onClick={() => void logout()}>
            ログアウト
          </button>
        </div>
      </section>
    </main>
  );
}
