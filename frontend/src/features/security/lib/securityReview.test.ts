import { describe, expect, it } from 'vitest';
import type { TFunction } from 'i18next';
import { REDACTED_SECRET } from '@/shared/lib/redaction';
import { adminAuthReview, basicAuthReview, enablesAdminAuth, migrationRows } from './securityReview';

const t = ((key: string) => key) as unknown as TFunction;
const HASH = '$2b$10$storedhash';

describe('security review rows', () => {
  it('lists removed users, expiry and renames without any secret', () => {
    const rows = adminAuthReview(
      { users: [{ username: 'admin', permissions: '*', password: HASH }, { username: 'bob', permissions: 'read', password: HASH }], sessionExpiryTime: 3600 },
      [{ username: 'bobby', permissions: 'read', password: 'typed' }],
      0,
      t,
    );
    expect(rows.map((row) => [row.settingsKey, row.configuredValue, row.formValue])).toEqual([
      ['adminAuth.users[bobby]', null, { permissions: 'read', password: REDACTED_SECRET }],
      ['adminAuth.users[admin]', { permissions: '*', password: REDACTED_SECRET }, null],
      ['adminAuth.users[bob]', { permissions: 'read', password: REDACTED_SECRET }, null],
      ['adminAuth.sessionExpiryTime', 3600, null],
    ]);
    expect(rows.every((row) => row.restart === 'hard')).toBe(true);
    expect(JSON.stringify(rows)).not.toMatch(/typed|\$2b/);
  });

  it('reads an unchanged form as no rows', () => {
    expect(adminAuthReview({ users: [{ username: 'a', permissions: '*' }], sessionExpiryTime: 60 }, [{ username: 'a', permissions: '*', password: '' }], 60, t)).toEqual([]);
    expect(basicAuthReview('httpNodeAuth', { user: 'n', pass: HASH }, { user: ' n ', pass: '' }, t)).toEqual([]);
  });

  it('removing a basic-auth user removes its stored password too', () => {
    const rows = basicAuthReview('httpStaticAuth', { user: 's', pass: HASH }, { user: '', pass: '' }, t);
    expect(rows.map((row) => [row.settingsKey, row.configuredValue, row.formValue])).toEqual([
      ['httpStaticAuth.user', 's', null],
      ['httpStaticAuth.pass', REDACTED_SECRET, null],
    ]);
  });

  it('names the legacy migration and the adminAuth enable case', () => {
    expect(migrationRows(['staticAuth'], t)[0]).toMatchObject({ settingsKey: 'staticAuth', configuredValue: 'staticAuth', formValue: 'httpStaticAuth' });
    expect(enablesAdminAuth(0, 1)).toBe(true);
    expect(enablesAdminAuth(1, 2)).toBe(false);
    expect(enablesAdminAuth(0, 0)).toBe(false);
  });
});
