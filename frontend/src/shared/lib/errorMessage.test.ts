import { describe, it, expect } from 'vitest';
import { AxiosError, AxiosHeaders } from 'axios';
import { errorMessage } from './errorMessage';

// Use the real AxiosError class so axios.isAxiosError returns true via
// instanceof — no mock gymnastics required.
function makeAxiosError(status: number, body: unknown): AxiosError {
  const headers = new AxiosHeaders();
  // Minimal axios config-shape payload AxiosError accepts in tests.
  return new AxiosError(
    `Request failed with status code ${status}`,
    String(status),
    {
      url: '/api/test',
      method: 'post',
      headers,
      data: undefined,
      params: undefined,
      timeout: 0,
      transitional: undefined as never,
      signal: undefined as never,
      baseURL: '',
    },
    undefined,
    {
      status,
      data: body,
      statusText: String(status),
      headers,
      config: undefined as never,
    },
  );
}

describe('errorMessage', () => {
  it('returns server error.message with code when present', () => {
    const err = makeAxiosError(400, { error: { code: 'VALIDATION_ERROR', message: 'adminAuth: bad password' } });
    expect(errorMessage(err)).toBe('adminAuth: bad password (code: VALIDATION_ERROR)');
  });

  it('omits the (code: …) suffix when the server payload has no code', () => {
    const err = makeAxiosError(500, { error: { message: 'kaboom' } });
    expect(errorMessage(err)).toBe('kaboom');
  });

  it('falls back to err.message when the response body has no error.message', () => {
    const err = makeAxiosError(400, {});
    expect(errorMessage(err)).toBe('Request failed with status code 400');
  });

  it('returns Error.message for plain Errors', () => {
    expect(errorMessage(new Error('boom'))).toBe('boom');
  });

  it('returns String(err) for anything else', () => {
    expect(errorMessage('plain string')).toBe('plain string');
    expect(errorMessage(undefined)).toBe('undefined');
    expect(errorMessage(42)).toBe('42');
  });
});
