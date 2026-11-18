import { describe, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { DotenvEditor } from './DotenvEditor';

const { getDotenv, saveDotenv } = vi.hoisted(() => ({
  getDotenv: vi.fn(),
  saveDotenv: vi.fn(),
}));

vi.mock('../services', () => ({ envService: { getDotenv, saveDotenv } }));
vi.mock('sonner', () => ({ toast: { success: vi.fn(), info: vi.fn(), error: vi.fn() } }));

function renderEditor() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <DotenvEditor />
    </QueryClientProvider>,
  );
}

describe('DotenvEditor', () => {
  it('shows a parse error inline as an alert and does not save', async () => {
    getDotenv.mockResolvedValue({ content: '' });
    renderEditor();

    const editor = await screen.findByRole('textbox', { name: '.env file' });
    await userEvent.type(editor, 'NOT A LINE');
    await userEvent.click(screen.getByRole('button', { name: 'Save .env' }));

    expect(screen.getByRole('alert')).toHaveTextContent('Line 1 is not in KEY=VALUE form. Nothing was saved.');
    expect(editor).toHaveAttribute('aria-invalid', 'true');
    expect(saveDotenv).not.toHaveBeenCalled();
  });

  it('announces loading through the shared status', () => {
    getDotenv.mockReturnValue(new Promise(() => {}));
    renderEditor();

    expect(screen.getByRole('status')).toHaveTextContent('Loading...');
  });
});
