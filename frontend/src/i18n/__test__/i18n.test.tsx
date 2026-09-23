import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { act, render, screen, fireEvent } from '@testing-library/react';
import React from 'react';
import {
  I18nProvider,
  useT,
  LocaleSwitcher,
  I18N_STORAGE_KEY,
  SUPPORTED_LOCALES,
  DEFAULT_LOCALE,
} from '../index';

// i18n foundation contract — slice 1 of issue #767

const Capture: React.FC<{ onReady?: (lang: string) => void }> = ({ onReady }) => {
  const { i18n } = useT();
  React.useEffect(() => {
    onReady?.(i18n.language);
  }, [i18n.language, onReady]);
  return null;
};

const Probe: React.FC<{ testId: string }> = ({ testId }) => {
  const { t } = useT();
  return <span data-testid={testId}>{t('cancel')}</span>;
};

describe('i18n foundation (slice 1)', () => {
  beforeEach(() => {
    localStorage.clear();
    vi.restoreAllMocks();
  });

  afterEach(() => {
    localStorage.clear();
  });

  it('SUPPORTED_LOCALES contains EN and ES only', () => {
    expect(SUPPORTED_LOCALES).toEqual(['en', 'es']);
  });

  it('DEFAULT_LOCALE is English', () => {
    expect(DEFAULT_LOCALE).toBe('en');
  });

  it('I18N_STORAGE_KEY is a stable string', () => {
    expect(I18N_STORAGE_KEY).toBe('nrcc.locale');
  });

  it('provider defaults to English when localStorage is empty', async () => {
    let captured = '';
    render(
      <I18nProvider>
        <Capture onReady={(l) => { captured = l; }} />
      </I18nProvider>,
    );
    await act(async () => {
      await new Promise((r) => setTimeout(r, 5));
    });
    expect(captured).toBe('en');
  });

  it('useT returns the English string for a known EN key', async () => {
    render(
      <I18nProvider>
        <Probe testId="x" />
      </I18nProvider>,
    );
    expect(await screen.findByTestId('x')).toHaveTextContent('Cancel');
  });

  it('provider writes the chosen locale to localStorage', async () => {
    let changeLang: ((l: string) => Promise<void>) | null = null;
    const Setup: React.FC = () => {
      const { i18n } = useT();
      changeLang = (l: string) => i18n.changeLanguage(l).then(() => undefined);
      return null;
    };
    render(
      <I18nProvider>
        <Setup />
      </I18nProvider>,
    );
    await act(async () => {
      await new Promise((r) => setTimeout(r, 5));
      await changeLang!('es');
      await new Promise((r) => setTimeout(r, 5));
    });
    expect(localStorage.getItem(I18N_STORAGE_KEY)).toBe('es');
  });

  it('provider reads a valid locale from localStorage at init', async () => {
    localStorage.setItem(I18N_STORAGE_KEY, 'es');
    let captured = '';
    render(
      <I18nProvider>
        <Capture onReady={(l) => { captured = l; }} />
      </I18nProvider>,
    );
    await act(async () => {
      await new Promise((r) => setTimeout(r, 5));
    });
    expect(captured).toBe('es');
  });

  it('ignores an invalid locale from localStorage and falls back to EN', async () => {
    localStorage.setItem(I18N_STORAGE_KEY, 'fr-CA');
    let captured = '';
    render(
      <I18nProvider>
        <Capture onReady={(l) => { captured = l; }} />
      </I18nProvider>,
    );
    await act(async () => {
      await new Promise((r) => setTimeout(r, 5));
    });
    expect(captured).toBe('en');
  });

  it('LocaleSwitcher renders both supported locales', async () => {
    render(
      <I18nProvider>
        <LocaleSwitcher />
      </I18nProvider>,
    );
    expect(await screen.findByRole('button', { name: /en/i })).toBeInTheDocument();
    expect(await screen.findByRole('button', { name: /es/i })).toBeInTheDocument();
  });

  it('LocaleSwitcher click changes the active locale', async () => {
    const Switch: React.FC = () => {
      const { t, i18n } = useT();
      return (
        <>
          <LocaleSwitcher />
          <span data-testid="x">{t('cancel')} :: {i18n.language}</span>
        </>
      );
    };
    render(
      <I18nProvider>
        <Switch />
      </I18nProvider>,
    );
    const esButton = await screen.findByRole('button', { name: /es/i });
    await act(async () => {
      fireEvent.click(esButton);
      await new Promise((r) => setTimeout(r, 5));
    });
    expect(localStorage.getItem(I18N_STORAGE_KEY)).toBe('es');
    expect(await screen.findByTestId('x')).toHaveTextContent('es');
  });
});
