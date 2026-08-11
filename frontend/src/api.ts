export type ApiError = {
  code: string;
  message: string;
};

export type AuthUser = {
  id: string;
  name: string;
  email: string;
  status: string;
};

type ApiResponse<T> = {
  data: T;
};

type ApiErrorResponse = {
  error: ApiError;
};

type AuthHandlers = {
  onAccessTokenRefreshed: (accessToken: string, expiresAt: string) => void;
  onAuthFailed: () => void;
};

type ApiOptions = {
  accessToken?: string | null;
  skipAuthRetry?: boolean;
};

type TokenRefreshResult = {
  accessToken: string;
  expiresAt: string;
  csrfToken?: string;
  tokenType?: string;
};

let storedCsrfToken: string | null = null;

const SESSION_HINT_KEY = 'notehub_session_hint';

function markSessionHint() {
  if (typeof sessionStorage !== 'undefined') {
    sessionStorage.setItem(SESSION_HINT_KEY, '1');
  }
}

function clearSessionHint() {
  if (typeof sessionStorage !== 'undefined') {
    sessionStorage.removeItem(SESSION_HINT_KEY);
  }
}

export function hasSessionHint(): boolean {
  return typeof sessionStorage !== 'undefined' && sessionStorage.getItem(SESSION_HINT_KEY) === '1';
}

function setStoredCsrfToken(token: string | null) {
  storedCsrfToken = token;
}

function applyCsrfToken(token?: string) {
  if (token) {
    setStoredCsrfToken(token);
  }
}

function clearStoredAuthState() {
  setStoredCsrfToken(null);
  clearSessionHint();
  stopProactiveRefresh();
}

type AccessTokenListener = (accessToken: string, expiresAt: string) => void;

const PROACTIVE_REFRESH_MARGIN_MS = 60_000;

function resolveApiUrl(path: string): string {
  const base = import.meta.env.VITE_API_BASE_URL?.replace(/\/$/, '') ?? '';
  return `${base}${path}`;
}

let authHandlers: AuthHandlers | null = null;
let refreshPromise: Promise<TokenRefreshResult> | null = null;
let proactiveRefreshTimer: number | null = null;
let accessTokenExpiresAt: string | null = null;
const accessTokenListeners = new Set<AccessTokenListener>();

export function configureAuthHandlers(handlers: AuthHandlers | null) {
  authHandlers = handlers;
  if (!handlers) {
    clearStoredAuthState();
  }
}

export function subscribeAccessTokenRefresh(listener: AccessTokenListener): () => void {
  accessTokenListeners.add(listener);
  return () => {
    accessTokenListeners.delete(listener);
  };
}

export function stopProactiveRefresh() {
  if (proactiveRefreshTimer !== null) {
    window.clearTimeout(proactiveRefreshTimer);
    proactiveRefreshTimer = null;
  }
  accessTokenExpiresAt = null;
}

function parseExpiresAt(expiresAt: string): number | null {
  const expiryMs = Date.parse(expiresAt);
  return Number.isNaN(expiryMs) ? null : expiryMs;
}

export function isAccessTokenExpiredOrExpiringSoon(marginMs = PROACTIVE_REFRESH_MARGIN_MS): boolean {
  if (!accessTokenExpiresAt) {
    return true;
  }
  const expiryMs = parseExpiresAt(accessTokenExpiresAt);
  if (expiryMs === null) {
    return true;
  }
  return Date.now() >= expiryMs - marginMs;
}

function scheduleProactiveRefresh(expiresAt: string) {
  stopProactiveRefresh();
  accessTokenExpiresAt = expiresAt;

  const expiryMs = parseExpiresAt(expiresAt);
  if (expiryMs === null) {
    return;
  }

  const delay = expiryMs - Date.now() - PROACTIVE_REFRESH_MARGIN_MS;
  proactiveRefreshTimer = window.setTimeout(() => {
    proactiveRefreshTimer = null;
    void refreshAccessToken().catch(() => {
      // onAuthFailed is handled inside refreshAccessToken
    });
  }, Math.max(delay, 0));
}

function notifyAccessTokenRefresh(result: TokenRefreshResult) {
  authHandlers?.onAccessTokenRefreshed(result.accessToken, result.expiresAt);
  for (const listener of accessTokenListeners) {
    listener(result.accessToken, result.expiresAt);
  }
  scheduleProactiveRefresh(result.expiresAt);
}

function applyTokenRefreshResult(result: TokenRefreshResult): TokenRefreshResult {
  applyCsrfToken(result.csrfToken);
  markSessionHint();
  notifyAccessTokenRefresh(result);
  return result;
}

function isRefreshableAuthError(status: number, code: string | undefined): boolean {
  return status === 401 && code === 'ACCESS_TOKEN_EXPIRED';
}

function performTokenRefresh(): Promise<TokenRefreshResult> {
  if (refreshPromise) {
    return refreshPromise;
  }

  refreshPromise = (async () => {
    try {
      const result = await request<TokenRefreshResult & { tokenType: string }>(
        '/api/auth/refresh',
        {
          method: 'POST',
        },
        { skipAuthRetry: true },
      );
      return applyTokenRefreshResult(result);
    } catch {
      clearStoredAuthState();
      authHandlers?.onAuthFailed();
      throw {
        code: 'ACCESS_TOKEN_EXPIRED',
        message: 'ログインの有効期限が切れました再度ログインしてください',
        status: 401,
      } satisfies ApiError & { status: number };
    } finally {
      refreshPromise = null;
    }
  })();

  return refreshPromise;
}

async function refreshAccessToken(): Promise<string> {
  const result = await performTokenRefresh();
  return result.accessToken;
}

export async function ensureFreshAccessToken(currentToken: string | null): Promise<string | null> {
  if (currentToken && !isAccessTokenExpiredOrExpiringSoon(0)) {
    return currentToken;
  }
  try {
    return await refreshAccessToken();
  } catch {
    return null;
  }
}

async function request<T>(
  path: string,
  init: RequestInit = {},
  options: ApiOptions = {},
  retried = false,
): Promise<T> {
  const headers = new Headers(init.headers);
  const token = options.accessToken ?? null;
  if (token) {
    headers.set('Authorization', `Bearer ${token}`);
  }
  if (init.body && !headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json');
  }

  const response = await fetch(resolveApiUrl(path), {
    ...init,
    headers,
    credentials: 'include',
  });

  const payload = await response.json().catch(() => null);
  if (!response.ok) {
    const error = (payload as ApiErrorResponse | null)?.error;
    const apiError = {
      code: error?.code ?? 'UNKNOWN_ERROR',
      message: error?.message ?? 'リクエストに失敗しました',
      status: response.status,
    } satisfies ApiError & { status: number };

    if (
      !options.skipAuthRetry &&
      !retried &&
      token &&
      isRefreshableAuthError(response.status, error?.code)
    ) {
      const nextToken = await refreshAccessToken();
      return request<T>(path, init, { ...options, accessToken: nextToken }, true);
    }

    throw apiError;
  }

  return (payload as ApiResponse<T>).data;
}

export async function api<T>(path: string, init: RequestInit = {}, accessToken?: string | null): Promise<T> {
  return request<T>(path, init, { accessToken });
}

export function getCsrfToken(): string {
  if (storedCsrfToken) {
    return storedCsrfToken;
  }
  const match = document.cookie.match(/(?:^|;\s*)notehub_csrf_token=([^;]*)/);
  return match ? decodeURIComponent(match[1]) : '';
}

export type RegisterResult = {
  user: AuthUser;
  createdAt: string;
};

export type LoginResult = {
  user: AuthUser;
  accessToken: string;
  tokenType: string;
  expiresAt: string;
  csrfToken?: string;
};

export type RefreshResult = {
  accessToken: string;
  tokenType: string;
  expiresAt: string;
  csrfToken?: string;
};

export function register(name: string, email: string, password: string) {
  return api<RegisterResult>('/api/auth/register', {
    method: 'POST',
    body: JSON.stringify({ name, email, password }),
  });
}

export async function login(email: string, password: string) {
  const result = await api<LoginResult>('/api/auth/login', {
    method: 'POST',
    body: JSON.stringify({ email, password }),
  });
  applyTokenRefreshResult(result);
  return result;
}

export async function refresh(): Promise<RefreshResult> {
  const result = await performTokenRefresh();
  return {
    accessToken: result.accessToken,
    tokenType: result.tokenType ?? 'Bearer',
    expiresAt: result.expiresAt,
    csrfToken: result.csrfToken,
  };
}

export async function logout(accessToken: string) {
  const headers: Record<string, string> = {
    Authorization: `Bearer ${accessToken}`,
  };
  const csrfToken = getCsrfToken();
  if (csrfToken) {
    headers['X-CSRF-Token'] = csrfToken;
  }
  try {
    return await api<{ message: string }>('/api/auth/logout', {
      method: 'POST',
      headers,
    });
  } finally {
    clearStoredAuthState();
  }
}

export function me(accessToken: string) {
  return api<{ user: AuthUser }>('/api/me', {
    headers: {
      Authorization: `Bearer ${accessToken}`,
    },
  });
}

export type WorkspaceListItem = {
  id: string;
  name: string;
  hostId: string;
  hostName: string;
  role: string;
  isAvailable: boolean;
  unavailableReason: string;
  updatedAt: string;
};

export type DocumentListItem = {
  id: string;
  title: string;
  updatedBy: string;
  updatedAt: string;
};

export type WorkspaceDetail = {
  id: string;
  name: string;
  role: string;
};

export type DocumentDetail = {
  id: string;
  workspaceId: string;
  title: string;
  content: string;
  updatedBy: string;
  updatedAt: string;
};

export function listWorkspaces(accessToken: string) {
  return api<WorkspaceListItem[]>('/api/workspaces', {}, accessToken);
}

export function createWorkspace(accessToken: string, name: string) {
  return api<{ id: string; name: string }>(
    '/api/workspaces',
    {
      method: 'POST',
      body: JSON.stringify({ name }),
    },
    accessToken,
  );
}

export function getWorkspace(accessToken: string, workspaceId: string) {
  return api<WorkspaceDetail>(`/api/workspaces/${workspaceId}`, {}, accessToken);
}

export function updateWorkspace(accessToken: string, workspaceId: string, name: string) {
  return api<{ id: string; name: string; updatedAt: string }>(
    `/api/workspaces/${workspaceId}`,
    {
      method: 'PATCH',
      body: JSON.stringify({ name }),
    },
    accessToken,
  );
}

export function deleteWorkspace(accessToken: string, workspaceId: string, reason: string) {
  return api<{ workspaceId: string; deletedAt: string }>(
    `/api/workspaces/${workspaceId}`,
    {
      method: 'DELETE',
      body: JSON.stringify({ reason }),
    },
    accessToken,
  );
}

export function getDocument(accessToken: string, documentId: string) {
  return api<DocumentDetail>(`/api/documents/${documentId}`, {}, accessToken);
}

export function listDocuments(accessToken: string, workspaceId: string) {
  return api<DocumentListItem[]>(`/api/workspaces/${workspaceId}/documents`, {}, accessToken);
}

export function createDocument(accessToken: string, workspaceId: string, title: string, content = '') {
  return api<{ id: string; title: string }>(
    `/api/workspaces/${workspaceId}/documents`,
    {
      method: 'POST',
      body: JSON.stringify({ title, content }),
    },
    accessToken,
  );
}

export function updateDocument(accessToken: string, documentId: string, title: string) {
  return api<{ id: string; title: string; updatedAt: string }>(
    `/api/documents/${documentId}`,
    {
      method: 'PATCH',
      body: JSON.stringify({ title }),
    },
    accessToken,
  );
}

export function deleteDocument(accessToken: string, documentId: string) {
  return api<{ documentId: string; deletedAt: string }>(
    `/api/documents/${documentId}`,
    { method: 'DELETE' },
    accessToken,
  );
}

export type SearchUserResult = {
  id: string;
  email: string;
  name: string;
  status: string;
};

export type MemberListItem = {
  userId: string;
  name: string;
  email: string;
  status: string;
  role: string;
  joinedAt: string;
};

export function searchUser(accessToken: string, workspaceId: string, email: string) {
  const params = new URLSearchParams({ email });
  return api<SearchUserResult>(`/api/workspaces/${workspaceId}/users/search?${params}`, {}, accessToken);
}

export function listMembers(accessToken: string, workspaceId: string) {
  return api<MemberListItem[]>(`/api/workspaces/${workspaceId}/members`, {}, accessToken);
}

export function addMember(accessToken: string, workspaceId: string, userId: string) {
  return api<MemberListItem>(`/api/workspaces/${workspaceId}/members`, {
    method: 'POST',
    body: JSON.stringify({ userId }),
  }, accessToken);
}

export function removeMember(accessToken: string, workspaceId: string, userId: string) {
  return api<Record<string, never>>(`/api/workspaces/${workspaceId}/members/${userId}`, {
    method: 'DELETE',
  }, accessToken);
}

export function suspendMember(accessToken: string, workspaceId: string, userId: string) {
  return api<{ userId: string; status: string; suspendedAt: string }>(
    `/api/workspaces/${workspaceId}/members/${userId}/suspend`,
    { method: 'POST' },
    accessToken,
  );
}

export function reactivateMember(accessToken: string, workspaceId: string, userId: string) {
  return api<{ userId: string; status: string }>(
    `/api/workspaces/${workspaceId}/members/${userId}/reactivate`,
    { method: 'POST' },
    accessToken,
  );
}

export function deleteAccount(accessToken: string, workspaceId: string, userId: string) {
  return api<{ userId: string; status: string; deletedAt: string }>(
    `/api/workspaces/${workspaceId}/members/${userId}/delete-account`,
    { method: 'POST' },
    accessToken,
  );
}

export type VersionListItem = {
  id: string;
  versionType: string;
  createdBy: string;
  createdAt: string;
};

export type VersionDetail = {
  id: string;
  documentId: string;
  content: string;
  versionType: string;
  createdBy: string;
  createdAt: string;
};

export function listVersions(accessToken: string, documentId: string) {
  return api<VersionListItem[]>(`/api/documents/${documentId}/versions`, {}, accessToken);
}

export function getVersion(accessToken: string, documentId: string, versionId: string) {
  return api<VersionDetail>(`/api/documents/${documentId}/versions/${versionId}`, {}, accessToken);
}

export function saveManualVersion(accessToken: string, documentId: string) {
  return api<{ id: string; createdAt: string }>(
    `/api/documents/${documentId}/versions`,
    { method: 'POST' },
    accessToken,
  );
}

export function restoreVersion(accessToken: string, documentId: string, versionId: string) {
  return api<{ documentId: string; versionId: string; restoredAt: string }>(
    `/api/documents/${documentId}/versions/${versionId}/restore`,
    { method: 'POST' },
    accessToken,
  );
}

export function validatePassword(password: string): string | null {
  if (password.length < 8 || password.length > 15) {
    return 'パスワードは8文字以上15文字以下で入力してください';
  }
  if (password.trim() === '') {
    return 'パスワードを入力してください';
  }
  const hasLetter = /[A-Za-z]/.test(password);
  const hasDigit = /\d/.test(password);
  if (!hasLetter || !hasDigit) {
    return 'パスワードは英字と数字をそれぞれ1文字以上含めてください';
  }
  return null;
}
