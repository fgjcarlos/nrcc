import { describe, expect, it, vi } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { userEvent } from '@testing-library/user-event';
import { ConfirmationDialog } from './ConfirmationDialog';
import { UI_COPY } from '@/shared/constants/uiCopy';

describe('ConfirmationDialog', () => {
  it('renders when isOpen is true', () => {
    render(
      <ConfirmationDialog
        isOpen={true}
        title="Delete User"
        description="Are you sure?"
        onConfirm={vi.fn()}
        onCancel={vi.fn()}
      />
    );

    expect(screen.getByText('Delete User')).toBeInTheDocument();
    expect(screen.getByText('Are you sure?')).toBeInTheDocument();
    expect(screen.getByText(UI_COPY.cancel)).toBeInTheDocument();
    expect(screen.getByText(UI_COPY.confirm)).toBeInTheDocument();
  });

  it('exposes dialog semantics for assistive technology (#296)', () => {
    render(
      <ConfirmationDialog
        isOpen={true}
        title="Delete User"
        description="Are you sure?"
        onConfirm={vi.fn()}
        onCancel={vi.fn()}
      />
    );

    const dialog = screen.getByRole('dialog');
    expect(dialog).toHaveAttribute('aria-modal', 'true');
    expect(dialog).toHaveAccessibleName('Delete User');
  });

  it('does not render when isOpen is false', () => {
    render(
      <ConfirmationDialog
        isOpen={false}
        title="Delete User"
        description="Are you sure?"
        onConfirm={vi.fn()}
        onCancel={vi.fn()}
      />
    );

    expect(screen.queryByText('Delete User')).not.toBeInTheDocument();
  });

  it('calls onCancel when cancel button is clicked', async () => {
    const onCancel = vi.fn();
    render(
      <ConfirmationDialog
        isOpen={true}
        title="Delete User"
        description="Are you sure?"
        onConfirm={vi.fn()}
        onCancel={onCancel}
      />
    );

    await userEvent.click(screen.getByText(UI_COPY.cancel));
    expect(onCancel).toHaveBeenCalled();
  });

  it('calls onConfirm when confirm button is clicked', async () => {
    const onConfirm = vi.fn();
    render(
      <ConfirmationDialog
        isOpen={true}
        title="Delete User"
        description="Are you sure?"
        onConfirm={onConfirm}
        onCancel={vi.fn()}
      />
    );

    await userEvent.click(screen.getByText(UI_COPY.confirm));
    expect(onConfirm).toHaveBeenCalled();
  });

  it('closes on Escape key when not pending', async () => {
    const onCancel = vi.fn();
    render(
      <ConfirmationDialog
        isOpen={true}
        title="Delete User"
        description="Are you sure?"
        isPending={false}
        onConfirm={vi.fn()}
        onCancel={onCancel}
      />
    );

    fireEvent.keyDown(document, { key: 'Escape' });
    expect(onCancel).toHaveBeenCalled();
  });

  it('does not close on Escape key when pending', async () => {
    const onCancel = vi.fn();
    render(
      <ConfirmationDialog
        isOpen={true}
        title="Delete User"
        description="Are you sure?"
        isPending={true}
        onConfirm={vi.fn()}
        onCancel={onCancel}
      />
    );

    fireEvent.keyDown(document, { key: 'Escape' });
    expect(onCancel).not.toHaveBeenCalled();
  });

  it('confirms on Enter key when not pending and canConfirm is true', async () => {
    const onConfirm = vi.fn();
    render(
      <ConfirmationDialog
        isOpen={true}
        title="Delete User"
        description="Are you sure?"
        isPending={false}
        onConfirm={onConfirm}
        onCancel={vi.fn()}
      />
    );

    fireEvent.keyDown(document, { key: 'Enter' });
    expect(onConfirm).toHaveBeenCalled();
  });

  it('renders English copy for confirmation text label when confirmText is provided', async () => {
    render(
      <ConfirmationDialog
        isOpen={true}
        title="Delete User"
        description="Are you sure?"
        confirmText="username"
        onConfirm={vi.fn()}
        onCancel={vi.fn()}
      />
    );

    expect(screen.getByText(UI_COPY.typeToConfirmDelete('username'))).toBeInTheDocument();
  });

  it('disables confirm button when confirmText does not match input', async () => {
    render(
      <ConfirmationDialog
        isOpen={true}
        title="Delete User"
        description="Are you sure?"
        confirmText="username"
        onConfirm={vi.fn()}
        onCancel={vi.fn()}
      />
    );

    const input = screen.getByPlaceholderText('username');
    await userEvent.type(input, 'wrong');

    const confirmBtn = screen.getByRole('button', { name: UI_COPY.confirm });
    expect(confirmBtn).toBeDisabled();
  });

  it('enables confirm button when confirmText matches input', async () => {
    const onConfirm = vi.fn();
    render(
      <ConfirmationDialog
        isOpen={true}
        title="Delete User"
        description="Are you sure?"
        confirmText="username"
        onConfirm={onConfirm}
        onCancel={vi.fn()}
      />
    );

    const input = screen.getByPlaceholderText('username') as HTMLInputElement;
    await userEvent.type(input, 'username');

    const confirmBtn = screen.getByRole('button', { name: UI_COPY.confirm });
    expect(confirmBtn).not.toBeDisabled();

    await userEvent.click(confirmBtn);
    expect(onConfirm).toHaveBeenCalled();
  });

  it('shows processing state when isPending is true', () => {
    render(
      <ConfirmationDialog
        isOpen={true}
        title="Delete User"
        description="Are you sure?"
        isPending={true}
        onConfirm={vi.fn()}
        onCancel={vi.fn()}
      />
    );

    expect(screen.getByText(UI_COPY.processing)).toBeInTheDocument();
  });

  it('disables buttons when isPending is true', () => {
    render(
      <ConfirmationDialog
        isOpen={true}
        title="Delete User"
        description="Are you sure?"
        isPending={true}
        onConfirm={vi.fn()}
        onCancel={vi.fn()}
      />
    );

    const cancelBtn = screen.getByText(UI_COPY.cancel);
    const confirmBtn = screen.getByText(UI_COPY.processing);

    expect(cancelBtn).toBeDisabled();
    expect(confirmBtn).toBeDisabled();
  });

  it('renders danger variant styling', () => {
    render(
      <ConfirmationDialog
        isOpen={true}
        title="Delete User"
        description="Are you sure?"
        variant="danger"
        onConfirm={vi.fn()}
        onCancel={vi.fn()}
      />
    );

    const confirmBtn = screen.getByText(UI_COPY.confirm);
    expect(confirmBtn).toHaveClass('bg-error');
  });

  it('auto-focuses input when confirmText is provided', async () => {
    render(
      <ConfirmationDialog
        isOpen={true}
        title="Delete User"
        description="Are you sure?"
        confirmText="username"
        onConfirm={vi.fn()}
        onCancel={vi.fn()}
      />
    );

    await waitFor(() => {
      const input = screen.getByPlaceholderText('username') as HTMLInputElement;
      expect(document.activeElement).toBe(input);
    });
  });

  it('gates Confirm behind an acknowledgement checkbox when acknowledgement is provided', async () => {
    const user = userEvent.setup();
    render(
      <ConfirmationDialog
        isOpen={true}
        title="Edit settings.js"
        description="Risky action"
        acknowledgement="I understand the risks"
        onConfirm={vi.fn()}
        onCancel={vi.fn()}
      />
    );

    const ack = screen.getByTestId('confirmation-dialog-ack');
    expect(ack).not.toBeChecked();
    const confirmBtn = screen.getByRole('button', { name: UI_COPY.confirm });
    expect(confirmBtn).toBeDisabled();

    await user.click(ack);
    expect(ack).toBeChecked();
    expect(confirmBtn).toBeEnabled();
  });

  it('combines confirmText and acknowledgement gates', async () => {
    const user = userEvent.setup();
    render(
      <ConfirmationDialog
        isOpen={true}
        title="Edit settings.js"
        description="Risky action"
        confirmText="settings.js"
        acknowledgement="I understand the risks"
        onConfirm={vi.fn()}
        onCancel={vi.fn()}
      />
    );

    const confirmBtn = screen.getByRole('button', { name: UI_COPY.confirm });
    const ack = screen.getByTestId('confirmation-dialog-ack');
    const input = screen.getByPlaceholderText('settings.js') as HTMLInputElement;

    // Both gates closed.
    expect(confirmBtn).toBeDisabled();
    // Tick the ack only — still gated by confirmText.
    await user.click(ack);
    expect(confirmBtn).toBeDisabled();
    // Type the wrong text.
    await user.type(input, 'wrong');
    expect(confirmBtn).toBeDisabled();
    // Fix the text — both gates open.
    await user.clear(input);
    await user.type(input, 'settings.js');
    expect(confirmBtn).toBeEnabled();
  });

  it('resets the acknowledgement state when the dialog is re-opened', async () => {
    const user = userEvent.setup();
    const { rerender } = render(
      <ConfirmationDialog
        isOpen={true}
        title="Edit"
        description="x"
        acknowledgement="I understand"
        onConfirm={vi.fn()}
        onCancel={vi.fn()}
      />
    );

    await user.click(screen.getByTestId('confirmation-dialog-ack'));
    expect(screen.getByTestId('confirmation-dialog-ack')).toBeChecked();

    rerender(
      <ConfirmationDialog
        isOpen={false}
        title="Edit"
        description="x"
        acknowledgement="I understand"
        onConfirm={vi.fn()}
        onCancel={vi.fn()}
      />
    );
    rerender(
      <ConfirmationDialog
        isOpen={true}
        title="Edit"
        description="x"
        acknowledgement="I understand"
        onConfirm={vi.fn()}
        onCancel={vi.fn()}
      />
    );

    // Re-opened dialog should not carry the previous tick.
    expect(screen.getByTestId('confirmation-dialog-ack')).not.toBeChecked();
    expect(screen.getByRole('button', { name: UI_COPY.confirm })).toBeDisabled();
  });
});
