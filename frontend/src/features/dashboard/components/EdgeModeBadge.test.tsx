import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { EdgeModeBadge } from './EdgeModeBadge';

describe('EdgeModeBadge', () => {
  it('shows "enabled" when edge mode is on', () => {
    render(<EdgeModeBadge enabled />);
    expect(screen.getByText('Edge mode: enabled')).toBeInTheDocument();
  });

  it('shows "disabled" when edge mode is off', () => {
    render(<EdgeModeBadge enabled={false} />);
    expect(screen.getByText('Edge mode: disabled')).toBeInTheDocument();
  });

  it('shows "disabled" when edge mode is undefined (older backend)', () => {
    render(<EdgeModeBadge />);
    expect(screen.getByText('Edge mode: disabled')).toBeInTheDocument();
  });
});
