// cronBuilder.helpers.ts — pure data + functions for CronBuilder.
// Lives in a sibling file so the CronBuilder.tsx module only exports
// React components. Otherwise React Fast Refresh treats the file as
// non-component and refuses to hot-swap it during development.

// PRESET_CRON — the canonical cron expression for each preset key.
// Kept module-level so both CronBuilder and its tests can read it
// without re-creating the map on every render.
export const PRESET_CRON: Record<string, string> = {
  hourly: '0 * * * *',
  every6h: '0 */6 * * *',
  daily: '0 2 * * *',
  weekly: '0 2 * * 0',
};

// Map a (date, time) pair to the canonical one-shot cron
// `min hr dom mon dow` where dow is `*`. Returns null when the
// inputs are empty or invalid.
export function cronFromDateTime(date: string, time: string): string | null {
  if (!date || !time) return null;
  // date is YYYY-MM-DD, time is HH:MM
  const dateMatch = /^(\d{4})-(\d{2})-(\d{2})$/.exec(date);
  const timeMatch = /^(\d{2}):(\d{2})$/.exec(time);
  if (!dateMatch || !timeMatch) return null;
  // Year is captured but not used (cron has no year field).
  const mo = Number(dateMatch[2]);
  const d = Number(dateMatch[3]);
  const h = Number(timeMatch[1]);
  const mi = Number(timeMatch[2]);
  // Sanity ranges (the native pickers constrain these but
  // a pasted value could be anything).
  if (mo < 1 || mo > 12 || d < 1 || d > 31 || h < 0 || h > 23 || mi < 0 || mi > 59) return null;
  return `${mi} ${h} ${d} ${mo} *`;
}

// Inverse of cronFromDateTime. Returns null when the cron is
// not a one-shot (i.e. any field is `*`, `/`, or `-`).
export function dateTimeFromCron(cron: string): { date: string; time: string } | null {
  const trimmed = cron.trim();
  const fields = trimmed.split(/\s+/);
  if (fields.length !== 5) return null;
  const [mi, hr, dom, mon, dow] = fields;
  // one-shot: every field is a single number, no wildcards, dow = *
  const isPlain = (s: string) => /^\d+$/.test(s);
  if (!isPlain(mi) || !isPlain(hr) || !isPlain(dom) || !isPlain(mon) || dow !== '*') return null;
  const today = new Date();
  const y = today.getFullYear();
  return {
    date: `${y}-${String(mon).padStart(2, '0')}-${String(dom).padStart(2, '0')}`,
    time: `${String(hr).padStart(2, '0')}:${String(mi).padStart(2, '0')}`,
  };
}
