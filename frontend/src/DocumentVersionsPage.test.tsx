import { cleanup, screen, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { Route, Routes } from 'react-router-dom';
import DocumentVersionsPage from './DocumentVersionsPage';
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
  id: 'd1',
  workspaceId: 'ws-1',
  title: 'Doc Title',
  content: 'body',
  updatedBy: 'u1',
  updatedAt: '2026-01-01T00:00:00Z',
};
const versions = [
  {
    id: 'v1',
    documentId: 'd1',
    versionType: 'manual_save' as const,
    createdBy: 'u1',
    createdAt: '2026-01-01T00:00:00Z',
  },
];

function renderPage(initialEntry = '/workspaces/ws-1/documents/d1/versions') {
  return renderWithProviders(
    <Routes>
      <Route
        path="/workspaces/:workspaceId/documents/:documentId/versions"
        element={<DocumentVersionsPage />}
      />
    </Routes>,
    { router: { initialEntries: [initialEntry] } },
  );
}

describe('DocumentVersionsPage', () => {
  afterEach(() => {
    cleanup();
  });

  beforeEach(() => {
    vi.clearAllMocks();
    setMockAuth();
    vi.mocked(api.getWorkspace).mockResolvedValue(workspace);
    vi.mocked(api.getDocument).mockResolvedValue(document);
    vi.mocked(api.listVersions).mockResolvedValue(versions);
  });

  it('loads versions', async () => {
    renderPage();

    await waitFor(() => {
      expect(screen.getByRole('heading', { name: '編集履歴' })).toBeInTheDocument();
    });
    expect(screen.getByRole('link', { name: /Doc Title/ })).toBeInTheDocument();
    expect(screen.getByText('Workspace')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: /保存/ })).toHaveAttribute(
      'href',
      '/workspaces/ws-1/documents/d1/versions/v1',
    );
  });

  it('shows empty state', async () => {
    vi.mocked(api.listVersions).mockResolvedValue([]);
    renderPage();

    await waitFor(() => {
      expect(screen.getByText('保存された履歴はまだありません')).toBeInTheDocument();
    });
  });

  it('shows mismatch and fetch errors', async () => {
    vi.mocked(api.getDocument).mockResolvedValue({ ...document, workspaceId: 'other' });
    renderPage();
    await waitFor(() => {
      expect(screen.getByText('ドキュメントがワークスペースに属していません')).toBeInTheDocument();
    });

    vi.mocked(api.getDocument).mockResolvedValue(document);
    vi.mocked(api.getWorkspace).mockRejectedValue({ code: 'WORKSPACE_NOT_FOUND', message: 'missing' });
    renderPage();
    await waitFor(() => {
      expect(screen.getByText('ワークスペースが見つかりません')).toBeInTheDocument();
    });
  });

  it('skips load without access token', async () => {
    setMockAuth({ accessToken: null });
    renderPage();
    expect(api.getWorkspace).not.toHaveBeenCalled();
  });
});
