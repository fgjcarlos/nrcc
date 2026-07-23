import { describe, expect, it, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { BulkImportModal } from './BulkImportModal';
import { envService } from '../services/envService';

vi.mock('../services/envService', () => ({
  envService: {
    bulkImport: vi.fn(),
  },
}));

const bulkImport = vi.mocked(envService.bulkImport);

describe('BulkImportModal', () => {
  beforeEach(() => {
    bulkImport.mockReset();
  });

  it('blocks import until validation succeeds', async () => {
    bulkImport.mockResolvedValueOnce({
      lines: [],
      issues: [{ line: 1, reason: "missing '=' between key and value" }],
      valid: false,
      summary: '1 invalid line(s)',
    });
    render(<BulkImportModal open onClose={vi.fn()} onImported={vi.fn()} />);

    await userEvent.type(screen.getByPlaceholderText(/# KEY=VALUE/), 'BADLINE');
    await userEvent.click(screen.getByRole('button', { name: 'Validate' }));

    await waitFor(() => {
      expect(screen.getByText("missing '=' between key and value")).toBeInTheDocument();
    });
    expect(screen.getByRole('button', { name: /Import/ })).toBeDisabled();
  });

  it('calls bulkImport with commit=true on a valid payload', async () => {
    bulkImport
      .mockResolvedValueOnce({
        lines: [{ line: 1, key: 'API_URL', value: 'https://x.test', type: 'string' }],
        issues: [],
        valid: true,
        summary: '1 variable(s) ready',
      })
      .mockResolvedValueOnce({
        lines: [{ line: 1, key: 'API_URL', value: 'https://x.test', type: 'string' }],
        issues: [],
        valid: true,
        summary: '1 variable(s) ready',
      });
    const onClose = vi.fn();
    const onImported = vi.fn();
    render(<BulkImportModal open onClose={onClose} onImported={onImported} />);

    const textarea = screen.getByPlaceholderText(/# KEY=VALUE/);
    await userEvent.type(textarea, 'API_URL=https://x.test');
    await userEvent.click(screen.getByRole('button', { name: 'Validate' }));
    await waitFor(() => {
      expect(screen.getByRole('button', { name: /Import/ })).not.toBeDisabled();
    });
    await userEvent.click(screen.getByRole('button', { name: /Import/ }));

    await waitFor(() => {
      expect(bulkImport).toHaveBeenNthCalledWith(2, 'API_URL=https://x.test', true);
    });
    await waitFor(() => {
      expect(onImported).toHaveBeenCalled();
      expect(onClose).toHaveBeenCalled();
    });
  });
});