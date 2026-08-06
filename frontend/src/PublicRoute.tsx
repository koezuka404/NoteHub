import { Navigate, Outlet } from 'react-router-dom';
import { useAuth } from './auth';

export default function PublicRoute() {
  const { accessToken, loading } = useAuth();

  if (loading) {
    return (
      <main className="page">
        <p className="loading">読み込み中...</p>
      </main>
    );
  }

  if (accessToken) {
    return <Navigate to="/" replace />;
  }

  return <Outlet />;
}
