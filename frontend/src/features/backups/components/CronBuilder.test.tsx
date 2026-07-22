import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import {
  CronBuilder,
  cronFromDateTime,
  dateTimeFromCron,
  type PresetType,
} from './CronBuilder';
import { validateCron } from '@/features/backups/lib/cronUtils';

describe('validateCron', () => {
  it('should validate proper 5-field cron expressions', () => {
    expect(validateCron('0 * * * *')).toBe(true);
    expect(validateCron('0 2 * * *')).toBe(true);
    expect(validateCron('*/15 * * * *')).toBe(true);
    expect(validateCron('0 2 1 * *')).toBe(true);
  });

  it('should reject invalid cron expressions', () => {
    expect(validateCron('99 99 99 99 99')).toBe(false);
    expect(validateCron('invalid')).toBe(false);
    expect(validateCron('0 2')).toBe(false);
    expect(validateCron('')).toBe(false);
  });

  it('should trim whitespace', () => {
    expect(validateCron('  0 2 * * *  ')).toBe(true);
  });
});

describe('cronFromDateTime', () => {
  it('maps a date+time pair to a one-shot cron expression', () => {
    expect(cronFromDateTime('2026-07-25', '14:30')).toBe('30 14 25 7 *');
  });
  it('returns null when date or time is empty', () => {
    expect(cronFromDateTime('', '14:30')).toBeNull();
    expect(cronFromDateTime('2026-07-25', '')).toBeNull();
  });
  it('returns null on malformed input', () => {
    expect(cronFromDateTime('not-a-date', '14:30')).toBeNull();
    expect(cronFromDateTime('2026-07-25', '25:99')).toBeNull();
  });
});

describe('dateTimeFromCron', () => {
  it('returns null for non-one-shot crons (wildcards, ranges)', () => {
    expect(dateTimeFromCron('0 * * * *')).toBeNull();
    expect(dateTimeFromCron('0 2 * * 0')).toBeNull();
    expect(dateTimeFromCron('*/5 * * * *')).toBeNull();
  });
  it('round-trips with cronFromDateTime', () => {
    const cron = cronFromDateTime('2026-12-01', '09:15');
    const back = dateTimeFromCron(cron!);
    expect(back).not.toBeNull();
    expect(back!.time).toBe('09:15');
    expect(back!.date.endsWith('-12-01')).toBe(true);
  });
});

function renderBuilder(schedule: PresetType, customSchedule = '', props: Partial<React.ComponentProps<typeof CronBuilder>> = {}) {
  const onChange = props.onChange ?? vi.fn();
  const onPresetChange = props.onPresetChange ?? vi.fn();
  return {
    onChange,
    onPresetChange,
    ...render(
      <CronBuilder
        schedule={schedule}
        customSchedule={customSchedule}
        onChange={onChange}
        onPresetChange={onPresetChange}
        {...props}
      />,
    ),
  };
}

describe('CronBuilder', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders the preset select with all options', () => {
    renderBuilder('disabled');
    const sel = screen.getByTestId('preset-select') as HTMLSelectElement;
    expect(sel).toBeInTheDocument();
    expect(Array.from(sel.options).map((o) => o.value)).toEqual([
      'disabled', 'hourly', 'every6h', 'daily', 'weekly', 'custom',
    ]);
  });

  it('marks the active preset in the select based on schedule, not customSchedule', () => {
    // schedule='daily' with empty customSchedule: today the operator
    // cannot tell which preset is active. With schedule-driven init,
    // the select shows 'daily'.
    renderBuilder('daily', '');
    const sel = screen.getByTestId('preset-select') as HTMLSelectElement;
    expect(sel.value).toBe('daily');
  });

  it('shows the cron expression of the active preset', () => {
    renderBuilder('daily');
    expect(screen.getByText('0 2 * * *')).toBeInTheDocument();
  });

  it('emits preset cron when a preset is selected', () => {
    const { onChange, onPresetChange } = renderBuilder('disabled');
    const sel = screen.getByTestId('preset-select');
    fireEvent.change(sel, { target: { value: 'hourly' } });
    expect(onPresetChange).toHaveBeenCalledWith('hourly');
    expect(onChange).toHaveBeenCalledWith('0 * * * *');
  });

  it('emits daily cron when daily is selected', () => {
    const { onChange, onPresetChange } = renderBuilder('disabled');
    fireEvent.change(screen.getByTestId('preset-select'), { target: { value: 'daily' } });
    expect(onPresetChange).toHaveBeenCalledWith('daily');
    expect(onChange).toHaveBeenCalledWith('0 2 * * *');
  });

  it('emits every6h cron when every6h is selected', () => {
    const { onChange } = renderBuilder('disabled');
    fireEvent.change(screen.getByTestId('preset-select'), { target: { value: 'every6h' } });
    expect(onChange).toHaveBeenCalledWith('0 */6 * * *');
  });

  it('emits weekly cron when weekly is selected', () => {
    const { onChange } = renderBuilder('disabled');
    fireEvent.change(screen.getByTestId('preset-select'), { target: { value: 'weekly' } });
    expect(onChange).toHaveBeenCalledWith('0 2 * * 0');
  });

  it('does not call onChange when disabled is selected', () => {
    const { onChange } = renderBuilder('hourly');
    fireEvent.change(screen.getByTestId('preset-select'), { target: { value: 'disabled' } });
    // disabled is a state transition, not a cron emission
    expect(onChange).not.toHaveBeenCalled();
  });

  it('shows date and time pickers when custom is active', () => {
    renderBuilder('custom', '30 14 25 7 *');
    expect(screen.getByTestId('custom-date')).toBeInTheDocument();
    expect(screen.getByTestId('custom-time')).toBeInTheDocument();
  });

  it('initializes the custom pickers from a one-shot cron', () => {
    renderBuilder('custom', '30 14 25 7 *');
    const date = screen.getByTestId('custom-date') as HTMLInputElement;
    const time = screen.getByTestId('custom-time') as HTMLInputElement;
    expect(date.value.endsWith('-07-25')).toBe(true);
    expect(time.value).toBe('14:30');
  });

  it('emits a one-shot cron when the picker changes', () => {
    const { onChange } = renderBuilder('custom', '');
    const date = screen.getByTestId('custom-date');
    const time = screen.getByTestId('custom-time');
    fireEvent.change(date, { target: { value: '2026-08-15' } });
    fireEvent.change(time, { target: { value: '09:00' } });
    expect(onChange).toHaveBeenCalledWith('0 9 15 8 *');
  });

  it('hides the raw cron input behind the advanced toggle by default', () => {
    renderBuilder('custom', '30 14 25 7 *');
    expect(screen.queryByTestId('custom-cron-input')).toBeNull();
    expect(screen.getByTestId('custom-advanced-toggle')).toBeInTheDocument();
  });

  it('shows the raw cron input after toggling advanced', () => {
    renderBuilder('custom', '30 14 25 7 *');
    fireEvent.click(screen.getByTestId('custom-advanced-toggle'));
    const raw = screen.getByTestId('custom-cron-input') as HTMLInputElement;
    expect(raw).toBeInTheDocument();
    expect(raw.value).toBe('30 14 25 7 *');
  });

  it('validates the raw cron on blur and surfaces an error', () => {
    renderBuilder('custom', '0 2 * * *');
    fireEvent.click(screen.getByTestId('custom-advanced-toggle'));
    const raw = screen.getByTestId('custom-cron-input');
    fireEvent.change(raw, { target: { value: 'invalid' } });
    fireEvent.blur(raw);
    expect(screen.getByText(/invalid cron/i)).toBeInTheDocument();
  });

  it('accepts onSave and shows a save button when provided', () => {
    renderBuilder('daily', '', { onSave: vi.fn() });
    expect(screen.getByRole('button', { name: /save/i })).toBeInTheDocument();
  });

  it('hides the save button when onSave is not provided', () => {
    renderBuilder('daily');
    expect(screen.queryByRole('button', { name: /save|confirm/i })).toBeNull();
  });

  it('calls onSave when the save button is clicked', () => {
    const onSave = vi.fn();
    renderBuilder('daily', '', { onSave });
    fireEvent.click(screen.getByRole('button', { name: /save/i }));
    expect(onSave).toHaveBeenCalled();
  });

  it('shows the next-run summary for custom', () => {
    renderBuilder('custom', '0 9 15 8 *');
    // The summary line: "Runs once on ... at ..."
    expect(screen.getByText(/runs once on/i)).toBeInTheDocument();
  });
});
