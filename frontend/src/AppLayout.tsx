import { Link } from 'react-router-dom';
import type { ReactNode } from 'react';
import { useAuth } from './auth';

type AppLayoutProps = {
  children: ReactNode;
};

export default function AppLayout({ children }: AppLayoutProps) {
  const { logout } = useAuth();

  return (
    <div className="app-shell">
      <header className="app-header">
        <div>
          <h1 className="app-title">NoteHub</h1>
          <Link to="/" className="app-home-link">
            トップへ
          </Link>
        </div>
        <button type="button" className="button secondary-button header-button" onClick={() => void logout()}>
          ログアウト
        </button>
      </header>
      {children}
    </div>
  );
}
