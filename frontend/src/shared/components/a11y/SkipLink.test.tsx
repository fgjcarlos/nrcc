import { describe, expect, it } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { SkipLink } from './SkipLink';

describe('SkipLink', () => {
  it('renders an anchor that points to the configured target', () => {
    render(<SkipLink targetId="main-content" label="Skip to main content" />);
    const link = screen.getByRole('link', { name: /skip to main content/i });
    expect(link).toHaveAttribute('href', '#main-content');
  });

  it('is visually hidden until focused, then becomes visible', async () => {
    const user = userEvent.setup();
    render(<SkipLink targetId="main-content" label="Skip to main content" />);
    const link = screen.getByRole('link', { name: /skip to main content/i });

    // The link must not capture tab order at index 0 of the document
    // when it is not the natural focus position — but it MUST be in the
    // tab order when focused (use the standard hidden-until-focus
    // technique, not display:none). The sr-only utility preserves
    // focusability.
    expect(link).toHaveClass('sr-only');
    expect(link).toHaveClass('focus:not-sr-only');

    await user.tab();
    expect(link).toHaveFocus();
  });

  it('uses the label prop as the accessible name (i18n-ready)', () => {
    render(<SkipLink targetId="main-content" label="Saltar al contenido principal" />);
    expect(
      screen.getByRole('link', { name: /saltar al contenido principal/i })
    ).toHaveAttribute('href', '#main-content');
  });
});