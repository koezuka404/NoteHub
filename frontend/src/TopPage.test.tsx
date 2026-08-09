import { cleanup, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import TopPage from './TopPage';
import { mockAuthState, mockUser, setMockAuth } from './test/mockAuth';
import { renderWithProviders } from './test/render';

vi.mock('./auth', () => ({
  useAuth: () => mockAuthState,
  AuthProvider: ({ children }: { children: React.ReactNode }) => children,
}));

describe('TopPage', () => {
  afterEach(() => {
    cleanup();
  });

  beforeEach(() => {
    setMockAuth();
  });

  it('shows user info and navigation link', () => {
    renderWithProviders(<TopPage />);

    expect(screen.getByText('ログイン済みです')).toBeInTheDocument();
    expect(screen.getByText(mockUser.name)).toBeInTheDocument();
    expect(screen.getByText(mockUser.email)).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'ワークスペースへ' })).toHaveAttribute('href', '/workspaces');
  });

  it('shows placeholders when user is missing', () => {
    setMockAuth({ user: null });
    renderWithProviders(<TopPage />);

    expect(screen.getAllByText('—')).toHaveLength(2);
  });

  it('calls logout', async () => {
    const user = userEvent.setup();
    const logout = vi.fn();
    setMockAuth({ logout });

    renderWithProviders(<TopPage />);
    await user.click(screen.getByRole('button', { name: 'ログアウト' }));
    expect(logout).toHaveBeenCalled();
  });
});
