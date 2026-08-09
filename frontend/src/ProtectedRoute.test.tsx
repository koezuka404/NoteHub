import { cleanup, screen } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { Route, Routes } from 'react-router-dom';
import ProtectedRoute from './ProtectedRoute';
import { mockAuthState, setMockAuth } from './test/mockAuth';
import { renderWithProviders } from './test/render';

vi.mock('./auth', () => ({
  useAuth: () => mockAuthState,
  AuthProvider: ({ children }: { children: React.ReactNode }) => children,
}));

function renderProtectedRoute(initialEntry = '/') {
  return renderWithProviders(
    <Routes>
      <Route element={<ProtectedRoute />}>
        <Route path="/" element={<div>protected-content</div>} />
      </Route>
      <Route path="/login" element={<div>login-page</div>} />
    </Routes>,
    { router: { initialEntries: [initialEntry] } },
  );
}

describe('ProtectedRoute', () => {
  afterEach(() => {
    cleanup();
  });

  beforeEach(() => {
    setMockAuth();
  });

  it('shows loading state', () => {
    setMockAuth({ loading: true });
    renderProtectedRoute();
    expect(screen.getByText('読み込み中...')).toBeInTheDocument();
  });

  it('redirects to login without access token', () => {
    setMockAuth({ accessToken: null, loading: false, user: null });
    renderProtectedRoute();
    expect(screen.getByText('login-page')).toBeInTheDocument();
  });

  it('renders child routes when authenticated', () => {
    setMockAuth({ accessToken: 'token', loading: false });
    renderProtectedRoute();
    expect(screen.getByText('protected-content')).toBeInTheDocument();
  });
});
