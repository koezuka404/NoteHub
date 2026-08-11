import type { ApiError } from './api';

const ERROR_MESSAGES: Record<string, string> = {
  UNKNOWN_ERROR: 'リクエストに失敗しました',
  INVALID_REQUEST: 'リクエスト形式が不正です',
  VALIDATION_ERROR: '入力値が不正です',
  PASSWORD_INVALID: 'パスワードの入力内容を確認してください',
  EMAIL_ALREADY_EXISTS: 'このメールアドレスは既に登録されています',
  INVALID_CREDENTIALS: 'メールアドレスまたはパスワードが正しくありません',
  LOGIN_RATE_LIMITED: '時間を空けて再度お試しください',
  REFRESH_TOKEN_REQUIRED: 'ログインが必要です',
  REFRESH_TOKEN_EXPIRED: 'ログインの有効期限が切れました再度ログインしてください',
  REFRESH_TOKEN_INVALID: 'ログイン情報が無効です再度ログインしてください',
  REFRESH_TOKEN_REVOKED: 'ログイン状態が無効になりました再度ログインしてください',
  TOKEN_OWNER_MISMATCH: 'トークンの所有者が一致しません',
  CSRF_TOKEN_INVALID: 'CSRFトークンが不正です',
  CSRF_TOKEN_REQUIRED: 'CSRFトークンが必要です',
  ORIGIN_NOT_ALLOWED: 'リクエスト元が許可されていません',
  AUTH_SERVICE_UNAVAILABLE: '認証サービスを利用できません',
  ACCESS_TOKEN_INVALID: 'ログイン情報が無効です再度ログインしてください',
  ACCESS_TOKEN_REQUIRED: 'ログインが必要です',
  ACCESS_TOKEN_EXPIRED: 'ログインの有効期限が切れました再度ログインしてください',
  ACCESS_TOKEN_REVOKED: 'ログイン状態が無効になりました再度ログインしてください',
  ACCOUNT_UNAVAILABLE: 'このアカウントは利用できません',
  INTERNAL_ERROR: '内部エラーが発生しました',
  DATABASE_ERROR: 'データベース処理に失敗しました',
  WORKSPACE_NOT_FOUND: 'ワークスペースが見つかりません',
  WORKSPACE_ALREADY_DELETED: 'ワークスペースは既に削除されています',
  WORKSPACE_ACCESS_DENIED: 'このワークスペースへアクセスできません',
  WORKSPACE_PERMISSION_DENIED: 'この操作を実行する権限がありません',
  HOST_PERMISSION_REQUIRED: 'ホスト権限が必要です',
  WORKSPACE_HOST_SUSPENDED: 'ホストが停止されているため利用できません',
  WORKSPACE_HOST_DELETED: 'ホストが削除されているため利用できません',
  HOST_SUSPENDED: 'ホストが停止されているため利用できません',
  HOST_DELETED: 'ホストが削除されているため利用できません',
  USER_NOT_FOUND: 'ユーザーが見つかりません',
  TARGET_USER_NOT_FOUND: 'ユーザーが見つかりません',
  TARGET_ACCOUNT_UNAVAILABLE: 'このユーザーを追加できません',
  CANNOT_ADD_SELF: '自分自身を追加できません',
  MEMBER_ALREADY_EXISTS: '既に参加しています',
  MEMBER_NOT_FOUND: 'メンバーが見つかりません',
  CANNOT_REMOVE_HOST: 'ホストを削除できません',
  CANNOT_SUSPEND_SELF: '自分自身を停止できません',
  CANNOT_SUSPEND_HOST: 'ホストを停止できません',
  ACCOUNT_ALREADY_SUSPENDED: 'このアカウントは既に停止されています',
  ACCOUNT_NOT_SUSPENDED: 'このアカウントは停止されていません',
  CANNOT_REACTIVATE_SELF: '自分自身の停止は解除できません',
  CANNOT_DELETE_SELF: '自分自身を削除できません',
  ACCOUNT_DELETED: 'このアカウントは削除されています',
  ACCOUNT_SUSPENDED: 'このアカウントは停止されています',
  DOCUMENT_NOT_FOUND: 'ドキュメントが見つかりません',
  DOCUMENT_DELETED: 'ドキュメントは削除されています',
  DOCUMENT_CONFLICT: 'ドキュメントが更新されています',
  DOCUMENT_CONTENT_TOO_LARGE: 'ドキュメント本文が上限を超えています',
  VERSION_NOT_FOUND: '編集履歴が見つかりません',
  WEBSOCKET_TLS_REQUIRED: 'WSS接続が必要です',
  WEBSOCKET_CONNECTION_LIMIT_EXCEEDED: 'WebSocket接続数の上限に達しています',
  UNSUPPORTED_EVENT: '未対応のイベントです',
  RATE_LIMIT_SERVICE_UNAVAILABLE: 'アクセス制限サービスを利用できません',
  RATE_LIMIT_EXCEEDED: 'リクエスト回数が上限を超えました',
};

export function getErrorMessage(error: unknown): string {
  if (error instanceof TypeError) {
    return 'サーバーに接続できません';
  }

  if (error && typeof error === 'object') {
    const apiError = error as ApiError & { status?: number };
    if (apiError.code && ERROR_MESSAGES[apiError.code]) {
      return ERROR_MESSAGES[apiError.code];
    }
    if (apiError.message) {
      if (ERROR_MESSAGES[apiError.message]) {
        return ERROR_MESSAGES[apiError.message];
      }
      if (apiError.message === 'Failed to fetch') {
        return 'サーバーに接続できません';
      }
      return apiError.message;
    }
  }

  return ERROR_MESSAGES.UNKNOWN_ERROR;
}

export function formatRole(role: string): string {
  switch (role) {
    case 'host':
      return 'オーナー';
    case 'member':
      return 'メンバー';
    default:
      return role;
  }
}

export function formatUserStatus(status: string): string {
  switch (status) {
    case 'active':
      return '有効';
    case 'suspended':
      return '停止中';
    case 'deleted':
      return '削除済み';
    default:
      return status;
  }
}

export function formatUnavailableReason(reason: string): string {
  if (!reason) {
    return '利用不可';
  }
  return ERROR_MESSAGES[reason] ?? reason;
}

export function formatDate(value: string): string {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  return date.toLocaleString('ja-JP');
}

export function formatVersionType(type: string): string {
  switch (type) {
    case 'auto_save':
      return '自動保存';
    case 'manual_save':
      return '保存';
    case 'before_restore':
      return '復元前';
    case 'restore':
      return '復元';
    default:
      return type;
  }
}

export function resolveMemberName(userId: string, namesByUserId: ReadonlyMap<string, string>): string {
  return namesByUserId.get(userId) ?? '不明';
}
