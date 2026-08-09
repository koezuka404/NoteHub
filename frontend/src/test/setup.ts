import '@testing-library/jest-dom/vitest';
import { createElement } from 'react';
import { afterEach, vi } from 'vitest';
import { setMockAuth } from './mockAuth';

afterEach(() => {
  setMockAuth();
});

vi.mock('@monaco-editor/react', () => ({
  default: ({
    value,
    onChange,
    options,
  }: {
    value: string;
    onChange?: (value: string) => void;
    options?: { readOnly?: boolean };
  }) => {
    const readOnly = options?.readOnly ?? false;
    return createElement('textarea', {
      'data-testid': 'monaco-editor',
      value,
      readOnly,
      onChange: readOnly
        ? undefined
        : (event: Event & { target: HTMLTextAreaElement }) => {
            const value = event.target.value;
            onChange?.(value === '__undefined__' ? undefined : value);
          },
    });
  },
}));
