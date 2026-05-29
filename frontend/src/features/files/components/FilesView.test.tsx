import { beforeEach, describe, expect, it, vi } from 'vitest';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { FilesView } from './FilesView';
import { useFiles } from '../hooks';
import type { ManagedFile } from '../types';

vi.mock('../hooks', () => ({
  useFiles: vi.fn(),
}));

vi.mock('sonner', () => ({
  toast: {
    success: vi.fn(),
    error: vi.fn(),
  },
}));

const mockFile: ManagedFile = {
  name: 'flows.json',
  size: 2048,
  modTime: 1_765_000_000,
};

const createMutation = () => ({
  mutate: vi.fn(),
  isPending: false,
  isError: false,
});

function mockUseFiles(overrides: Record<string, unknown> = {}) {
  const value = {
    files: [] as ManagedFile[],
    isLoading: false,
    isError: false,
    error: null,
    refetch: vi.fn(),
    uploadMutation: createMutation(),
    deleteMutation: createMutation(),
    ...overrides,
  } as unknown as ReturnType<typeof useFiles>;

  vi.mocked(useFiles).mockReturnValue(value);
  return value;
}

function renderView() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });

  return render(
    <QueryClientProvider client={queryClient}>
      <FilesView />
    </QueryClientProvider>,
  );
}

describe('FilesView', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders an empty state when no files exist', () => {
    mockUseFiles();

    renderView();

    expect(screen.getByRole('heading', { name: 'Files' })).toBeInTheDocument();
    expect(screen.getByText('No files uploaded yet')).toBeInTheDocument();
  });

  it('lists files returned by the API hook', () => {
    mockUseFiles({ files: [mockFile] });

    renderView();

    expect(screen.getByText('flows.json')).toBeInTheDocument();
    expect(screen.getByText('2.0 KB')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Download flows.json' })).toHaveAttribute(
      'href',
      '/api/files/flows.json/download',
    );
  });

  it('uploads the selected file without a full page refresh', async () => {
    const user = userEvent.setup();
    const uploadMutation = createMutation();
    mockUseFiles({ uploadMutation });
    renderView();

    const input = screen.getByLabelText('Choose file to upload');
    const file = new File(['hello'], 'hello.txt', { type: 'text/plain' });

    await user.upload(input, file);

    expect(uploadMutation.mutate).toHaveBeenCalledWith(file, expect.objectContaining({ onSettled: expect.any(Function) }));
  });

  it('shows an upload error message', () => {
    mockUseFiles({ uploadMutation: { ...createMutation(), isError: true } });

    renderView();

    expect(screen.getByRole('alert')).toHaveTextContent('Upload failed');
  });

  it('asks for confirmation before deleting a file', async () => {
    const user = userEvent.setup();
    const deleteMutation = createMutation();
    mockUseFiles({ files: [mockFile], deleteMutation });
    renderView();

    await user.click(screen.getByRole('button', { name: 'Delete flows.json' }));

    expect(screen.getByRole('heading', { name: 'Delete file' })).toBeInTheDocument();
    expect(deleteMutation.mutate).not.toHaveBeenCalled();

    await user.type(screen.getByPlaceholderText('flows.json'), 'flows.json');
    await user.click(screen.getByRole('button', { name: 'Confirm' }));

    await waitFor(() => expect(deleteMutation.mutate).toHaveBeenCalledWith('flows.json', expect.any(Object)));
  });

  it('renders loading and error states', () => {
    mockUseFiles({ isLoading: true });
    const { rerender } = render(
      <QueryClientProvider client={new QueryClient()}>
        <FilesView />
      </QueryClientProvider>,
    );

    expect(screen.getByRole('status')).toHaveTextContent('Loading files');

    mockUseFiles({ isError: true });
    rerender(
      <QueryClientProvider client={new QueryClient()}>
        <FilesView />
      </QueryClientProvider>,
    );
    expect(screen.getByText('Could not load files.')).toBeInTheDocument();
  });
});
