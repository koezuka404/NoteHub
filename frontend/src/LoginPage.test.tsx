import { cleanup, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import LoginPage from './LoginPage';
import { mockAuthState, setMockAuth } from './test/mockAuth';
import { renderWithProviders } from './test/render';

vi.mock('./auth', () => ({
  useAuth: () => mockAuthState,
  AuthProvider: ({ children }: { children: React.ReactNode }) => children,
}));

describe('LoginPage', () => {
  function submitForm(user: ReturnType<typeof userEvent.setup>) {
    const form = screen.getByLabelText('メールアドレス').closest('form');
    if (!form) {
      throw new Error('login form not found');
    }
    return user.click(within(form).getByRole('button'));
  }

  afterEach(() => {
    cleanup();
  });

  beforeEach(() => {
    setMockAuth();
  });

  it('logs in successfully', async () => {
    const user = userEvent.setup();
    const login = vi.fn().mockResolvedValue(undefined);
    setMockAuth({ login });

    renderWithProviders(<LoginPage />);

    await user.type(screen.getByLabelText('メールアドレス'), 'user@example.com');
    await user.type(screen.getByLabelText('パスワード'), 'abc12345');
    await submitForm(user);

    await waitFor(() => {
      expect(login).toHaveBeenCalledWith('user@example.com', 'abc12345');
    });
  });

  it('shows login errors', async () => {
    const user = userEvent.setup();
    const login = vi.fn().mockRejectedValue({ code: 'INVALID_CREDENTIALS', message: 'bad' });
    setMockAuth({ login });

    renderWithProviders(<LoginPage />);

    await user.type(screen.getByLabelText('メールアドレス'), 'user@example.com');
    await user.type(screen.getByLabelText('パスワード'), 'abc12345');
    await submitForm(user);

    await waitFor(() => {
      expect(screen.getByText('メールアドレスまたはパスワードが正しくありません')).toBeInTheDocument();
    });
  });

  it('registers with validation and clears password on success', async () => {
    const user = userEvent.setup();
    const register = vi.fn().mockResolvedValue(undefined);
    setMockAuth({ register });

    renderWithProviders(<LoginPage />);

    await user.click(screen.getByRole('button', { name: '新規登録' }));
    await user.type(screen.getByLabelText('名前'), 'Test User');
    await user.type(screen.getByLabelText('メールアドレス'), 'user@example.com');
    await user.type(screen.getByLabelText('パスワード'), 'short');
    await user.click(screen.getByRole('button', { name: '登録する' }));

    await waitFor(() => {
      expect(screen.getByText(/8文字以上/)).toBeInTheDocument();
    });
    expect(register).not.toHaveBeenCalled();

    await user.clear(screen.getByLabelText('パスワード'));
    await user.type(screen.getByLabelText('パスワード'), 'abc12345');
    await user.click(screen.getByRole('button', { name: '登録する' }));

    await waitFor(() => {
      expect(register).toHaveBeenCalledWith('Test User', 'user@example.com', 'abc12345');
    });
    expect(screen.getByLabelText('パスワード')).toHaveValue('');
  });

  it('switches tabs and clears errors', async () => {
    const user = userEvent.setup();
    const login = vi.fn().mockRejectedValue({ code: 'INVALID_CREDENTIALS', message: 'bad' });
    setMockAuth({ login });

    renderWithProviders(<LoginPage />);

    await user.type(screen.getByLabelText('メールアドレス'), 'user@example.com');
    await user.type(screen.getByLabelText('パスワード'), 'abc12345');
    await submitForm(user);

    await waitFor(() => {
      expect(screen.getByText('メールアドレスまたはパスワードが正しくありません')).toBeInTheDocument();
    });

    await user.click(screen.getByRole('button', { name: '新規登録' }));
    expect(screen.queryByText('メールアドレスまたはパスワードが正しくありません')).not.toBeInTheDocument();
    expect(screen.getByLabelText('名前')).toBeInTheDocument();

    await user.click(screen.getAllByRole('button', { name: 'ログイン' })[0]);
    expect(screen.queryByText('メールアドレスまたはパスワードが正しくありません')).not.toBeInTheDocument();
  });
});
