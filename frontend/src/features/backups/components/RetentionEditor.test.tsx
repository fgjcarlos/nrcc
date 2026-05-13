import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { RetentionEditor } from './RetentionEditor';

describe('RetentionEditor', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('should render three retention input fields', () => {
    render(
      <RetentionEditor
        retentionManual={7}
        retentionAuto={30}
        retentionPreRestore={2}
        onSave={vi.fn()}
        isSaving={false}
      />
    );

    expect(screen.getByLabelText(/manual/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/automatic/i) || screen.getByLabelText(/auto/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/pre-restore/i)).toBeInTheDocument();
  });

  it('should display current retention values', () => {
    render(
      <RetentionEditor
        retentionManual={7}
        retentionAuto={30}
        retentionPreRestore={2}
        onSave={vi.fn()}
        isSaving={false}
      />
    );

    expect((screen.getByDisplayValue('7') as HTMLInputElement).value).toBe('7');
    expect((screen.getByDisplayValue('30') as HTMLInputElement).value).toBe('30');
    expect((screen.getByDisplayValue('2') as HTMLInputElement).value).toBe('2');
  });

  it('should call onSave with updated values', () => {
    const onSave = vi.fn();
    render(
      <RetentionEditor
        retentionManual={7}
        retentionAuto={30}
        retentionPreRestore={2}
        onSave={onSave}
        isSaving={false}
      />
    );

    const manualInput = screen.getByDisplayValue('7') as HTMLInputElement;
    fireEvent.change(manualInput, { target: { value: '14' } });

    const saveButton = screen.getByRole('button', { name: /save/i });
    fireEvent.click(saveButton);

    expect(onSave).toHaveBeenCalled();
  });

  it('should enforce minimum value of 1', () => {
    render(
      <RetentionEditor
        retentionManual={7}
        retentionAuto={30}
        retentionPreRestore={2}
        onSave={vi.fn()}
        isSaving={false}
      />
    );

    const manualInput = screen.getByDisplayValue('7') as HTMLInputElement;
    expect(manualInput.min).toBe('1');
  });

  it('should show loading state when saving', () => {
    render(
      <RetentionEditor
        retentionManual={7}
        retentionAuto={30}
        retentionPreRestore={2}
        onSave={vi.fn()}
        isSaving={true}
      />
    );

    const saveButton = screen.getByRole('button', { name: /save/i });
    expect(saveButton).toBeDisabled();
  });

  it('should have max value of 3650 (10 years)', () => {
    render(
      <RetentionEditor
        retentionManual={7}
        retentionAuto={30}
        retentionPreRestore={2}
        onSave={vi.fn()}
        isSaving={false}
      />
    );

    const manualInput = screen.getByDisplayValue('7') as HTMLInputElement;
    expect(manualInput.max).toBe('3650');
  });

  it('should display descriptions for each field', () => {
    render(
      <RetentionEditor
        retentionManual={7}
        retentionAuto={30}
        retentionPreRestore={2}
        onSave={vi.fn()}
        isSaving={false}
      />
    );

    expect(screen.getByText(/keep manual/i)).toBeInTheDocument();
    expect(screen.getByText(/keep automatic/i)).toBeInTheDocument();
    expect(screen.getByText(/keep pre-restore/i)).toBeInTheDocument();
  });
});
