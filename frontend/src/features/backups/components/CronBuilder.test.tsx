import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { CronBuilder, validateCron } from './CronBuilder';

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

describe('CronBuilder', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('should render all preset buttons', () => {
    render(
      <CronBuilder 
        value="disabled"
        onChange={vi.fn()}
      />
    );

    expect(screen.getByTestId('preset-hourly')).toBeInTheDocument();
    expect(screen.getByTestId('preset-every6h')).toBeInTheDocument();
    expect(screen.getByTestId('preset-daily')).toBeInTheDocument();
    expect(screen.getByTestId('preset-weekly')).toBeInTheDocument();
    expect(screen.getByTestId('preset-custom')).toBeInTheDocument();
  });

  it('should emit hourly cron when hourly preset clicked', () => {
    const onChange = vi.fn();
    render(
      <CronBuilder value="disabled" onChange={onChange} />
    );

    fireEvent.click(screen.getByTestId('preset-hourly'));
    expect(onChange).toHaveBeenCalledWith('0 * * * *');
  });

  it('should emit daily cron when daily preset clicked', () => {
    const onChange = vi.fn();
    render(
      <CronBuilder value="disabled" onChange={onChange} />
    );

    fireEvent.click(screen.getByTestId('preset-daily'));
    expect(onChange).toHaveBeenCalledWith('0 2 * * *');
  });

  it('should emit every6h cron when every6h preset clicked', () => {
    const onChange = vi.fn();
    render(
      <CronBuilder value="disabled" onChange={onChange} />
    );

    fireEvent.click(screen.getByTestId('preset-every6h'));
    expect(onChange).toHaveBeenCalledWith('0 */6 * * *');
  });

  it('should emit weekly cron when weekly preset clicked', () => {
    const onChange = vi.fn();
    render(
      <CronBuilder value="disabled" onChange={onChange} />
    );

    fireEvent.click(screen.getByTestId('preset-weekly'));
    expect(onChange).toHaveBeenCalledWith('0 2 * * 0');
  });

  it('should show custom input when custom preset clicked', () => {
    render(
      <CronBuilder value="disabled" onChange={vi.fn()} />
    );

    fireEvent.click(screen.getByTestId('preset-custom'));
    expect(screen.getByPlaceholderText(/format/i)).toBeInTheDocument();
  });

  it('should validate custom cron on blur', () => {
    render(
      <CronBuilder value="disabled" onChange={vi.fn()} />
    );

    fireEvent.click(screen.getByTestId('preset-custom'));
    const input = screen.getByPlaceholderText(/format/i) as HTMLInputElement;

    fireEvent.change(input, { target: { value: 'invalid' } });
    fireEvent.blur(input);

    expect(screen.getByText(/invalid/i)).toBeInTheDocument();
  });

  it('should emit valid custom cron on blur', () => {
    const onChange = vi.fn();
    render(
      <CronBuilder value="disabled" onChange={onChange} />
    );

    fireEvent.click(screen.getByTestId('preset-custom'));
    const input = screen.getByPlaceholderText(/format/i) as HTMLInputElement;

    fireEvent.change(input, { target: { value: '*/5 * * * *' } });
    fireEvent.blur(input);

    expect(onChange).toHaveBeenCalledWith('*/5 * * * *');
  });

  it('should mark correct preset as checked based on value', () => {
    render(
      <CronBuilder value="0 2 * * *" onChange={vi.fn()} />
    );

    const dailyRadio = screen.getByTestId('preset-daily') as HTMLInputElement;
    expect(dailyRadio.checked).toBe(true);
  });

  it('should show custom value when initialized with custom cron', () => {
    render(
      <CronBuilder value="*/15 * * * *" onChange={vi.fn()} />
    );

    const customRadio = screen.getByTestId('preset-custom') as HTMLInputElement;
    expect(customRadio.checked).toBe(true);
    expect((screen.getByPlaceholderText(/format/i) as HTMLInputElement).value).toBe('*/15 * * * *');
  });

  it('should show error for invalid initial value', () => {
    render(
      <CronBuilder value="invalid-cron" onChange={vi.fn()} />
    );

    expect(screen.getByText(/invalid/i)).toBeInTheDocument();
  });

  it('should not call onChange for invalid custom cron', () => {
    const onChange = vi.fn();
    render(
      <CronBuilder value="disabled" onChange={onChange} />
    );

    fireEvent.click(screen.getByTestId('preset-custom'));
    const input = screen.getByPlaceholderText(/format/i) as HTMLInputElement;

    fireEvent.change(input, { target: { value: '99 99 99 99 99' } });
    fireEvent.blur(input);

    // onChange should NOT be called for invalid cron
    expect(onChange).not.toHaveBeenCalled();
    expect(screen.getByText(/invalid/i)).toBeInTheDocument();
  });

  it('should trim whitespace from custom input', () => {
    const onChange = vi.fn();
    render(
      <CronBuilder value="disabled" onChange={onChange} />
    );

    fireEvent.click(screen.getByTestId('preset-custom'));
    const input = screen.getByPlaceholderText(/format/i) as HTMLInputElement;

    fireEvent.change(input, { target: { value: '  0 2 * * *  ' } });
    fireEvent.blur(input);

    expect(onChange).toHaveBeenCalledWith('0 2 * * *');
  });

  // NEW TEST: Verify that CronBuilder has onSave callback for explicit save (not auto-save)
  it('should accept onSave callback for explicit save when provided', () => {
    const onSave = vi.fn();
    const onChange = vi.fn();
    render(
      <CronBuilder 
        value="disabled" 
        onChange={onChange}
        onSave={onSave}
      />
    );

    // When onSave is provided, there should be a save button
    expect(screen.queryByRole('button', { name: /save|confirm/i })).toBeTruthy();
  });

  // TRIANGULATION: Verify save button is NOT shown when onSave not provided
  it('should not show save button when onSave not provided', () => {
    render(
      <CronBuilder 
        value="disabled" 
        onChange={vi.fn()}
      />
    );

    expect(screen.queryByRole('button', { name: /save|confirm/i })).toBeNull();
  });

  // TRIANGULATION: Verify save button calls onSave when clicked
  it('should call onSave callback when save button clicked', () => {
    const onSave = vi.fn();
    render(
      <CronBuilder 
        value="0 2 * * *" 
        onChange={vi.fn()}
        onSave={onSave}
      />
    );

    const saveButton = screen.getByRole('button', { name: /save/i });
    fireEvent.click(saveButton);

    expect(onSave).toHaveBeenCalled();
  });

  // NEW: Verify helpful text about cron format is visible when custom is selected
  it('should show cron format help text in custom mode', () => {
    render(
      <CronBuilder value="disabled" onChange={vi.fn()} />
    );

    fireEvent.click(screen.getByTestId('preset-custom'));
    
    // Should show the format explanation
    expect(screen.getByText(/5-field format/i)).toBeInTheDocument();
    expect(screen.getByText(/minute hour day-of-month month day-of-week/i)).toBeInTheDocument();
  });

  // NEW: Verify placeholder text provides example
  it('should show example cron in input placeholder', () => {
    render(
      <CronBuilder value="disabled" onChange={vi.fn()} />
    );

    fireEvent.click(screen.getByTestId('preset-custom'));
    const input = screen.getByPlaceholderText(/0 2 \* \* \*/i);
    
    expect(input).toBeInTheDocument();
  });
});
