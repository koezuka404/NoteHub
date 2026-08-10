import { cleanup, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it } from 'vitest';
import HostBadge from './HostBadge';

describe('HostBadge', () => {
  afterEach(() => {
    cleanup();
  });

  it('renders host badge', () => {
    render(<HostBadge role="host" />);
    expect(screen.getByText('オーナー')).toBeInTheDocument();
  });

  it('renders member badge', () => {
    render(<HostBadge role="member" />);
    expect(screen.getByText('メンバー')).toBeInTheDocument();
  });

  it('renders nothing for unknown role', () => {
    const { container } = render(<HostBadge role="guest" />);
    expect(container).toBeEmptyDOMElement();
  });
});
