import { describe, expect, it, vi, beforeEach } from 'vitest';

const renderMock = vi.fn();

vi.mock('react-dom/client', () => ({
  createRoot: vi.fn(() => ({ render: renderMock })),
}));

vi.mock('./App', () => ({
  default: () => null,
}));

vi.mock('./auth', () => ({
  AuthProvider: ({ children }: { children: React.ReactNode }) => children,
}));

vi.mock('./index.css', () => ({}));

describe('main', () => {
  beforeEach(() => {
    document.body.innerHTML = '<div id="root"></div>';
    renderMock.mockClear();
    vi.resetModules();
  });

  it('mounts the application', async () => {
    await import('./main');
    const { createRoot } = await import('react-dom/client');
    expect(createRoot).toHaveBeenCalledWith(document.getElementById('root'));
    expect(renderMock).toHaveBeenCalled();
  });
});
