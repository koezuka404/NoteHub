import { cleanup, fireEvent, screen, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import MembersPage from './MembersPage';
import { AuthProvider } from './auth';
import { mockAuthState, setMockAuth } from './test/mockAuth';
import { renderWithProviders } from './test/render';
import * as api from './api';

vi.mock('./auth', () => ({
  useAuth: () => mockAuthState,
  AuthProvider: ({ children }: { children: React.ReactNode }) => children,
}));

vi.mock('./api');

const hostWorkspace = { id: 'ws-1', name: 'Workspace', role: 'host' as const };
const memberWorkspace = { ...hostWorkspace, role: 'member' as const };

const activeMember = {
  userId: 'u2',
  name: 'Member Two',
  email: 'member@example.com',
  status: 'active' as const,
  role: 'member' as const,
  joinedAt: '2026-01-01T00:00:00Z',
};

const suspendedMember = {
  ...activeMember,
  userId: 'u3',
  name: 'Suspended User',
  status: 'suspended' as const,
};

const deletedMember = {
  ...activeMember,
  userId: 'u4',
  name: 'Deleted User',
  status: 'deleted' as const,
};

function submitSearch(email: string) {
  fireEvent.change(screen.getByPlaceholderText('追加するユーザーのメールアドレス'), {
    target: { value: email },
  });
  fireEvent.submit(screen.getByPlaceholderText('追加するユーザーのメールアドレス').closest('form')!);
}

const hostMember = {
  userId: 'u-host',
  name: 'Host User',
  email: '',
  status: 'active' as const,
  role: 'host' as const,
  joinedAt: '2026-01-01T00:00:00Z',
};

function renderPage(initialEntry = '/workspaces/ws-1/members') {
  return renderWithProviders(
    <Routes>
      <Route path="/workspaces/:workspaceId/members" element={<MembersPage />} />
    </Routes>,
    { router: { initialEntries: [initialEntry] } },
  );
}

function rerenderPage(view: ReturnType<typeof renderPage>, initialEntry = '/workspaces/ws-1/members') {
  view.rerender(
    <MemoryRouter initialEntries={[initialEntry]}>
      <AuthProvider>
        <Routes>
          <Route path="/workspaces/:workspaceId/members" element={<MembersPage />} />
        </Routes>
      </AuthProvider>
    </MemoryRouter>,
  );
}

describe.sequential('MembersPage', () => {
  afterEach(() => {
    cleanup();
  });

  beforeEach(() => {
    vi.clearAllMocks();
    setMockAuth();
    vi.spyOn(window, 'confirm').mockReturnValue(true);
    vi.mocked(api.getWorkspace).mockResolvedValue(hostWorkspace);
    vi.mocked(api.listMembers).mockResolvedValue([activeMember, suspendedMember, deletedMember]);
  });

  it('loads members for host and supports search/add flow', async () => {
    vi.mocked(api.searchUser).mockResolvedValue({
      id: 'u5',
      name: 'Found User',
      email: 'found@example.com',
      status: 'active',
    });
    vi.mocked(api.addMember).mockResolvedValue(activeMember);

    renderPage();

    await waitFor(() => expect(screen.getByText('Member Two')).toBeInTheDocument());

    submitSearch('found@example.com');

    await waitFor(() => expect(screen.getByText('Found User')).toBeInTheDocument());
    fireEvent.click(screen.getByRole('button', { name: 'メンバーに追加' }));

    await waitFor(() => {
      expect(api.addMember).toHaveBeenCalledWith('access-token', 'ws-1', 'u5');
      expect(screen.getByText('メンバーを追加しました')).toBeInTheDocument();
    });
  });

  it('shows not-found search message and generic search errors', async () => {
    vi.mocked(api.searchUser).mockRejectedValueOnce({ code: 'USER_NOT_FOUND', message: 'ユーザーが見つかりません' });
    vi.mocked(api.searchUser).mockRejectedValueOnce({ code: 'VALIDATION_ERROR', message: 'bad' });

    renderPage();

    await waitFor(() => expect(screen.getByPlaceholderText('追加するユーザーのメールアドレス')).toBeInTheDocument());

    submitSearch('missing@example.com');
    await waitFor(() => {
      expect(screen.getByText('条件に一致するユーザーが見つかりませんでした')).toBeInTheDocument();
    });

    submitSearch('bad@example.com');
    await waitFor(() => {
      expect(screen.getByText('入力値が不正です')).toBeInTheDocument();
    });
  });

  it('handles remove/suspend/reactivate/delete with confirm cancel and success', async () => {
    vi.mocked(api.removeMember).mockResolvedValue({});
    vi.mocked(api.suspendMember).mockResolvedValue({ userId: 'u2', status: 'suspended', suspendedAt: '2026-01-01' });
    vi.mocked(api.reactivateMember).mockResolvedValue({ userId: 'u3', status: 'active' });
    vi.mocked(api.deleteAccount).mockResolvedValue({ userId: 'u3', status: 'deleted', deletedAt: '2026-01-01' });

    renderPage();

    await waitFor(() => expect(screen.getByText('Member Two')).toBeInTheDocument());

    vi.spyOn(window, 'confirm').mockReturnValueOnce(false);
    fireEvent.click(screen.getByRole('button', { name: '停止' }));
    expect(api.suspendMember).not.toHaveBeenCalled();

    fireEvent.click(screen.getByRole('button', { name: '停止' }));
    await waitFor(() => expect(api.suspendMember).toHaveBeenCalledWith('access-token', 'ws-1', 'u2'));

    vi.spyOn(window, 'confirm').mockReturnValueOnce(false);
    fireEvent.click(screen.getByRole('button', { name: '復帰' }));
    expect(api.reactivateMember).not.toHaveBeenCalled();

    fireEvent.click(screen.getByRole('button', { name: '復帰' }));
    await waitFor(() => expect(api.reactivateMember).toHaveBeenCalledWith('access-token', 'ws-1', 'u3'));

    vi.spyOn(window, 'confirm').mockReturnValueOnce(false);
    fireEvent.click(screen.getByRole('button', { name: 'アカウント削除' }));
    expect(api.deleteAccount).not.toHaveBeenCalled();

    fireEvent.click(screen.getByRole('button', { name: 'アカウント削除' }));
    await waitFor(() => expect(api.deleteAccount).toHaveBeenCalledWith('access-token', 'ws-1', 'u3'));

    vi.spyOn(window, 'confirm').mockReturnValueOnce(false);
    fireEvent.click(screen.getAllByRole('button', { name: 'WSから除外' })[0]);
    expect(api.removeMember).not.toHaveBeenCalled();

    fireEvent.click(screen.getAllByRole('button', { name: 'WSから除外' })[0]);
    await waitFor(() => expect(api.removeMember).toHaveBeenCalledWith('access-token', 'ws-1', 'u2'));
  });

  it('shows member-only hint and hides host controls', async () => {
    vi.mocked(api.getWorkspace).mockResolvedValue(memberWorkspace);
    vi.mocked(api.listMembers).mockResolvedValue([]);

    renderPage();

    await waitFor(() => {
      expect(screen.getByText('メンバーがいません')).toBeInTheDocument();
    });
    expect(screen.queryByText('ユーザーを検索して追加')).not.toBeInTheDocument();
    expect(
      screen.getByText('メンバーの追加・停止・復帰・アカウント削除はホストのみ実行できます。'),
    ).toBeInTheDocument();
  });

  it('shows load and action errors', async () => {
    vi.mocked(api.getWorkspace).mockRejectedValueOnce({ code: 'WORKSPACE_NOT_FOUND', message: 'missing' });

    const { unmount: unmountError } = renderPage();

    await waitFor(() => {
      expect(screen.getByText('ワークスペースが見つかりません')).toBeInTheDocument();
    });
    unmountError();

    vi.mocked(api.getWorkspace).mockResolvedValue(hostWorkspace);
    vi.mocked(api.searchUser).mockResolvedValue({
      id: 'u5',
      name: 'Found User',
      email: 'found@example.com',
      status: 'active',
    });
    vi.mocked(api.addMember).mockRejectedValue({ code: 'MEMBER_ALREADY_EXISTS', message: 'exists' });
    vi.mocked(api.suspendMember).mockRejectedValue({ code: 'CANNOT_SUSPEND_HOST', message: 'host' });

    renderPage();

    await waitFor(() => expect(screen.getByText('Member Two')).toBeInTheDocument());

    submitSearch('found@example.com');
    await waitFor(() => expect(screen.getByText('Found User')).toBeInTheDocument());
    fireEvent.click(screen.getByRole('button', { name: 'メンバーに追加' }));
    await waitFor(() => expect(screen.getByText('既に参加しています')).toBeInTheDocument());

    fireEvent.click(screen.getByRole('button', { name: '停止' }));
    await waitFor(() => expect(screen.getByText('ホストを停止できません')).toBeInTheDocument());
  });

  it('handles member action errors and missing auth', async () => {
    vi.mocked(api.removeMember).mockRejectedValue({ code: 'MEMBER_NOT_FOUND', message: 'missing' });
    vi.mocked(api.reactivateMember).mockRejectedValue({ code: 'ACCOUNT_NOT_SUSPENDED', message: 'active' });
    vi.mocked(api.deleteAccount).mockRejectedValue({ code: 'CANNOT_DELETE_SELF', message: 'self' });

    renderPage();
    await waitFor(() => expect(screen.getByText('Member Two')).toBeInTheDocument());

    fireEvent.click(screen.getAllByRole('button', { name: 'WSから除外' })[0]);
    await waitFor(() => expect(screen.getByText('メンバーが見つかりません')).toBeInTheDocument());

    fireEvent.click(screen.getByRole('button', { name: '復帰' }));
    await waitFor(() => expect(screen.getByText('このアカウントは停止されていません')).toBeInTheDocument());

    fireEvent.click(screen.getByRole('button', { name: 'アカウント削除' }));
    await waitFor(() => expect(screen.getByText('自分自身を削除できません')).toBeInTheDocument());

    setMockAuth({ accessToken: null });
    submitSearch('found@example.com');
    expect(api.searchUser).not.toHaveBeenCalled();
  });

  it('shows host member without actions and missing email placeholder', async () => {
    vi.mocked(api.listMembers).mockResolvedValue([hostMember, activeMember]);

    renderPage();

    await waitFor(() => expect(screen.getByText('Host User')).toBeInTheDocument());
    expect(screen.getByText(/— · ホスト/)).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'WSから除外' })).toBeInTheDocument();
    expect(screen.getAllByRole('button', { name: 'WSから除外' })).toHaveLength(1);
  });

  it('skips member actions without access token after rerender', async () => {
    const view = renderPage();
    await waitFor(() => expect(screen.getByText('Member Two')).toBeInTheDocument());

    setMockAuth({ accessToken: null });
    rerenderPage(view);

    fireEvent.click(screen.getAllByRole('button', { name: 'WSから除外' })[0]);
    fireEvent.click(screen.getByRole('button', { name: '停止' }));
    fireEvent.click(screen.getByRole('button', { name: '復帰' }));
    fireEvent.click(screen.getByRole('button', { name: 'アカウント削除' }));

    expect(api.removeMember).not.toHaveBeenCalled();
    expect(api.suspendMember).not.toHaveBeenCalled();
    expect(api.reactivateMember).not.toHaveBeenCalled();
    expect(api.deleteAccount).not.toHaveBeenCalled();
  });

  it('skips load with empty workspace id', async () => {
    renderPage('/workspaces//members');
    expect(api.getWorkspace).not.toHaveBeenCalled();
  });

  it('skips add member without access token after search', async () => {
    vi.mocked(api.searchUser).mockResolvedValue({
      id: 'u5',
      name: 'Found User',
      email: 'found@example.com',
      status: 'active',
    });

    const view = renderPage();
    await waitFor(() => expect(screen.getByPlaceholderText('追加するユーザーのメールアドレス')).toBeInTheDocument());
    submitSearch('found@example.com');
    await waitFor(() => expect(screen.getByText('Found User')).toBeInTheDocument());

    setMockAuth({ accessToken: null });
    rerenderPage(view);
    fireEvent.click(screen.getByRole('button', { name: 'メンバーに追加' }));
    expect(api.addMember).not.toHaveBeenCalled();
  });

  it('skips member actions when confirm is cancelled', async () => {
    vi.spyOn(window, 'confirm').mockReturnValue(false);
    renderPage();
    await waitFor(() => expect(screen.getByText('Member Two')).toBeInTheDocument());

    fireEvent.click(screen.getAllByRole('button', { name: 'WSから除外' })[0]);
    fireEvent.click(screen.getByRole('button', { name: '停止' }));
    fireEvent.click(screen.getByRole('button', { name: '復帰' }));
    fireEvent.click(screen.getByRole('button', { name: 'アカウント削除' }));

    expect(api.removeMember).not.toHaveBeenCalled();
    expect(api.suspendMember).not.toHaveBeenCalled();
    expect(api.reactivateMember).not.toHaveBeenCalled();
    expect(api.deleteAccount).not.toHaveBeenCalled();
  });
});
