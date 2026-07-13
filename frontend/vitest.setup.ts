import '@testing-library/jest-dom/vitest'
import { afterAll, afterEach, beforeAll } from 'vitest'
import { server } from './src/test/msw/server'

// Node 26 exposes `globalThis.localStorage` only when started with
// `--localstorage-file`; otherwise the global is `undefined`. Several
// production modules (and a handful of test suites) call `localStorage`
// directly rather than `window.localStorage`, so a working global is
// required for the suite to be green on a stock Node 26 install.
//
// Provide a minimal in-memory Storage polyfill on `globalThis` (and
// `window.localStorage` if missing) before any test module loads. The
// store is process-local and cleared at the start of each test via
// `localStorage.clear()` from `beforeEach` hooks — same behaviour the
// suites already rely on.
function createMemoryStorage(): Storage {
  let data: Record<string, string> = {};
  return {
    get length() {
      return Object.keys(data).length;
    },
    clear() {
      data = {};
    },
    getItem(key: string): string | null {
      return Object.prototype.hasOwnProperty.call(data, key) ? data[key] : null;
    },
    key(index: number): string | null {
      return Object.keys(data)[index] ?? null;
    },
    removeItem(key: string): void {
      delete data[key];
    },
    setItem(key: string, value: string): void {
      data[key] = String(value);
    },
  };
}

const storage = createMemoryStorage();

if (typeof globalThis.localStorage === 'undefined' || globalThis.localStorage === null) {
  Object.defineProperty(globalThis, 'localStorage', {
    value: storage,
    writable: false,
    configurable: true,
  });
}
if (typeof window !== 'undefined' && (typeof window.localStorage === 'undefined' || window.localStorage === null)) {
  Object.defineProperty(window, 'localStorage', {
    value: storage,
    writable: false,
    configurable: true,
  });
}

beforeAll(() => server.listen({ onUnhandledRequest: 'error' }))
afterEach(() => server.resetHandlers())
afterAll(() => server.close())
