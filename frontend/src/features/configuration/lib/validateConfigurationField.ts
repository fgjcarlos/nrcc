import { z } from 'zod';
import type { NodeRedConfigFormData } from '@/shared/types';
import type { FieldDiff, ValidationState } from '../hooks/useConfigurationDiff';
import { passwordSchema } from '@/shared/validation/schemas';

/**
 * Issue #766 slice F — per-field validation helper.
 *
 * Promotes the auth-only `validateAuthFields` from
 * `useConfigurationActions` to a per-field callable so the six-column
 * row can show validation feedback while the operator types. The
 * pre-existing `validateAuthFields` wrapper still runs the aggregate on
 * Save; this module only adds the per-field primitive.
 */

interface ValidationResult {
  state: ValidationState;
  message?: string;
}

const portRange = z.number().int().min(1).max(65535);
const positiveInt = z.number().int().positive();
const requiredString = z.string().trim().min(1, 'Required');
const minUsername = z.string().trim().min(3, 'Username must be at least 3 characters');

/**
 * Per-field validators. Return `{ state: 'valid' }` for fields with no
 * specific rule.
 */
export function validateConfigurationField(
  key: keyof NodeRedConfigFormData,
  value: unknown,
  rawSettingsContent: string
): FieldDiff['validation'] {
  const result = validateOne(key, value);
  if (result.state === 'valid') return result;
  // Only fields the operator has touched are reported as invalid.
  // Untyped / untouched fields stay at `valid` so the row does not
  // light up red before any input.
  if (!isFieldTouched(key, value, rawSettingsContent)) {
    return { state: 'valid' };
  }
  return result;
}

function validateOne(key: keyof NodeRedConfigFormData, value: unknown): ValidationResult {
  switch (key) {
    case 'uiPort': {
      const parsed = portRange.safeParse(value);
      return parsed.success
        ? { state: 'valid' }
        : { state: 'invalid', message: 'Port must be 1–65535' };
    }
    case 'uiHost': {
      const v = typeof value === 'string' ? value.trim() : '';
      return v.length === 0 ? { state: 'invalid', message: 'Host is required' } : { state: 'valid' };
    }
    case 'httpAdminRoot':
    case 'httpNodeRoot': {
      const v = typeof value === 'string' ? value.trim() : '';
      if (!v.startsWith('/')) return { state: 'invalid', message: 'Must start with /' };
      return { state: 'valid' };
    }
    case 'flowFile': {
      const v = typeof value === 'string' ? value.trim() : '';
      return v.length === 0 ? { state: 'invalid', message: 'flowFile is required' } : { state: 'valid' };
    }
    case 'editorCodeFontSize': {
      const parsed = positiveInt.safeParse(value);
      return parsed.success
        ? { state: 'valid' }
        : { state: 'invalid', message: 'Font size must be a positive integer' };
    }
    case 'authAdminPassword': {
      // Only validate when non-empty: the form treats empty as "leave
      // existing hash intact".
      if (typeof value !== 'string' || value.length === 0) return { state: 'valid' };
      const parsed = passwordSchema.safeParse(value);
      return parsed.success
        ? { state: 'valid' }
        : { state: 'invalid', message: parsed.error.issues[0]?.message ?? 'Invalid password' };
    }
    case 'authAdminUser':
    case 'authNodeHttpUser':
    case 'authStaticUser': {
      if (typeof value !== 'string' || value.length === 0) return { state: 'valid' };
      const parsed = minUsername.safeParse(value);
      return parsed.success
        ? { state: 'valid' }
        : { state: 'invalid', message: parsed.error.issues[0]?.message ?? 'Invalid username' };
    }
    case 'authNodeHttpPassword':
    case 'authStaticPassword': {
      if (typeof value !== 'string' || value.length === 0) return { state: 'valid' };
      const parsed = passwordSchema.safeParse(value);
      return parsed.success
        ? { state: 'valid' }
        : { state: 'invalid', message: parsed.error.issues[0]?.message ?? 'Invalid password' };
    }
    case 'httpsPort': {
      // Optional; 0 means "no TLS port"
      if (typeof value !== 'number' || value === 0) return { state: 'valid' };
      const parsed = portRange.safeParse(value);
      return parsed.success
        ? { state: 'valid' }
        : { state: 'invalid', message: 'Port must be 1–65535' };
    }
    case 'lang': {
      const parsed = requiredString.safeParse(value);
      return parsed.success
        ? { state: 'valid' }
        : { state: 'invalid', message: parsed.error.issues[0]?.message ?? 'Language is required' };
    }
    default:
      return { state: 'valid' };
  }
}

/**
 * Heuristic: a field is "touched" if its value differs from the
 * raw-settings content (when present) or if it is currently invalid.
 * This prevents lighting up empty defaults as invalid before the
 * operator has interacted with the form.
 */
function isFieldTouched(
  _key: keyof NodeRedConfigFormData,
  value: unknown,
  rawSettingsContent: string
): boolean {
  if (!rawSettingsContent) {
    // Without raw settings as a baseline we cannot tell; default to
    // treating any present value as touched.
    return value !== undefined && value !== null && value !== '';
  }
  // Quick heuristic: if the value's string representation does not
  // appear anywhere in the raw settings, the operator has typed
  // something new. This is intentionally permissive — false positives
  // only show "invalid" for already-bad inputs.
  if (typeof value === 'string') {
    return !rawSettingsContent.includes(value);
  }
  if (typeof value === 'number') {
    return !rawSettingsContent.includes(String(value));
  }
  return false;
}