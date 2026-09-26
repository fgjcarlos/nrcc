import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { I18nProvider } from '@/i18n';
import { NodeRedVersionChip } from './NodeRedVersionChip';

function renderChip(version: string | null | undefined) {
  return render(
    <I18nProvider>
      <NodeRedVersionChip version={version} />
    </I18nProvider>,
  );
}

describe('NodeRedVersionChip', () => {
  it('renders the version with the success palette for Node-RED 5.x', () => {
    renderChip('5.0.7');
    const chip = screen.getByTestId('node-red-version-chip');
    expect(chip.className).toMatch(/bg-ds-success/);
    expect(chip.className).toMatch(/text-ds-success/);
    expect(chip).toHaveTextContent(/5\.0\.7/);
  });

  it('renders the version with the warning palette for Node-RED 4.x', () => {
    renderChip('4.0.2');
    const chip = screen.getByTestId('node-red-version-chip');
    expect(chip.className).toMatch(/bg-ds-warning/);
    expect(chip).toHaveTextContent(/4\.0\.2/);
  });

  it('renders the danger palette for unknown / unparseable versions', () => {
    renderChip('unknown');
    const chip = screen.getByTestId('node-red-version-chip');
    expect(chip.className).toMatch(/bg-ds-danger/);
  });

  it('renders the danger palette for Node-RED below 4.x', () => {
    renderChip('3.0.0');
    const chip = screen.getByTestId('node-red-version-chip');
    expect(chip.className).toMatch(/bg-ds-danger/);
  });

  it('renders the neutral palette while version is undefined', () => {
    renderChip(undefined);
    const chip = screen.getByTestId('node-red-version-chip');
    expect(chip.className).toMatch(/bg-base-300/);
    expect(chip).toHaveTextContent(/unknown/i);
  });

  it('renders the neutral palette while version is null', () => {
    renderChip(null);
    const chip = screen.getByTestId('node-red-version-chip');
    expect(chip.className).toMatch(/bg-base-300/);
  });

  it('uses the sm size by default', () => {
    renderChip('5.0.7');
    const chip = screen.getByTestId('node-red-version-chip');
    expect(chip.className).toMatch(/px-2\.5/);
  });

  it('exposes a screen-reader-friendly aria-label that includes the version', () => {
    renderChip('5.0.7');
    const chip = screen.getByTestId('node-red-version-chip');
    expect(chip.getAttribute('aria-label')).toMatch(/5\.0\.7/);
  });

  it('falls back to "unknown" in the aria-label while loading', () => {
    renderChip(undefined);
    const chip = screen.getByTestId('node-red-version-chip');
    expect(chip.getAttribute('aria-label')).toMatch(/unknown/i);
  });
});
