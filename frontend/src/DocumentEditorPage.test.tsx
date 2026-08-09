import { act, cleanup, fireEvent, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { Route, Routes, MemoryRouter } from 'react-router-dom';
import { AuthProvider } from './auth';
import DocumentEditorPage from './DocumentEditorPage';
import { mockAuthState, setMockAuth } from './test/mockAuth';
import { renderWithProviders } from './test/render';
import * as api from './api';

const wsHandlers: Record<string, ReturnType<typeof vi.fn>> = {};
let mockSendEdit = vi.fn();
let mockWsClose = vi.fn();
let capturedWsOptions: { getAccessToken: () => string | null } | null = null;

vi.mock('./auth', () => ({
  useAuth: () => mockAuthState,
  AuthProvider: ({ children }: { children: React.ReactNode }) => children,
}));

vi.mock('./api');

vi.mock('./documentWs', () => ({
  connectDocumentWebSocket: vi.fn((_documentId, options, handlers) => {
    capturedWsOptions = options;
    for (const [key, handler] of Object.entries(handlers)) {
      if (typeof handler === 'function') {
        wsHandlers[key] = vi.fn(handler);
      }
    }
    mockSendEdit = vi.fn((content: string) => {
      wsHandlers.onSync?.(content);
    });
    mockWsClose = vi.fn();
    return { sendEdit: mockSendEdit, close: mockWsClose };
  }),
}));

const workspace = { id: 'ws-1', name: 'Workspace', role: 'host' as const };
const document = {
  id: 'doc-1',
  workspaceId: 'ws-1',
  title: 'Doc Title',
  content: 'initial content',
  updatedBy: 'u1',
  updatedAt: '2026-01-01T00:00:00Z',
};

const editorRoutes = (
  <>
    <Route path="/workspaces/:workspaceId/documents/:documentId" element={<DocumentEditorPage />} />
    <Route path="/workspaces/:workspaceId/documents" element={<div data-testid="documents-page" />} />
    <Route path="/workspaces" element={<div data-testid="workspaces-page" />} />
    <Route path="/login" element={<div data-testid="login-page" />} />
  </>
);

function renderPage(initialEntry = '/workspaces/ws-1/documents/doc-1') {
  return renderWithProviders(
    <Routes>{editorRoutes}</Routes>,
    { router: { initialEntries: [initialEntry] } },
  );
}

function rerenderPage(
  view: ReturnType<typeof renderPage>,
  initialEntry = '/workspaces/ws-1/documents/doc-1',
) {
  view.rerender(
    <MemoryRouter initialEntries={[initialEntry]}>
      <AuthProvider>
        <Routes>{editorRoutes}</Routes>
      </AuthProvider>
    </MemoryRouter>,
  );
}

async function invokeWsHandler(name: string, ...args: unknown[]) {
  await act(async () => {
    wsHandlers[name]?.(...args);
  });
}

describe.sequential('DocumentEditorPage', () => {
  afterEach(() => {
    cleanup();
    vi.useRealTimers();
  });

  beforeEach(() => {
    vi.clearAllMocks();
    capturedWsOptions = null;
    setMockAuth();
    vi.spyOn(window, 'confirm').mockReturnValue(true);
    vi.mocked(api.getWorkspace).mockResolvedValue(workspace);
    vi.mocked(api.getDocument).mockResolvedValue(document);
    vi.mocked(api.saveManualVersion).mockResolvedValue({
      id: 'v1',
      documentId: 'doc-1',
      content: 'initial content',
      versionType: 'manual_save',
      createdBy: 'u1',
      createdAt: '2026-01-01T00:00:00Z',
    });
    vi.mocked(api.updateDocument).mockResolvedValue({ id: 'doc-1', title: 'Saved Title', updatedAt: '2026-01-02' });
    vi.mocked(api.deleteDocument).mockResolvedValue({ documentId: 'doc-1', deletedAt: '2026-01-02' });
  });

  it('debounces websocket edits and clears pending debounce on rapid edits', async () => {
    renderPage();
    await waitFor(() => expect(screen.getByDisplayValue('Doc Title')).toBeInTheDocument());

    mockSendEdit.mockClear();
    vi.useFakeTimers();
    fireEvent.change(screen.getByTestId('monaco-editor'), { target: { value: 'first edit' } });
    fireEvent.change(screen.getByTestId('monaco-editor'), { target: { value: 'second edit' } });
    expect(mockSendEdit).not.toHaveBeenCalled();
    vi.advanceTimersByTime(300);
    expect(mockSendEdit).toHaveBeenCalledTimes(1);
    expect(mockSendEdit).toHaveBeenCalledWith('second edit');
    vi.useRealTimers();
  });

  it('supports manual save and open status', async () => {
    const user = userEvent.setup();
    renderPage();

    await waitFor(() => expect(screen.getByDisplayValue('Doc Title')).toBeInTheDocument());
    await invokeWsHandler('onOpen');
    await waitFor(() => expect(screen.getByText('接続中')).toBeInTheDocument());

    await user.click(screen.getByRole('button', { name: '手動保存' }));
    await waitFor(() => {
      expect(api.saveManualVersion).toHaveBeenCalledWith('access-token', 'doc-1');
      expect(screen.getByText('手動保存しました')).toBeInTheDocument();
    });
  });

  it('ignores remote updates from current user and applies others', async () => {
    renderPage();

    await waitFor(() => expect(screen.getByDisplayValue('Doc Title')).toBeInTheDocument());

    await invokeWsHandler('onUpdated', { updatedBy: 'u1', title: 'Self', content: 'self content' });
    expect(screen.getByDisplayValue('Doc Title')).toBeInTheDocument();

    await invokeWsHandler('onUpdated', { updatedBy: 'u2', title: 'Remote', content: 'remote content' });
    await waitFor(() => {
      expect(screen.getByDisplayValue('Remote')).toBeInTheDocument();
      expect(screen.getByDisplayValue('remote content')).toBeInTheDocument();
    });

    await invokeWsHandler('onSync', 'sync content');
    await waitFor(() => expect(screen.getByDisplayValue('sync content')).toBeInTheDocument());

    await invokeWsHandler('onRestored', 'restored content');
    await waitFor(() => expect(screen.getByDisplayValue('restored content')).toBeInTheDocument());

    await invokeWsHandler('onEditorJoined', { userId: 'u2', name: 'Bob' });
    await invokeWsHandler('onEditorJoined', { userId: 'u2', name: 'Bob' });
    await invokeWsHandler('onEditorLeft', { userId: 'u2', name: 'Bob' });
    expect(screen.getByText('編集中: なし')).toBeInTheDocument();
  });

  it('handles title save, delete, and ws errors', async () => {
    const user = userEvent.setup();
    vi.mocked(api.saveManualVersion).mockRejectedValue({ code: 'DOCUMENT_NOT_FOUND', message: 'missing' });
    vi.mocked(api.updateDocument).mockRejectedValueOnce({ code: 'VALIDATION_ERROR', message: 'bad' });

    renderPage();

    await waitFor(() => expect(screen.getByDisplayValue('Doc Title')).toBeInTheDocument());

    await user.clear(screen.getByLabelText('ドキュメントタイトル'));
    await user.click(screen.getByRole('button', { name: 'タイトル保存' }));
    expect(api.updateDocument).not.toHaveBeenCalled();

    await user.type(screen.getByLabelText('ドキュメントタイトル'), 'Saved Title');
    await user.click(screen.getByRole('button', { name: 'タイトル保存' }));
    await waitFor(() => expect(screen.getByText('入力値が不正です')).toBeInTheDocument());

    vi.mocked(api.updateDocument).mockResolvedValue({ id: 'doc-1', title: 'Saved Title', updatedAt: '2026-01-02' });
    await user.click(screen.getByRole('button', { name: 'タイトル保存' }));
    await waitFor(() => expect(api.updateDocument).toHaveBeenCalledWith('access-token', 'doc-1', 'Saved Title'));

    await user.click(screen.getByRole('button', { name: '手動保存' }));
    await waitFor(() => expect(screen.getByText('ドキュメントが見つかりません')).toBeInTheDocument());

    vi.spyOn(window, 'confirm').mockReturnValueOnce(false);
    await user.click(screen.getByRole('button', { name: 'ドキュメント削除' }));
    expect(api.deleteDocument).not.toHaveBeenCalled();

    vi.mocked(api.deleteDocument).mockRejectedValueOnce({ code: 'DOCUMENT_NOT_FOUND', message: 'missing' });
    await user.click(screen.getByRole('button', { name: 'ドキュメント削除' }));
    await waitFor(() => expect(screen.getByText('ドキュメントが見つかりません')).toBeInTheDocument());
  });

  it('flushes pending edits on manual save and deletes successfully', async () => {
    renderPage();
    await waitFor(() => expect(screen.getByDisplayValue('Doc Title')).toBeInTheDocument());

    vi.useFakeTimers();
    fireEvent.change(screen.getByTestId('monaco-editor'), { target: { value: 'pending edit' } });
    vi.useRealTimers();

    fireEvent.click(screen.getByRole('button', { name: '手動保存' }));
    await waitFor(() => expect(mockSendEdit).toHaveBeenCalledWith('pending edit'));

    vi.mocked(api.deleteDocument).mockResolvedValueOnce({ documentId: 'doc-1', deletedAt: '2026-01-02' });
    fireEvent.click(screen.getByRole('button', { name: 'ドキュメント削除' }));
    await waitFor(() => expect(screen.getByTestId('documents-page')).toBeInTheDocument());
  });

  it('syncs editors list and skips debounce for remote updates', async () => {
    renderPage();
    await waitFor(() => expect(screen.getByDisplayValue('Doc Title')).toBeInTheDocument());

    await invokeWsHandler('onEditors', [{ userId: 'u2', name: 'Bob' }]);
    expect(screen.getByText('編集中: Bob')).toBeInTheDocument();

    mockSendEdit.mockClear();
    await invokeWsHandler('onSync', 'remote only');
    fireEvent.change(screen.getByTestId('monaco-editor'), { target: { value: 'remote only' } });
    expect(mockSendEdit).not.toHaveBeenCalled();
  });

  it('shows load errors for mismatched workspace and failed fetch', async () => {
    vi.mocked(api.getDocument).mockResolvedValueOnce({ ...document, workspaceId: 'other' });
    const { unmount: unmountMismatch } = renderPage();
    await waitFor(() => {
      expect(screen.getByText('ドキュメントがワークスペースに属していません')).toBeInTheDocument();
    });
    unmountMismatch();

    vi.mocked(api.getDocument).mockRejectedValue({ code: 'DOCUMENT_NOT_FOUND', message: 'missing' });
    renderPage();
    await waitFor(() => {
      expect(screen.getByText('ドキュメントが見つかりません')).toBeInTheDocument();
    });
  });

  it('handles terminal websocket events and reconnect state', async () => {
    renderPage();
    await waitFor(() => expect(screen.getByDisplayValue('Doc Title')).toBeInTheDocument());

    await invokeWsHandler('onReconnecting');
    expect(screen.getByText('再接続中...')).toBeInTheDocument();

    await invokeWsHandler('onError', 'WebSocket error');
    expect(screen.getByText('WebSocket error')).toBeInTheDocument();

    await invokeWsHandler('onClose');
    expect(screen.getByText('切断')).toBeInTheDocument();
  });

  it('uses access token ref for websocket auth', async () => {
    const view = renderPage();
    await waitFor(() => expect(screen.getByDisplayValue('Doc Title')).toBeInTheDocument());
    expect(capturedWsOptions?.getAccessToken()).toBe('access-token');

    setMockAuth({ accessToken: 'rotated-token' });
    rerenderPage(view);
    await waitFor(() => expect(capturedWsOptions?.getAccessToken()).toBe('rotated-token'));
  });

  it('skips load without access token', async () => {
    setMockAuth({ accessToken: null });
    renderPage();
    expect(api.getDocument).not.toHaveBeenCalled();
  });

  it('ignores load results after unmount', async () => {
    let resolveDocument: (value: typeof document) => void = () => {};
    vi.mocked(api.getDocument).mockImplementation(
      () =>
        new Promise((resolve) => {
          resolveDocument = resolve;
        }),
    );
    vi.mocked(api.getWorkspace).mockImplementation(
      () =>
        new Promise((resolve) => {
          resolve(workspace);
        }),
    );

    const { unmount } = renderPage();
    unmount();
    resolveDocument(document);
    await Promise.resolve();
  });

  it('applies partial remote updates and skips debounce after remote sync', async () => {
    renderPage();
    await waitFor(() => expect(screen.getByDisplayValue('Doc Title')).toBeInTheDocument());

    await invokeWsHandler('onUpdated', { updatedBy: 'u2', content: 'content only' });
    await waitFor(() => expect(screen.getByDisplayValue('content only')).toBeInTheDocument());
    expect(screen.getByDisplayValue('Doc Title')).toBeInTheDocument();

    await invokeWsHandler('onUpdated', { updatedBy: 'u2', title: 'Title only' });
    await waitFor(() => expect(screen.getByDisplayValue('Title only')).toBeInTheDocument());

    mockSendEdit.mockClear();
    await act(async () => {
      wsHandlers.onSync?.('synced content');
      fireEvent.change(screen.getByTestId('monaco-editor'), { target: { value: 'synced content' } });
    });
    expect(mockSendEdit).not.toHaveBeenCalled();
  });

  it('skips save actions without access token after rerender', async () => {
    const user = userEvent.setup();
    const view = renderPage();
    await waitFor(() => expect(screen.getByDisplayValue('Doc Title')).toBeInTheDocument());

    setMockAuth({ accessToken: null });
    rerenderPage(view);

    await user.click(screen.getByRole('button', { name: '手動保存' }));
    await user.type(screen.getByLabelText('ドキュメントタイトル'), 'New Title');
    await user.click(screen.getByRole('button', { name: 'タイトル保存' }));
    await user.click(screen.getByRole('button', { name: 'ドキュメント削除' }));

    expect(api.saveManualVersion).not.toHaveBeenCalled();
    expect(api.updateDocument).not.toHaveBeenCalled();
    expect(api.deleteDocument).not.toHaveBeenCalled();
  });

  it('ignores load errors after unmount', async () => {
    let rejectDocument: (reason?: unknown) => void = () => {};
    vi.mocked(api.getDocument).mockImplementation(
      () =>
        new Promise((_resolve, reject) => {
          rejectDocument = reject;
        }),
    );

    const { unmount } = renderPage();
    unmount();
    rejectDocument({ code: 'DOCUMENT_NOT_FOUND', message: 'missing' });
    await Promise.resolve();
  });

  it.each([
    ['onDeleted', 'documents-page'],
    ['onWorkspaceDeleted', 'workspaces-page'],
    ['onWorkspaceHostSuspended', 'workspaces-page'],
    ['onWorkspaceHostDeleted', 'workspaces-page'],
    ['onAccountSuspended', 'login-page'],
    ['onAccountDeleted', 'login-page'],
    ['onMemberRemoved', 'workspaces-page'],
  ] as const)('handles terminal websocket event %s', async (handlerName, redirectTestId) => {
    renderPage();
    await waitFor(() => expect(screen.getByDisplayValue('Doc Title')).toBeInTheDocument());
    await invokeWsHandler(handlerName);
    await waitFor(() => expect(screen.getByTestId(redirectTestId)).toBeInTheDocument());
  });
});
