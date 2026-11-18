/**
 * The API's own secret redaction marker (`RedactedSecret` in
 * internal/model/config_secrets.go). Every diff the UI shows prints a
 * secret as this marker, never as its value.
 */
export const REDACTED_SECRET = '[redacted]';

/** The marker for a non-empty secret; an empty one is returned as is. */
export const redactSecret = (value: unknown): unknown =>
  value === null || value === undefined || value === '' ? value : REDACTED_SECRET;
