import { cleanup, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it } from 'vitest';
import HostPanel from './HostPanel';

describe('HostPanel', () => {
  afterEach(() => {
    cleanup();
  });

  it('renders title, description, and children', () => {
    render(
      <HostPanel title="メンバーを招待" description="説明文">
        <button type="button">招待</button>
      </HostPanel>,
    );

    expect(screen.getByLabelText('メンバーを招待')).toBeInTheDocument();
    expect(screen.getByText('メンバーを招待')).toBeInTheDocument();
    expect(screen.getByText('説明文')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '招待' })).toBeInTheDocument();
  });

  it('omits description when not provided', () => {
    render(
      <HostPanel title="設定">
        <p>content</p>
      </HostPanel>,
    );

    expect(screen.queryByText('説明文')).not.toBeInTheDocument();
    expect(screen.getByText('content')).toBeInTheDocument();
  });
});
