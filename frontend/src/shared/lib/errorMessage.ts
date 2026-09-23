import axios from 'axios';
import { i18n } from '@/i18n';

/**
 * Extracts the user-facing message from an unknown thrown value.
 *
 * The backend writes errors via `model.RespondError` with envelope
 * `{ success, error: { code, message }, timestamp }`. The default
 * `error.toString()` for an AxiosError is `AxiosError: Request failed
 * with status code 400`, which loses the actionable server text. Use
 * this helper anywhere a mutation's `onError` surfaces a toast.
 *
 * Closes #708.
 */
export function errorMessage(err: unknown): string {
  if (axios.isAxiosError(err)) {
    const data = err.response?.data as { error?: { code?: string; message?: string } } | undefined;
    if (data?.error?.message) {
      const code = data.error.code ? ` (code: ${data.error.code})` : '';
      return translateMessage(`${data.error.message}${code}`);
    }
    return translateMessage(err.message);
  }
  if (err instanceof Error) return translateMessage(err.message);
  return translateMessage(String(err));
}

function translateMessage(message: string): string {
  return i18n.exists(message) ? i18n.t(message) : message;
}
