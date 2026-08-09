import { cleanup, fireEvent, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { Route, Routes } from 'react-router-dom';
import WorkspacesPage from './WorkspacesPage';
import { mockAuthState, setMockAuth } from './test/mockAuth';
import { renderWithProviders } from './test/render';
import * as api from './api';

vi.mock('./auth', () => ({
  useAuth: () => mockAuthState,
  AuthProvider: ({ children }: { children: React.ReactNode }) => children,
}));

vi.mock('./api');

const workspaces = [
  {
    id: 'ws-1',
    name: 'Available WS',
    role: 'host' as const,
    isAvailable: true,
    unavailableReason: '',
  },
  {
    id: 'ws-2',
    name: 'Unavailable WS',
    role: 'member' as const,
    isAvailable: false,
    unavailableReason: 'HOST_SUSPENDED',
  },
];

function renderPage() {
  return renderWithProviders(
    <Routes>
      <Route path="/workspaces" element={<WorkspacesPage />} />
    </Routes>,
    { router: { initialEntries: ['/workspaces'] } },
  );
}

describe('WorkspacesPage', () => {
  afterEach(() => {
    cleanup();
  });

  beforeEach(() => {
    vi.clearAllMocks();
    setMockAuth();
    vi.mocked(api.listWorkspaces).mockResolvedValue(workspaces);
    vi.mocked(api.createWorkspace).mockResolvedValue({
      id: 'ws-3',
      name: 'New WS',
      updatedAt: '2026-01-01T00:00:00Z',
    });
  });

  it('loads and lists workspaces', async () => {
    renderPage();

    await waitFor(() => {
      expect(screen.getByText('Available WS')).toBeInTheDocument();
    });
    expect(screen.getByText('Unavailable WS')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: /Available WS/ })).toHaveAttribute(
      'href',
      '/workspaces/ws-1/documents',
    );
    expect(screen.getByText(/利用可能/)).toBeInTheDocument();
    expect(screen.getByText(/ホストが停止されているため利用できません/)).toBeInTheDocument();
  });

  it('creates a workspace', async () => {
    const user = userEvent.setup();
    renderPage();

    await waitFor(() => expect(screen.getByText('Available WS')).toBeInTheDocument());

    await user.type(screen.getByPlaceholderText('新しいワークスペース名'), 'New WS');
    await user.click(screen.getByRole('button', { name: '作成' }));

    await waitFor(() => {
      expect(api.createWorkspace).toHaveBeenCalledWith('access-token', 'New WS');
      expect(api.listWorkspaces).toHaveBeenCalledTimes(2);
    });
  });

  it('shows load and create errors', async () => {
    const user = userEvent.setup();
    vi.mocked(api.listWorkspaces).mockRejectedValue({ code: 'UNKNOWN_ERROR', message: 'fail' });

    renderPage();
    await waitFor(() => expect(screen.getByText('リクエストに失敗しました')).toBeInTheDocument());

    vi.mocked(api.listWorkspaces).mockResolvedValue(workspaces);
    vi.mocked(api.createWorkspace).mockRejectedValue({ code: 'VALIDATION_ERROR', message: 'bad name' });

    await user.type(screen.getByPlaceholderText('新しいワークスペース名'), 'Bad');
    await user.click(screen.getByRole('button', { name: '作成' }));

    await waitFor(() => expect(screen.getByText('入力値が不正です')).toBeInTheDocument());
  });

  it('skips load without access token', async () => {
    setMockAuth({ accessToken: null });
    renderPage();
    expect(api.listWorkspaces).not.toHaveBeenCalled();
  });

  it('skips create with blank workspace name', async () => {
    renderPage();
    await waitFor(() => expect(screen.getByText('Available WS')).toBeInTheDocument());

    const input = screen.getByPlaceholderText('新しいワークスペース名');
    fireEvent.change(input, { target: { value: '   ' } });
    fireEvent.submit(input.closest('form')!);
    expect(api.createWorkspace).not.toHaveBeenCalled();
  });
});
