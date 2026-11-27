import { beforeEach, describe, expect, it, vi } from 'vitest';
import { fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { I18nProvider } from '@/i18n';
import { LogsView } from './LogsView';
import { logsService } from '../services/logsService';

vi.mock('../services/logsService', () => ({ logsService: { history: vi.fn(), stream: vi.fn() } }));
const sample = [
  { id: '1', timestamp: '2025-01-01T12:00:10.000Z', level: 'error', source: 'runtime', message: 'broken flow' },
  { id: '2', timestamp: '2025-01-01T12:01:10.000Z', level: 'info', source: 'runtime', message: 'started' },
];

describe('LogsView', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(logsService.history).mockResolvedValue({ entries: sample, generation: 'g', highWatermark: 'g:2' });
    vi.mocked(logsService.stream).mockImplementation(() => new Promise(() => {}));
  });

  it('selecting a time bucket filters the visible table', async () => {
    render(<I18nProvider><MemoryRouter><LogsView /></MemoryRouter></I18nProvider>);
    await screen.findByRole('table', { name: 'Runtime log entries' });
    fireEvent.click(screen.getByRole('button', { name: /1 events at 12:00/ }));
    const table = screen.getByRole('table', { name: 'Runtime log entries' });
    await waitFor(() => expect(within(table).getByText('broken flow')).toBeInTheDocument());
    expect(within(table).queryByText('started')).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Resume live tail' }));
    await waitFor(() => expect(logsService.stream).toHaveBeenCalled());
    expect(within(table).queryByText('started')).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Pause live tail' }));
    expect(within(table).queryByText('started')).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Clear chart selection' }));
    await waitFor(() => expect(within(screen.getByRole('table', { name: 'Runtime log entries' })).getByText('started')).toBeInTheDocument());
  });

  it('aborts stale history when a quick filter is chosen and ignores the stale response', async () => {
    let finishInitial!: (page: { entries: typeof sample; generation: string; highWatermark: string; startedAt: string }) => void;
    vi.mocked(logsService.history).mockImplementationOnce(() => new Promise((resolve) => { finishInitial = resolve; }))
      .mockResolvedValueOnce({ entries: [sample[0]], generation: 'g', highWatermark: 'g:2', startedAt: '2025-01-01T00:00:00Z' });
    render(<I18nProvider><MemoryRouter><LogsView /></MemoryRouter></I18nProvider>);
    const stalePromise = vi.mocked(logsService.history).mock.results[0]?.value;
    fireEvent.click(screen.getByRole('button', { name: 'Warnings' }));
    await waitFor(() => expect(logsService.history).toHaveBeenCalledTimes(2));
    expect(vi.mocked(logsService.history).mock.calls[1][0]).toMatchObject({ levels: ['warn'] });
    expect(vi.mocked(logsService.history).mock.calls[0][1]?.aborted).toBe(true);
    expect(vi.mocked(logsService.history).mock.calls[1][1]?.aborted).toBe(false);
    finishInitial({ entries: sample, generation: 'g', highWatermark: 'g:2', startedAt: '2025-01-01T00:00:00Z' });
    await stalePromise;
    await waitFor(() => expect(screen.queryByText('started')).not.toBeInTheDocument());
    expect(screen.queryByText('broken flow')).not.toBeInTheDocument();
  });

  it('applies the active server filter to appended stream entries', async () => {
    let pushEntry!: (entry: typeof sample[number]) => void;
    vi.mocked(logsService.stream).mockImplementation((_cursor, _signal, onEntry) => {
      pushEntry = onEntry;
      return new Promise(() => {});
    });
    render(<I18nProvider><MemoryRouter><LogsView /></MemoryRouter></I18nProvider>);
    await screen.findByRole('table', { name: 'Runtime log entries' });
    fireEvent.click(screen.getByRole('button', { name: 'Warnings' }));
    await waitFor(() => expect(logsService.history).toHaveBeenCalledTimes(2));
    fireEvent.click(screen.getByRole('button', { name: 'Resume live tail' }));
    await waitFor(() => expect(logsService.stream).toHaveBeenCalled());
    pushEntry({ id: 'g:3', timestamp: '2025-01-01T12:02:00Z', level: 'error', source: 'r', message: 'hidden error' });
    pushEntry({ id: 'g:4', timestamp: '2025-01-01T12:03:00Z', level: 'warn', source: 'r', message: 'visible warning' });
    await waitFor(() => expect(screen.getByText('visible warning')).toBeInTheDocument());
    expect(screen.queryByText('hidden error')).not.toBeInTheDocument();
  });

  it('does not show the no-matches state when history loading fails', async () => {
    vi.mocked(logsService.history).mockRejectedValueOnce(new Error('history unavailable'));
    render(<I18nProvider><MemoryRouter><LogsView /></MemoryRouter></I18nProvider>);
    expect(await screen.findByRole('alert')).toHaveTextContent('Unable to load runtime logs');
    expect(screen.queryByText('No logs match these filters.')).not.toBeInTheDocument();
  });

  it('filters since restart by the emitting launch, not the line timestamp', async () => {
    const startedAt = '2025-01-01T12:00:00.500Z';
    const current = { ...sample[0], timestamp: '2025-01-01T12:00:00Z', runStartedAt: startedAt };
    vi.mocked(logsService.history).mockResolvedValue({ entries: [current], generation: 'g', highWatermark: 'g:2', startedAt });
    render(<I18nProvider><MemoryRouter><LogsView /></MemoryRouter></I18nProvider>);
    fireEvent.click(screen.getByRole('button', { name: /since last restart/i }));
    const table = await screen.findByRole('table', { name: 'Runtime log entries' });
    expect(within(table).getByText('broken flow')).toBeInTheDocument();
    expect(vi.mocked(logsService.history).mock.lastCall?.[0]).toMatchObject({ sinceRestart: true });
  });

  it('starts live streaming from the snapshot cursor when resumed before history resolves', async () => {
    let finishInitial!: (page: { entries: typeof sample; generation: string; highWatermark: string; startedAt: string }) => void;
    vi.mocked(logsService.history).mockImplementationOnce(() => new Promise((resolve) => { finishInitial = resolve; }))
      .mockResolvedValueOnce({ entries: sample, generation: 'g', highWatermark: 'g:42', startedAt: '2025-01-01T00:00:00Z' });
    render(<I18nProvider><MemoryRouter><LogsView /></MemoryRouter></I18nProvider>);
    fireEvent.click(screen.getByRole('button', { name: 'Resume live tail' }));
    finishInitial({ entries: sample, generation: 'g', highWatermark: 'g:42', startedAt: '2025-01-01T00:00:00Z' });
    await waitFor(() => expect(logsService.stream).toHaveBeenCalled());
    expect(vi.mocked(logsService.stream).mock.calls[0][0]).toBe('g:42');
  });

  it('pauses and resumes the live stream with keyboard-operable button', async () => {
    render(<I18nProvider><MemoryRouter><LogsView /></MemoryRouter></I18nProvider>);
    await screen.findByRole('table', { name: 'Runtime log entries' });
    const toggle = screen.getByRole('button', { name: 'Resume live tail' });
    toggle.focus();
    fireEvent.click(toggle);
    await waitFor(() => expect(logsService.stream).toHaveBeenCalled());
    expect(screen.getByRole('button', { name: 'Pause live tail' })).toHaveAttribute('aria-pressed', 'true');
    fireEvent.click(screen.getByRole('button', { name: 'Pause live tail' }));
    expect(screen.getByRole('button', { name: 'Resume live tail' })).toHaveAttribute('aria-pressed', 'false');
  });
});
