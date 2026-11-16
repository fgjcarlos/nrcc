import { beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { I18nProvider } from '@/i18n';
import { StyleguideView } from './StyleguideView';

const SECTIONS = [
  'Colour',
  'Type',
  'Spacing, radii and elevation',
  'Motion',
  'Voice',
  'Logo',
  'Buttons',
  'Fields',
  'Dialogs',
  'Tabs',
  'Tables',
  'Status and feedback',
  'Change circuit',
];

function renderGuide() {
  return render(
    <I18nProvider>
      <StyleguideView />
    </I18nProvider>,
  );
}

describe('StyleguideView (S2, S5)', () => {
  beforeEach(() => {
    window.matchMedia = vi.fn().mockImplementation((query: string) => ({
      matches: false,
      media: query,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
    }));
  });

  it('renders the marker heading, the theme and locale switches, and an index link per section', () => {
    renderGuide();
    expect(screen.getByRole('heading', { level: 1, name: 'NRCC styleguide' })).toBeInTheDocument();
    expect(screen.getByTestId('locale-switcher')).toBeInTheDocument();
    const index = screen.getByRole('navigation', { name: 'Styleguide sections' });
    for (const name of SECTIONS) {
      expect(within(index).getByRole('link', { name })).toBeInTheDocument();
      expect(screen.getByRole('region', { name })).toBeInTheDocument();
    }
  });

  it('lists every colour token for both themes with a contrast verdict', () => {
    renderGuide();
    const colour = screen.getByRole('region', { name: 'Colour' });
    expect(within(colour).getByText('--ds-text-primary')).toBeInTheDocument();
    expect(within(colour).getByText('--color-primary')).toBeInTheDocument();
    // jsdom loads no stylesheet, so values are unmeasured and never claim a pass.
    expect(within(colour).queryByText(/AA pass/)).not.toBeInTheDocument();
  });

  it('renders at least one real specimen per component group', async () => {
    renderGuide();
    const buttons = screen.getByRole('region', { name: 'Buttons' });
    for (const variant of ['primary', 'solid', 'secondary', 'ghost', 'danger']) {
      expect(within(buttons).getByText(variant, { selector: 'code' })).toBeInTheDocument();
    }
    // Real catalogue copy, one button per size plus disabled and focus.
    expect(within(buttons).getAllByRole('button', { name: 'Review 3 changes' })).toHaveLength(5);

    const fields = screen.getByRole('region', { name: 'Fields' });
    expect(within(fields).getByText('Key is required')).toBeInTheDocument();
    expect(within(fields).getAllByRole('button', { pressed: true }).length).toBeGreaterThan(0);

    expect(within(screen.getByRole('region', { name: 'Tabs' })).getByRole('tablist')).toBeInTheDocument();
    expect(within(screen.getByRole('region', { name: 'Tables' })).getAllByRole('table')).toHaveLength(2);

    const status = screen.getByRole('region', { name: 'Status and feedback' });
    expect(within(status).getByText('No environment variables configured')).toBeInTheDocument();
    expect(within(status).getAllByRole('alert').length).toBeGreaterThan(0);

    expect(screen.getAllByTestId('change-circuit')).toHaveLength(6);
    expect(screen.getAllByTestId('change-circuit-rollback')).toHaveLength(2);

    const dialogs = screen.getByRole('region', { name: 'Dialogs' });
    await userEvent.click(within(dialogs).getByRole('button', { name: 'Open dialog' }));
    expect(await screen.findByRole('dialog', { name: 'New Variable' })).toBeInTheDocument();
  });
});
