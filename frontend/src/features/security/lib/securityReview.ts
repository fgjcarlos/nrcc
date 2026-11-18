import type { TFunction } from 'i18next';
import type { ReviewEntry } from '@/features/configuration/components/ReviewChangesPanel';
import { REDACTED_SECRET } from '@/shared/lib/redaction';

/**
 * Review rows for the Security surfaces (P1 #1): what an apply would change
 * in settings.js, shown in the shared review drawer before anything is sent.
 *
 * Secrets never reach a row. A password or stored hash is printed as the
 * API's own redaction marker (`RedactedSecret` in
 * internal/model/config_secrets.go), whether it is the stored value or a
 * newly typed one; a missing value prints as "—".
 *
 * Every row restarts Node-RED: it reads its auth settings only at start.
 */

type T = TFunction;
type Permission = '*' | 'read';

const row = (
  id: string,
  settingsKey: string,
  label: string,
  configuredValue: unknown,
  formValue: unknown,
): ReviewEntry => ({ id, settingsKey, label, restart: 'hard', configuredValue, formValue });

const CANONICAL: Record<string, string> = { nodeHttpAuth: 'httpNodeAuth', staticAuth: 'httpStaticAuth' };

/** One row per legacy alias the apply migrates to its canonical surface. */
export function migrationRows(aliases: string[], t: T): ReviewEntry[] {
  return aliases.map((alias) =>
    row(`migrate.${alias}`, alias, t('security:review.migrated'), alias, CANONICAL[alias] ?? alias),
  );
}

export function adminAuthReview(
  server: { users?: Array<{ username?: string; permissions?: Permission; password?: string }>; sessionExpiryTime?: number } | null,
  users: Array<{ username: string; permissions: Permission; password: string }>,
  expiry: number,
  t: T,
): ReviewEntry[] {
  const rows: ReviewEntry[] = [];
  const before = [...(server?.users ?? [])];
  const userKey = (name: string, index: number) => `adminAuth.users[${name || `#${index + 1}`}]`;

  users.forEach((user, index) => {
    // Match by username: a rename reads as one user removed and one added.
    const at = before.findIndex((stored) => (stored.username ?? '') === user.username);
    const key = userKey(user.username, index);
    if (at < 0) {
      rows.push(
        row(`adminAuth.users.${index}`, key, t('security:review.userAdded'), null, {
          permissions: user.permissions,
          ...(user.password ? { password: REDACTED_SECRET } : {}),
        }),
      );
      return;
    }
    const [stored] = before.splice(at, 1);
    if ((stored.permissions ?? '*') !== user.permissions) {
      rows.push(
        row(`adminAuth.users.${index}.permissions`, `${key}.permissions`, t('security:adminAuthBoundary.permission'), stored.permissions ?? '*', user.permissions),
      );
    }
    if (user.password) {
      rows.push(
        row(`adminAuth.users.${index}.password`, `${key}.password`, t('security:adminAuthBoundary.newPassword'), REDACTED_SECRET, REDACTED_SECRET),
      );
    }
  });

  before.forEach((stored, index) =>
    rows.push(
      row(`adminAuth.users.removed.${index}`, userKey(stored.username ?? '', index), t('security:review.userRemoved'), {
        permissions: stored.permissions ?? '*',
        password: REDACTED_SECRET,
      }, null),
    ),
  );

  // The payload drops adminAuth entirely with no users, and omits a 0 expiry.
  const storedExpiry = server?.sessionExpiryTime ?? 0;
  if (users.length > 0 && expiry !== storedExpiry) {
    rows.push(
      row('adminAuth.sessionExpiryTime', 'adminAuth.sessionExpiryTime', t('security:adminAuthBoundary.sessionExpiry'), storedExpiry || null, expiry > 0 ? expiry : null),
    );
  }
  return rows;
}

/** True when the apply turns adminAuth on: the editor goes behind a login. */
export function enablesAdminAuth(storedUsers: number, nextUsers: number): boolean {
  return storedUsers === 0 && nextUsers > 0;
}

export function basicAuthReview(
  surface: 'httpNodeAuth' | 'httpStaticAuth',
  server: { user?: string; pass?: string } | null,
  form: { user: string; pass: string },
  t: T,
): ReviewEntry[] {
  const rows: ReviewEntry[] = [];
  const storedUser = server?.user ?? '';
  const nextUser = form.user.trim();
  if (nextUser !== storedUser) {
    rows.push(row(`${surface}.user`, `${surface}.user`, t(`security:${surface}.username`), storedUser || null, nextUser || null));
  }
  // An empty username removes the surface, stored password included.
  const storedPass = server?.pass ? REDACTED_SECRET : null;
  const passRow = (next: string | null) =>
    row(`${surface}.pass`, `${surface}.pass`, t(`security:${surface}.password`), storedPass, next);
  if (nextUser && form.pass) rows.push(passRow(REDACTED_SECRET));
  else if (!nextUser && storedPass) rows.push(passRow(null));
  return rows;
}
