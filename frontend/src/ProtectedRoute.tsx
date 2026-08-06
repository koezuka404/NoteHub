import { Navigate, Outlet } from 'react-router-dom';
import { useAuth } from './auth';

export default function ProtectedRoute() {
  const { accessToken, loading } = useAuth();

  if (loading) {
    return (
      <main className="page">
        <p className="loading">読み込み中...</p>
      </main>
    );
  }

  if (!accessToken) {
    return <Navigate to="/login" replace />;
  }

  return <Outlet />;
}
