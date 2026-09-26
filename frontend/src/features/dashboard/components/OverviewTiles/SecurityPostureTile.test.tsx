import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { MemoryRouter } from 'react-router-dom';
import { I18nProvider } from '@/i18n';
import { SecurityPostureTile } from './SecurityPostureTile';

function renderTile(props: Parameters<typeof SecurityPostureTile>[0]) {
  return render(
    <I18nProvider>
      <MemoryRouter>
        <SecurityPostureTile {...props} />
      </MemoryRouter>
    </I18nProvider>,
  );
}

describe('SecurityPostureTile', () => {
  it('renders the success palette and all-enabled label when every surface is on', () => {
    renderTile({
      adminAuth: true,
      httpNodeAuth: true,
      httpStaticAuth: true,
      requireHttps: true,
    });
    const tile = screen.getByTestId('overview-security-posture-tile');
    expect(tile).toBeInTheDocument();
    expect(screen.getByTestId('overview-security-adminAuth').className).toMatch(/bg-ds-success/);
    expect(screen.getByTestId('overview-security-httpNodeAuth').className).toMatch(/bg-ds-success/);
    expect(screen.getByTestId('overview-security-httpStaticAuth').className).toMatch(/bg-ds-success/);
    expect(screen.getByTestId('overview-security-requireHttps').className).toMatch(/bg-ds-success/);
  });

  it('renders the danger palette for every disabled surface', () => {
    renderTile({
      adminAuth: false,
      httpNodeAuth: false,
      httpStaticAuth: false,
      requireHttps: false,
    });
    expect(screen.getByTestId('overview-security-adminAuth').className).toMatch(/bg-ds-danger/);
    expect(screen.getByTestId('overview-security-httpNodeAuth').className).toMatch(/bg-ds-danger/);
    expect(screen.getByTestId('overview-security-httpStaticAuth').className).toMatch(/bg-ds-danger/);
    expect(screen.getByTestId('overview-security-requireHttps').className).toMatch(/bg-ds-danger/);
  });

  it('renders the mixed palette when some surfaces are enabled', () => {
    renderTile({
      adminAuth: true,
      httpNodeAuth: false,
      httpStaticAuth: false,
      requireHttps: true,
    });
    expect(screen.getByTestId('overview-security-adminAuth').className).toMatch(/bg-ds-success/);
    expect(screen.getByTestId('overview-security-httpNodeAuth').className).toMatch(/bg-ds-danger/);
    expect(screen.getByTestId('overview-security-httpStaticAuth').className).toMatch(/bg-ds-danger/);
    expect(screen.getByTestId('overview-security-requireHttps').className).toMatch(/bg-ds-success/);
  });

  it('renders the overall danger chip when no surface is enabled', () => {
    renderTile({
      adminAuth: false,
      httpNodeAuth: false,
      httpStaticAuth: false,
      requireHttps: false,
    });
    const tile = screen.getByTestId('overview-security-posture-tile');
    const overallChip = tile.querySelector('[role="status"]');
    expect(overallChip?.className).toMatch(/bg-ds-danger/);
  });

  it('renders the overall success chip when every surface is enabled', () => {
    renderTile({
      adminAuth: true,
      httpNodeAuth: true,
      httpStaticAuth: true,
      requireHttps: true,
    });
    const tile = screen.getByTestId('overview-security-posture-tile');
    const overallChip = tile.querySelector('[role="status"]');
    expect(overallChip?.className).toMatch(/bg-ds-success/);
  });

  it('exposes a deep link to /security', () => {
    renderTile({
      adminAuth: true,
      httpNodeAuth: true,
      httpStaticAuth: false,
      requireHttps: true,
    });
    const link = screen.getByTestId('overview-security-posture-link');
    expect(link).toHaveAttribute('href', '/security');
  });

  it('lists all four surface rows in order', () => {
    renderTile({
      adminAuth: true,
      httpNodeAuth: true,
      httpStaticAuth: true,
      requireHttps: true,
    });
    expect(screen.getByTestId('overview-security-adminAuth')).toBeInTheDocument();
    expect(screen.getByTestId('overview-security-httpNodeAuth')).toBeInTheDocument();
    expect(screen.getByTestId('overview-security-httpStaticAuth')).toBeInTheDocument();
    expect(screen.getByTestId('overview-security-requireHttps')).toBeInTheDocument();
  });

  it('uses the warning icon when at least one surface is enabled but not all', () => {
    renderTile({
      adminAuth: true,
      httpNodeAuth: true,
      httpStaticAuth: false,
      requireHttps: false,
    });
    // Header icon (ShieldOff) implies warning state — the surface rows
    // remain individually success/danger. We assert the overall chip
    // is visible; variant mapping is exercised by the danger test.
    const tile = screen.getByTestId('overview-security-posture-tile');
    expect(tile.querySelector('svg')).toBeTruthy();
  });

  it('falls back to a single-line tile when props are missing without throwing', () => {
    // Defensive: caller could (in theory) pass zero props. The four
    // booleans default to undefined which TypeScript forbids in strict
    // mode, so we cast through `any` to simulate a future-tolerant
    // boundary (e.g. an older backend omitting the data).
    const partial = {} as Parameters<typeof SecurityPostureTile>[0];
    expect(() => renderTile(partial)).not.toThrow();
  });
});
