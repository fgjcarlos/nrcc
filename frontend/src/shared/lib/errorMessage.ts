import axios from 'axios';

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
      return `${data.error.message}${code}`;
    }
    return err.message;
  }
  if (err instanceof Error) return err.message;
  return String(err);
}
