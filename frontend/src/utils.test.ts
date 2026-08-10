import { describe, expect, it } from 'vitest';
import { getErrorMessage, formatRole, formatUserStatus, formatUnavailableReason, formatDate, formatVersionType } from './utils';

describe('getErrorMessage', () => {
  it('maps known api error codes', () => {
    expect(getErrorMessage({ code: 'INVALID_CREDENTIALS', message: 'x' })).toBe(
      'メールアドレスまたはパスワードが正しくありません',
    );
  });

  it('handles TypeError as network failure', () => {
    expect(getErrorMessage(new TypeError('Failed to fetch'))).toBe('サーバーに接続できません');
  });

  it('maps message key and Failed to fetch', () => {
    expect(getErrorMessage({ code: '', message: 'INVALID_REQUEST' })).toBe('リクエスト形式が不正です');
    expect(getErrorMessage({ code: '', message: 'Failed to fetch' })).toBe('サーバーに接続できません');
  });

  it('returns raw message and unknown fallback', () => {
    expect(getErrorMessage({ code: '', message: 'custom' })).toBe('custom');
    expect(getErrorMessage({})).toBe('リクエストに失敗しました');
    expect(getErrorMessage(null)).toBe('リクエストに失敗しました');
  });
});

describe('formatters', () => {
  it('formatRole', () => {
    expect(formatRole('host')).toBe('オーナー');
    expect(formatRole('member')).toBe('メンバー');
    expect(formatRole('guest')).toBe('guest');
  });

  it('formatUserStatus', () => {
    expect(formatUserStatus('active')).toBe('有効');
    expect(formatUserStatus('suspended')).toBe('停止中');
    expect(formatUserStatus('deleted')).toBe('削除済み');
    expect(formatUserStatus('pending')).toBe('pending');
  });

  it('formatUnavailableReason', () => {
    expect(formatUnavailableReason('')).toBe('利用不可');
    expect(formatUnavailableReason('HOST_SUSPENDED')).toBe('ホストが停止されているため利用できません');
    expect(formatUnavailableReason('CUSTOM')).toBe('CUSTOM');
  });

  it('formatDate', () => {
    expect(formatDate('not-a-date')).toBe('not-a-date');
    expect(formatDate('2026-08-09T00:00:00.000Z')).toContain('2026');
  });

  it('formatVersionType', () => {
    expect(formatVersionType('auto_save')).toBe('自動保存');
    expect(formatVersionType('manual_save')).toBe('保存');
    expect(formatVersionType('before_restore')).toBe('復元前');
    expect(formatVersionType('restore')).toBe('復元');
    expect(formatVersionType('other')).toBe('other');
  });
});
