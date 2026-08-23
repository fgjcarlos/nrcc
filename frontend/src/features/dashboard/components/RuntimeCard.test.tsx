import { describe, expect, it, vi } from 'vitest';
import { fireEvent, render, screen } from '@testing-library/react';
import { RuntimeCard } from './RuntimeCard';

const renderCard = (props = {}) => {
  const onRequestRestart = vi.fn();
  const onOpenNodeRed = vi.fn();
  render(<RuntimeCard inDocker={true} container={{ inDocker: true, status: 'running', image: 'nodered/node-red:4.1.0' }} isRestarting={false} onRequestRestart={onRequestRestart} onOpenNodeRed={onOpenNodeRed} {...props} />);
  return { onRequestRestart, onOpenNodeRed };
};

describe('RuntimeCard', () => {
  it('shows real Docker state and image', () => {
    renderCard();
    expect(screen.getByText('Runtime')).toBeInTheDocument();
    expect(screen.getByText('running')).toBeInTheDocument();
    expect(screen.getByText('nodered/node-red:4.1.0')).toBeInTheDocument();
  });

  it('shows native and unavailable runtime values without sample data', () => {
    const { rerender } = render(<RuntimeCard inDocker={false} host={{ nodeRed: { mode: 'native' } } as never} isRestarting={false} onRequestRestart={vi.fn()} onOpenNodeRed={vi.fn()} />);
    expect(screen.getAllByText('native').length).toBeGreaterThan(0);
    rerender(<RuntimeCard inDocker={true} isRestarting={false} onRequestRestart={vi.fn()} onOpenNodeRed={vi.fn()} />);
    expect(screen.getAllByText('unavailable').length).toBeGreaterThan(0);
  });

  it('provides keyboard-operable actions and prevents duplicate restart', () => {
    const { onRequestRestart, onOpenNodeRed } = renderCard();
    const restart = screen.getByRole('button', { name: /restart/i });
    const open = screen.getByRole('button', { name: /open/i });
    expect(restart.className).toContain('focus-visible');
    fireEvent.keyDown(restart, { key: 'Enter' });
    fireEvent.keyDown(open, { key: ' ' });
    expect(onRequestRestart).toHaveBeenCalledOnce();
    expect(onOpenNodeRed).toHaveBeenCalledOnce();

    const { rerender } = render(<RuntimeCard inDocker={true} isRestarting={false} onRequestRestart={onRequestRestart} onOpenNodeRed={onOpenNodeRed} />);
    rerender(<RuntimeCard inDocker={true} isRestarting={true} onRequestRestart={onRequestRestart} onOpenNodeRed={onOpenNodeRed} />);
    expect(screen.getAllByRole('button', { name: /restart/i }).slice(-1)[0]).toBeDisabled();
  });
});
