import { cleanup, fireEvent, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { Route, Routes } from 'react-router-dom';
import DocumentsPage from './DocumentsPage';
import { mockAuthState, setMockAuth } from './test/mockAuth';
import { renderWithProviders } from './test/render';
import * as api from './api';

const wsHandlers: Record<string, ReturnType<typeof vi.fn>> = {};
let mockWsClose = vi.fn();
let capturedWsOptions: { getAccessToken: () => string | null } | null = null;

vi.mock('./auth', () => ({
  useAuth: () => mockAuthState,
  AuthProvider: ({ children }: { children: React.ReactNode }) => children,
}));

vi.mock('./api');

vi.mock('./workspaceWs', () => ({
  connectWorkspaceWebSocket: vi.fn((_workspaceId, options, handlers) => {
    capturedWsOptions = options;
    wsHandlers.onDocumentCreated = vi.fn(handlers.onDocumentCreated);
    wsHandlers.onDocumentUpdated = vi.fn(handlers.onDocumentUpdated);
    wsHandlers.onDocumentDeleted = vi.fn(handlers.onDocumentDeleted);
    wsHandlers.onError = vi.fn(handlers.onError);
    wsHandlers.onReconnecting = vi.fn(handlers.onReconnecting);
    mockWsClose = vi.fn();
    return { close: mockWsClose };
  }),
}));

const workspace = { id: 'ws-1', name: 'Workspace', role: 'host' as const };
const memberWorkspace = { ...workspace, role: 'member' as const };
const documents = [
  { id: 'd1', title: 'Doc 1', updatedBy: 'u1', updatedAt: '2026-01-01T00:00:00Z' },
  { id: 'd2', title: 'Doc 2', updatedBy: 'u1', updatedAt: '2026-01-02T00:00:00Z' },
];

function renderPage(initialEntry = '/workspaces/ws-1/documents') {
  return renderWithProviders(
    <Routes>
      <Route path="/workspaces/:workspaceId/documents" element={<DocumentsPage />} />
    </Routes>,
    { router: { initialEntries: [initialEntry] } },
  );
}

describe.sequential('DocumentsPage', () => {
  afterEach(() => {
    cleanup();
  });

  beforeEach(() => {
    vi.clearAllMocks();
    capturedWsOptions = null;
    setMockAuth();
    vi.spyOn(window, 'confirm').mockReturnValue(true);
    vi.mocked(api.getWorkspace).mockResolvedValue(workspace);
    vi.mocked(api.listDocuments).mockResolvedValue(documents);
  });

  it('loads workspace and documents then connects websocket', async () => {
    renderPage();

    await waitFor(() => {
      expect(screen.getByRole('heading', { name: 'Workspace' })).toBeInTheDocument();
    });
    expect(screen.getByText('Doc 1')).toBeInTheDocument();
    expect(screen.getByText('Doc 2')).toBeInTheDocument();
    expect(api.getWorkspace).toHaveBeenCalledWith('access-token', 'ws-1');
    expect(api.listDocuments).toHaveBeenCalledWith('access-token', 'ws-1');
    expect(capturedWsOptions?.getAccessToken()).toBe('access-token');
  });

  it('shows load error', async () => {
    vi.mocked(api.getWorkspace).mockRejectedValue({ code: 'WORKSPACE_NOT_FOUND', message: 'missing' });

    renderPage();

    await waitFor(() => {
      expect(screen.getByText('ワークスペースが見つかりません')).toBeInTheDocument();
    });
  });

  it('creates a document and navigates to editor', async () => {
    const user = userEvent.setup();
    vi.mocked(api.createDocument).mockResolvedValue({ id: 'd3', title: 'New Doc' });

    renderPage();

    await waitFor(() => expect(screen.getByLabelText('新しいドキュメント名')).toBeInTheDocument());

    await user.type(screen.getByLabelText('新しいドキュメント名'), 'New Doc');
    await user.click(screen.getByRole('button', { name: '作成' }));

    await waitFor(() => {
      expect(api.createDocument).toHaveBeenCalledWith('access-token', 'ws-1', 'New Doc');
    });
  });

  it('handles create errors and skips empty title', async () => {
    const user = userEvent.setup();
    vi.mocked(api.createDocument).mockRejectedValue({ code: 'VALIDATION_ERROR', message: 'bad' });

    renderPage();

    await waitFor(() => expect(screen.getByLabelText('新しいドキュメント名')).toBeInTheDocument());
    await user.click(screen.getByRole('button', { name: '作成' }));
    expect(api.createDocument).not.toHaveBeenCalled();

    await user.type(screen.getByLabelText('新しいドキュメント名'), 'New Doc');
    await user.click(screen.getByRole('button', { name: '作成' }));

    await waitFor(() => {
      expect(screen.getByText('入力値が不正です')).toBeInTheDocument();
    });
  });

  it('deletes a document and handles cancel/error', async () => {
    const user = userEvent.setup();
    vi.mocked(api.deleteDocument).mockResolvedValue({ documentId: 'd1', deletedAt: '2026-01-01' });

    renderPage();

    await waitFor(() => expect(screen.getByText('Doc 1')).toBeInTheDocument());

    vi.spyOn(window, 'confirm').mockReturnValueOnce(false);
    await user.click(screen.getAllByRole('button', { name: '削除' })[0]);
    expect(api.deleteDocument).not.toHaveBeenCalled();

    await user.click(screen.getAllByRole('button', { name: '削除' })[0]);
    await waitFor(() => expect(api.deleteDocument).toHaveBeenCalledWith('access-token', 'd1'));
    await waitFor(() => expect(screen.queryByText('Doc 1')).not.toBeInTheDocument());

    vi.mocked(api.deleteDocument).mockRejectedValueOnce({ code: 'DOCUMENT_NOT_FOUND', message: 'missing' });
    await user.click(screen.getByRole('button', { name: '削除' }));
    await waitFor(() => {
      expect(screen.getByText('ドキュメントが見つかりません')).toBeInTheDocument();
    });
  });

  it('updates and deletes workspace as host', async () => {
    const user = userEvent.setup();
    vi.mocked(api.updateWorkspace).mockResolvedValue({
      id: 'ws-1',
      name: 'Renamed',
      updatedAt: '2026-01-03T00:00:00Z',
    });
    vi.mocked(api.deleteWorkspace).mockResolvedValue({
      workspaceId: 'ws-1',
      deletedAt: '2026-01-03T00:00:00Z',
    });

    renderPage();

    await waitFor(() => expect(screen.getByDisplayValue('Workspace')).toBeInTheDocument());

    const nameInput = screen.getByPlaceholderText('ワークスペース名');
    await user.clear(nameInput);
    await user.type(nameInput, 'Renamed');
    await user.click(screen.getByRole('button', { name: '名前を更新' }));

    await waitFor(() => {
      expect(api.updateWorkspace).toHaveBeenCalledWith('access-token', 'ws-1', 'Renamed');
    });

    expect(screen.queryByLabelText('削除理由')).not.toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'ワークスペースを削除' }));
    await user.type(screen.getByLabelText('削除理由'), 'cleanup');
    vi.spyOn(window, 'confirm').mockReturnValueOnce(false);
    await user.click(screen.getByRole('button', { name: '削除する' }));
    expect(api.deleteWorkspace).not.toHaveBeenCalled();

    await user.click(screen.getByRole('button', { name: '削除する' }));
    await waitFor(() => {
      expect(api.deleteWorkspace).toHaveBeenCalledWith('access-token', 'ws-1', 'cleanup');
    });
  });

  it('hides host settings for members and dedupes websocket document events', async () => {
    vi.mocked(api.getWorkspace).mockResolvedValue(memberWorkspace);

    const { unmount } = renderPage();

    await waitFor(() => expect(screen.getByText('Doc 1')).toBeInTheDocument());
    expect(screen.queryByText('ワークスペース設定')).not.toBeInTheDocument();

    const created = { id: 'd1', title: 'Dup', updatedBy: 'u1', updatedAt: '2026-01-01' };
    wsHandlers.onDocumentCreated?.(created);
    wsHandlers.onDocumentCreated?.(created);
    expect(screen.getAllByText('Doc 1').length).toBeGreaterThan(0);

    wsHandlers.onDocumentCreated?.({
      id: 'd3',
      title: 'Created via WS',
      updatedBy: 'u2',
      updatedAt: '2026-01-05',
    });
    await waitFor(() => expect(screen.getByText('Created via WS')).toBeInTheDocument());

    wsHandlers.onDocumentUpdated?.({ id: 'd1', title: 'Updated', updatedBy: 'u2', updatedAt: '2026-01-04' });
    await waitFor(() => expect(screen.getByText('Updated')).toBeInTheDocument());

    wsHandlers.onDocumentDeleted?.('d2');
    await waitFor(() => expect(screen.queryByText('Doc 2')).not.toBeInTheDocument());

    unmount();
    expect(mockWsClose).toHaveBeenCalled();
  });

  it('handles workspace update/delete errors', async () => {
    const user = userEvent.setup();
    vi.mocked(api.updateWorkspace).mockRejectedValue({ code: 'VALIDATION_ERROR', message: 'bad' });
    vi.mocked(api.deleteWorkspace).mockRejectedValue({ code: 'WORKSPACE_NOT_FOUND', message: 'missing' });

    renderPage();

    await waitFor(() => expect(screen.getByDisplayValue('Workspace')).toBeInTheDocument());

    await user.click(screen.getByRole('button', { name: '名前を更新' }));
    await waitFor(() => expect(screen.getByText('入力値が不正です')).toBeInTheDocument());

    await user.click(screen.getByRole('button', { name: 'ワークスペースを削除' }));
    await user.type(screen.getByLabelText('削除理由'), 'cleanup');
    await user.click(screen.getByRole('button', { name: '削除する' }));
    await waitFor(() => {
      expect(screen.getByText('ワークスペースが見つかりません')).toBeInTheDocument();
    });
  });

  it('skips host update/delete with blank inputs', async () => {
    renderPage();
    await waitFor(() => expect(screen.getByDisplayValue('Workspace')).toBeInTheDocument());

    const nameInput = screen.getByPlaceholderText('ワークスペース名');
    fireEvent.change(nameInput, { target: { value: '   ' } });
    fireEvent.submit(nameInput.closest('form')!);
    expect(api.updateWorkspace).not.toHaveBeenCalled();

    fireEvent.click(screen.getByRole('button', { name: 'ワークスペースを削除' }));
    fireEvent.submit(screen.getByLabelText('削除理由').closest('form')!);
    expect(api.deleteWorkspace).not.toHaveBeenCalled();
  });

  it('skips load without access token', async () => {
    setMockAuth({ accessToken: null });
    renderPage();
    expect(api.listDocuments).not.toHaveBeenCalled();
  });

  it('skips create with blank title', async () => {
    renderPage();
    await waitFor(() => expect(screen.getByText('Doc 1')).toBeInTheDocument());

    const input = screen.getByPlaceholderText('新しいドキュメント名');
    fireEvent.change(input, { target: { value: '   ' } });
    fireEvent.submit(input.closest('form')!);
    expect(api.createDocument).not.toHaveBeenCalled();
  });
});
