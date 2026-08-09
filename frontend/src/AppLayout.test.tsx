import { cleanup, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import AppLayout from './AppLayout';
import { mockAuthState, setMockAuth } from './test/mockAuth';
import { renderWithProviders } from './test/render';

vi.mock('./auth', () => ({
  useAuth: () => mockAuthState,
  AuthProvider: ({ children }: { children: React.ReactNode }) => children,
}));

describe('AppLayout', () => {
  afterEach(() => {
    cleanup();
  });

  beforeEach(() => {
    setMockAuth();
  });

  it('renders shell and children', () => {
    renderWithProviders(
      <AppLayout>
        <p>page body</p>
      </AppLayout>,
    );

    expect(screen.getByRole('heading', { name: 'NoteHub' })).toBeInTheDocument();
    expect(screen.getByText('page body')).toBeInTheDocument();
  });

  it('calls logout from header button', async () => {
    const user = userEvent.setup();
    const logout = vi.fn();
    setMockAuth({ logout });

    renderWithProviders(
      <AppLayout>
        <p>page body</p>
      </AppLayout>,
    );

    await user.click(screen.getByRole('button', { name: 'ログアウト' }));
    expect(logout).toHaveBeenCalled();
  });
});
