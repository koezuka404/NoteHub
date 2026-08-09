import { cleanup, screen } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { Route, Routes } from 'react-router-dom';
import PublicRoute from './PublicRoute';
import { mockAuthState, setMockAuth } from './test/mockAuth';
import { renderWithProviders } from './test/render';

vi.mock('./auth', () => ({
  useAuth: () => mockAuthState,
  AuthProvider: ({ children }: { children: React.ReactNode }) => children,
}));

function renderPublicRoute(initialEntry = '/login') {
  return renderWithProviders(
    <Routes>
      <Route element={<PublicRoute />}>
        <Route path="/login" element={<div>login-content</div>} />
      </Route>
      <Route path="/" element={<div>home-page</div>} />
    </Routes>,
    { router: { initialEntries: [initialEntry] } },
  );
}

describe('PublicRoute', () => {
  afterEach(() => {
    cleanup();
  });

  beforeEach(() => {
    setMockAuth({ accessToken: null, loading: false, user: null });
  });

  it('shows loading state', () => {
    setMockAuth({ loading: true });
    renderPublicRoute();
    expect(screen.getByText('読み込み中...')).toBeInTheDocument();
  });

  it('redirects authenticated users to home', () => {
    setMockAuth({ accessToken: 'token', loading: false });
    renderPublicRoute();
    expect(screen.getByText('home-page')).toBeInTheDocument();
  });

  it('renders child routes for guests', () => {
    renderPublicRoute();
    expect(screen.getByText('login-content')).toBeInTheDocument();
  });
});
