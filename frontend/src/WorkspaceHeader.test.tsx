import { cleanup, render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { afterEach, describe, expect, it } from 'vitest';
import WorkspaceHeader from './WorkspaceHeader';

function renderHeader(role?: string, activeTab: 'documents' | 'members' = 'documents') {
  return render(
    <MemoryRouter>
      <WorkspaceHeader
        workspaceId="ws-1"
        workspaceName="Workspace"
        role={role}
        activeTab={activeTab}
      />
    </MemoryRouter>,
  );
}

describe('WorkspaceHeader', () => {
  afterEach(() => {
    cleanup();
  });

  it('shows host notice and badge for hosts', () => {
    const { container } = renderHeader('host', 'documents');

    expect(screen.getByText('Workspace')).toBeInTheDocument();
    expect(container.querySelector('.host-badge')).toHaveTextContent('オーナー');
    expect(screen.getByText('ワークスペースの設定変更ができます')).toBeInTheDocument();
    expect(screen.getByText('ドキュメント')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'メンバー' })).toHaveAttribute('href', '/workspaces/ws-1/members');
  });

  it('shows members tab notice for hosts', () => {
    renderHeader('host', 'members');
    expect(screen.getByText('メンバーの招待を行えます')).toBeInTheDocument();
  });

  it('shows member badge without host notice', () => {
    const { container } = renderHeader('member', 'members');

    expect(container.querySelector('.role-badge')).toHaveTextContent('メンバー');
    expect(container.querySelector('.host-notice')).not.toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'ドキュメント' })).toHaveAttribute('href', '/workspaces/ws-1/documents');
  });

  it('omits role badge when role is missing', () => {
    const { container } = renderHeader();

    expect(container.querySelector('.host-badge')).not.toBeInTheDocument();
    expect(container.querySelector('.role-badge')).not.toBeInTheDocument();
    expect(container.querySelector('.host-notice')).not.toBeInTheDocument();
  });
});
