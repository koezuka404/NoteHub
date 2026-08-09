import { vi } from 'vitest';

export type MockResponse = {
  ok: boolean;
  status?: number;
  data?: unknown;
  error?: { code: string; message: string };
  jsonError?: boolean;
};

export function mockFetch(responses: MockResponse[] | MockResponse) {
  const queue = Array.isArray(responses) ? [...responses] : [responses];
  const fetchMock = vi.fn(async () => {
    const next = queue.shift();
    if (!next) {
      throw new TypeError('Failed to fetch');
    }
    return {
      ok: next.ok,
      status: next.status ?? (next.ok ? 200 : 500),
      json: async () => {
        if (next.jsonError) {
          throw new Error('invalid json');
        }
        if (!next.ok) {
          return { error: next.error ?? { code: 'UNKNOWN_ERROR', message: 'error' } };
        }
        return { data: next.data };
      },
    } as Response;
  });
  vi.stubGlobal('fetch', fetchMock);
  return fetchMock;
}
