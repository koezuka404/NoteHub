import { useAuth } from './auth';
import AppShell from './AppShell';
import LoginPage from './LoginPage';

export default function App() {
  const { accessToken, loading } = useAuth();

  if (loading) {
    return (
      <main className="page">
        <p className="loading">読み込み中...</p>
      </main>
    );
  }

  if (accessToken) {
    return <AppShell />;
  }

  return <LoginPage />;
}
