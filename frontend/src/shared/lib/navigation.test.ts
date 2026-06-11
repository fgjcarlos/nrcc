import { afterEach, describe, expect, it, vi } from 'vitest';
import { setNavigator, redirectToLogin } from './navigation';

afterEach(() => {
  setNavigator(null);
  vi.restoreAllMocks();
});

describe('navigation bridge', () => {
  it('redirects to /login via the registered navigator', () => {
    const navigate = vi.fn();
    setNavigator(navigate);

    redirectToLogin();

    expect(navigate).toHaveBeenCalledWith('/login');
  });

  it('does not fall back to a full reload when a navigator is registered', () => {
    const navigate = vi.fn();
    setNavigator(navigate);

    const hrefSetter = vi.fn();
    const original = Object.getOwnPropertyDescriptor(window, 'location');
    Object.defineProperty(window, 'location', {
      configurable: true,
      value: { get href() { return ''; }, set href(v: string) { hrefSetter(v); } },
    });

    redirectToLogin();

    if (original) Object.defineProperty(window, 'location', original);
    expect(hrefSetter).not.toHaveBeenCalled();
  });

  it('falls back to window.location when no navigator is registered', () => {
    setNavigator(null);

    const hrefSetter = vi.fn();
    const original = Object.getOwnPropertyDescriptor(window, 'location');
    Object.defineProperty(window, 'location', {
      configurable: true,
      value: { get href() { return ''; }, set href(v: string) { hrefSetter(v); } },
    });

    redirectToLogin();

    if (original) Object.defineProperty(window, 'location', original);
    expect(hrefSetter).toHaveBeenCalledWith('/login');
  });
});
