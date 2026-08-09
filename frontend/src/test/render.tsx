import type { ReactElement } from 'react';
import { render, type RenderOptions } from '@testing-library/react';
import { MemoryRouter, type MemoryRouterProps } from 'react-router-dom';
import { AuthProvider } from '../auth';

type Options = RenderOptions & {
  router?: MemoryRouterProps;
  withAuth?: boolean;
};

export function renderWithProviders(ui: ReactElement, options: Options = {}) {
  const { router, withAuth = true, ...renderOptions } = options;

  const content = withAuth ? <AuthProvider>{ui}</AuthProvider> : ui;

  return render(
    <MemoryRouter {...router}>{content}</MemoryRouter>,
    renderOptions,
  );
}
