import { FormEvent, useState } from 'react';
import { validatePassword, type ApiError } from './api';
import { useAuth } from './auth';

type Mode = 'login' | 'register';

function getErrorMessage(error: unknown): string {
  if (error && typeof error === 'object' && 'message' in error) {
    return String((error as ApiError).message);
  }
  return 'リクエストに失敗しました';
}

export default function App() {
  const { user, accessToken, loading, login, register, logout } = useAuth();
  const [mode, setMode] = useState<Mode>('login');
  const [name, setName] = useState('');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [message, setMessage] = useState('');
  const [error, setError] = useState('');
  const [submitting, setSubmitting] = useState(false);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError('');
    setMessage('');
    setSubmitting(true);

    try {
      if (mode === 'register') {
        const passwordError = validatePassword(password);
        if (passwordError) {
          setError(passwordError);
          return;
        }
        await register(name, email, password);
        setMessage('登録が完了しました。ログインしてください。');
        setMode('login');
        setPassword('');
        return;
      }

      await login(email, password);
      setPassword('');
    } catch (err) {
      setError(getErrorMessage(err));
    } finally {
      setSubmitting(false);
    }
  }

  if (loading) {
    return (
      <main className="page">
        <p className="loading">読み込み中...</p>
      </main>
    );
  }

  if (accessToken) {
    return (
      <main className="page">
        <section className="card user-card">
          <div>
            <h1 className="title">NoteHub</h1>
            <p className="subtitle">ログイン中</p>
          </div>
          {user ? (
            <dl>
              <div>
                <dt>名前</dt>
                <dd>{user.name}</dd>
              </div>
              <div>
                <dt>メールアドレス</dt>
                <dd>{user.email}</dd>
              </div>
            </dl>
          ) : null}
          <button type="button" className="button secondary-button" onClick={() => void logout()}>
            ログアウト
          </button>
        </section>
      </main>
    );
  }

  return (
    <main className="page">
      <section className="card">
        <h1 className="title">NoteHub</h1>
        <p className="subtitle">共同編集エディタ</p>

        <div className="tabs">
          <button
            type="button"
            className={mode === 'login' ? 'tab active' : 'tab'}
            onClick={() => {
              setMode('login');
              setError('');
              setMessage('');
            }}
          >
            ログイン
          </button>
          <button
            type="button"
            className={mode === 'register' ? 'tab active' : 'tab'}
            onClick={() => {
              setMode('register');
              setError('');
              setMessage('');
            }}
          >
            新規登録
          </button>
        </div>

        {message ? <div className="success">{message}</div> : null}
        {error ? <div className="error">{error}</div> : null}

        <form onSubmit={handleSubmit}>
          {mode === 'register' ? (
            <div className="field">
              <label htmlFor="name">名前</label>
              <input
                id="name"
                name="name"
                value={name}
                onChange={(event) => setName(event.target.value)}
                maxLength={50}
                required
              />
            </div>
          ) : null}

          <div className="field">
            <label htmlFor="email">メールアドレス</label>
            <input
              id="email"
              name="email"
              type="email"
              value={email}
              onChange={(event) => setEmail(event.target.value)}
              autoComplete="email"
              required
            />
          </div>

          <div className="field">
            <label htmlFor="password">パスワード</label>
            <input
              id="password"
              name="password"
              type="password"
              value={password}
              onChange={(event) => setPassword(event.target.value)}
              autoComplete={mode === 'login' ? 'current-password' : 'new-password'}
              minLength={8}
              maxLength={15}
              required
            />
          </div>

          {mode === 'register' ? (
            <p className="hint">8〜15文字、英字と数字をそれぞれ1文字以上</p>
          ) : null}

          <button type="submit" className="button" disabled={submitting}>
            {submitting ? '送信中...' : mode === 'login' ? 'ログイン' : '登録する'}
          </button>
        </form>
      </section>
    </main>
  );
}
