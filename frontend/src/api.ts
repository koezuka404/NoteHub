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

export async function api<T>(path: string, init: RequestInit = {}, accessToken?: string | null): Promise<T> {
  const headers = new Headers(init.headers);
  if (accessToken) {
    headers.set('Authorization', `Bearer ${accessToken}`);
  }
  if (init.body && !headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json');
  }

  const response = await fetch(path, {
    ...init,
    headers,
    credentials: 'include',
  });

  const payload = await response.json().catch(() => null);
  if (!response.ok) {
    const error = (payload as ApiErrorResponse | null)?.error;
    throw {
      code: error?.code ?? 'UNKNOWN_ERROR',
      message: error?.message ?? 'リクエストに失敗しました',
      status: response.status,
    } satisfies ApiError & { status: number };
  }

  return (payload as ApiResponse<T>).data;
}

export function getCsrfToken(): string {
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
};

export function register(name: string, email: string, password: string) {
  return api<RegisterResult>('/api/auth/register', {
    method: 'POST',
    body: JSON.stringify({ name, email, password }),
  });
}

export function login(email: string, password: string) {
  return api<LoginResult>('/api/auth/login', {
    method: 'POST',
    body: JSON.stringify({ email, password }),
  });
}

export function refresh() {
  return api<{ accessToken: string; tokenType: string; expiresAt: string }>('/api/auth/refresh', {
    method: 'POST',
    headers: {
      'X-CSRF-Token': getCsrfToken(),
    },
  });
}

export function logout(accessToken: string) {
  return api<{ message: string }>('/api/auth/logout', {
    method: 'POST',
    headers: {
      Authorization: `Bearer ${accessToken}`,
      'X-CSRF-Token': getCsrfToken(),
    },
  });
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

export function getDocument(accessToken: string, documentId: string) {
  return api<DocumentDetail>(`/api/documents/${documentId}`, {}, accessToken);
}

export function listDocuments(accessToken: string, workspaceId: string) {
  return api<DocumentListItem[]>(`/api/workspaces/${workspaceId}/documents`, {}, accessToken);
}

export function createDocument(accessToken: string, workspaceId: string, title: string) {
  return api<{ id: string; title: string }>(
    `/api/workspaces/${workspaceId}/documents`,
    {
      method: 'POST',
      body: JSON.stringify({ title }),
    },
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
