import { cleanup, fireEvent, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import DocumentVersionDetailPage from './DocumentVersionDetailPage';
import { AuthProvider } from './auth';
import { mockAuthState, setMockAuth } from './test/mockAuth';
import { renderWithProviders } from './test/render';
import * as api from './api';

vi.mock('./auth', () => ({
  useAuth: () => mockAuthState,
  AuthProvider: ({ children }: { children: React.ReactNode }) => children,
}));

vi.mock('./api');

const workspace = { id: 'ws-1', name: 'Workspace', role: 'host' as const };
const document = {
  id: 'doc-1',
  workspaceId: 'ws-1',
  title: 'Doc Title',
  content: 'current content',
  updatedBy: 'u1',
  updatedAt: '2026-01-01T00:00:00Z',
};
const version = {
  id: 'v1',
  documentId: 'doc-1',
  content: 'version content',
  versionType: 'manual_save',
  createdBy: 'u1',
  createdAt: '2026-01-01T00:00:00Z',
};
const members = [{ userId: 'u1', name: 'Alice', email: 'a@b.com', status: 'active', role: 'host', joinedAt: '2026-01-01' }];

function renderPage(initialEntry = '/workspaces/ws-1/documents/doc-1/versions/v1') {
  return renderWithProviders(
    <Routes>
      <Route
        path="/workspaces/:workspaceId/documents/:documentId/versions/:versionId"
        element={<DocumentVersionDetailPage />}
      />
    </Routes>,
    { router: { initialEntries: [initialEntry] } },
  );
}

function rerenderPage(view: ReturnType<typeof renderPage>) {
  view.rerender(
    <MemoryRouter initialEntries={['/workspaces/ws-1/documents/doc-1/versions/v1']}>
      <AuthProvider>
        <Routes>
          <Route
            path="/workspaces/:workspaceId/documents/:documentId/versions/:versionId"
            element={<DocumentVersionDetailPage />}
          />
        </Routes>
      </AuthProvider>
    </MemoryRouter>,
  );
}

describe.sequential('DocumentVersionDetailPage', () => {
  afterEach(() => {
    cleanup();
    vi.useRealTimers();
  });

  beforeEach(() => {
    vi.clearAllMocks();
    setMockAuth();
    vi.spyOn(window, 'confirm').mockReturnValue(true);
    vi.mocked(api.getWorkspace).mockResolvedValue(workspace);
    vi.mocked(api.getDocument).mockResolvedValue(document);
    vi.mocked(api.getVersion).mockResolvedValue(version);
    vi.mocked(api.listMembers).mockResolvedValue(members);
    vi.mocked(api.restoreVersion).mockResolvedValue({
      documentId: 'doc-1',
      versionId: 'v1',
      restoredAt: '2026-01-02T00:00:00Z',
    });
  });

  it('loads version detail', async () => {
    renderPage();

    await waitFor(() => {
      expect(screen.getByText('保存')).toBeInTheDocument();
      expect(screen.getByDisplayValue('version content')).toBeInTheDocument();
      expect(screen.getByText(/Doc Title/)).toBeInTheDocument();
      expect(screen.getByText(/実施者: Alice/)).toBeInTheDocument();
    });
  });

  it('shows load error for mismatched workspace and failed fetch', async () => {
    vi.mocked(api.getDocument).mockResolvedValueOnce({ ...document, workspaceId: 'other' });
    const { unmount: unmountMismatch } = renderPage();
    await waitFor(() => {
      expect(screen.getByText('ドキュメントがワークスペースに属していません')).toBeInTheDocument();
    });
    unmountMismatch();

    vi.mocked(api.getVersion).mockRejectedValue({ code: 'VERSION_NOT_FOUND', message: 'missing' });
    renderPage();
    await waitFor(() => {
      expect(screen.getByText('編集履歴が見つかりません')).toBeInTheDocument();
    });
  });

  it('restores version on confirm and navigates after success', async () => {
    vi.useFakeTimers({ shouldAdvanceTime: true });
    const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime });

    renderPage();

    await waitFor(() => expect(screen.getByRole('button', { name: 'この履歴で復元' })).toBeInTheDocument());

    vi.spyOn(window, 'confirm').mockReturnValueOnce(false);
    await user.click(screen.getByRole('button', { name: 'この履歴で復元' }));
    expect(api.restoreVersion).not.toHaveBeenCalled();

    await user.click(screen.getByRole('button', { name: 'この履歴で復元' }));
    await waitFor(() => {
      expect(api.restoreVersion).toHaveBeenCalledWith('access-token', 'doc-1', 'v1');
      expect(screen.getByText('復元しましたエディタに戻ります')).toBeInTheDocument();
    });

    vi.advanceTimersByTime(800);
  });

  it('shows restore failure', async () => {
    const user = userEvent.setup();
    vi.mocked(api.restoreVersion).mockRejectedValue({ code: 'VERSION_NOT_FOUND', message: 'missing' });

    renderPage();

    await waitFor(() => expect(screen.getByRole('button', { name: 'この履歴で復元' })).toBeInTheDocument());
    await user.click(screen.getByRole('button', { name: 'この履歴で復元' }));

    await waitFor(() => {
      expect(screen.getByText('編集履歴が見つかりません')).toBeInTheDocument();
    });
  });

  it('skips load without access token', async () => {
    setMockAuth({ accessToken: null });
    renderPage();
    await waitFor(() => expect(screen.queryByText('保存')).not.toBeInTheDocument());
    expect(api.getVersion).not.toHaveBeenCalled();
  });

  it('skips restore without access token after rerender', async () => {
    const user = userEvent.setup();
    const view = renderPage();
    await waitFor(() => expect(screen.getByRole('button', { name: 'この履歴で復元' })).toBeInTheDocument());

    setMockAuth({ accessToken: null });
    rerenderPage(view);
    await user.click(screen.getByRole('button', { name: 'この履歴で復元' }));
    expect(api.restoreVersion).not.toHaveBeenCalled();
  });

  it('skips restore when confirm is cancelled', async () => {
    const user = userEvent.setup();
    vi.spyOn(window, 'confirm').mockReturnValue(false);
    renderPage();

    await waitFor(() => expect(screen.getByRole('button', { name: 'この履歴で復元' })).toBeInTheDocument());
    await user.click(screen.getByRole('button', { name: 'この履歴で復元' }));
    expect(api.restoreVersion).not.toHaveBeenCalled();
  });
});
