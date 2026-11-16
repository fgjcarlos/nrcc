import { render, screen, within } from '@testing-library/react';
import { userEvent } from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { useLibrariesActions } from '@/features/libraries/hooks/useLibrariesActions';
import { useLibrariesData } from '@/features/libraries/hooks/useLibrariesData';
import type { InstalledLibrary } from '@/features/libraries/types';
import { LibrariesView } from './LibrariesView';

vi.mock('@/features/libraries/hooks/useLibrariesData');
vi.mock('@/features/libraries/hooks/useLibrariesActions');

const dashboard: InstalledLibrary = {
  name: '@flowfuse/node-red-dashboard',
  version: '^1.31.0',
  description: 'Dashboard nodes for Node-RED',
  homepage: 'https://dashboard.flowfuse.com',
};

function mockData(overrides: Partial<ReturnType<typeof useLibrariesData>> = {}) {
  vi.mocked(useLibrariesData).mockReturnValue({
    libraries: [dashboard],
    isLoading: false,
    isError: false,
    error: null,
    refetch: vi.fn(),
    ...overrides,
  } as unknown as ReturnType<typeof useLibrariesData>);
}

function mockActions(overrides: Partial<ReturnType<typeof useLibrariesActions>> = {}) {
  const actions = {
    installMutation: { isPending: false, isSuccess: false },
    uninstallMutation: { isPending: false },
    searchResults: [],
    searching: false,
    handleSearch: vi.fn(),
    handleInstall: vi.fn(),
    handleUninstall: vi.fn(),
    handleClearSearch: vi.fn(),
    ...overrides,
  };
  vi.mocked(useLibrariesActions).mockReturnValue(actions as unknown as ReturnType<typeof useLibrariesActions>);
  return actions;
}

describe('LibrariesView', () => {
  beforeEach(() => {
    mockData();
    mockActions();
  });

  it('titles the page without a kicker above the heading', () => {
    render(<LibrariesView />);
    expect(screen.getByRole('heading', { level: 1, name: 'Libraries' })).toBeInTheDocument();
    expect(screen.queryByText('Extensions')).not.toBeInTheDocument();
  });

  it('shows the version range as neutral mono text, not a critical chip', () => {
    render(<LibrariesView />);
    const version = screen.getByText('^1.31.0');
    expect(screen.queryByText('v^1.31.0')).not.toBeInTheDocument();
    expect(version).toHaveClass('font-mono');
    expect(version.className).not.toMatch(/badge|error|danger|critical/);
  });

  it('breaks package names only at scope and hyphen boundaries', () => {
    render(<LibrariesView />);
    const table = screen.getByRole('table', { name: 'Installed libraries' });
    const name = within(table).getByText((_, el) => el?.tagName === 'CODE' && el.textContent === dashboard.name);
    expect(name.innerHTML).toBe('@flowfuse/<wbr>node-<wbr>red-<wbr>dashboard');
    expect(name).toHaveClass('break-normal');
  });

  it('asks for a danger confirmation naming the package before uninstalling', async () => {
    const user = userEvent.setup();
    const actions = mockActions();
    render(<LibrariesView />);

    await user.click(screen.getByRole('button', { name: 'Uninstall @flowfuse/node-red-dashboard' }));
    expect(actions.handleUninstall).not.toHaveBeenCalled();

    const dialog = screen.getByRole('dialog', { name: 'Uninstall @flowfuse/node-red-dashboard?' });
    await user.click(within(dialog).getByRole('button', { name: 'Cancel' }));
    expect(actions.handleUninstall).not.toHaveBeenCalled();

    await user.click(screen.getByRole('button', { name: 'Uninstall @flowfuse/node-red-dashboard' }));
    await user.click(within(screen.getByRole('dialog')).getByRole('button', { name: 'Confirm' }));
    expect(actions.handleUninstall).toHaveBeenCalledWith('@flowfuse/node-red-dashboard');
  });

  it('names install and package links after the package', () => {
    mockActions({
      searchResults: [{ name: 'node-red-contrib-modbus', version: '5.43.0', description: 'Modbus nodes' }],
    } as Partial<ReturnType<typeof useLibrariesActions>>);
    render(<LibrariesView />);
    expect(screen.getByRole('button', { name: 'Install node-red-contrib-modbus' })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'View node-red-contrib-modbus on npm' })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Homepage of @flowfuse/node-red-dashboard' })).toBeInTheDocument();
  });

  it('uses the shared empty and error states with catalog copy', () => {
    mockData({ libraries: [] });
    const { unmount } = render(<LibrariesView />);
    expect(screen.getByText('No libraries installed').closest('[role="status"]')).not.toBeNull();
    unmount();

    const refetch = vi.fn();
    mockData({ libraries: [], isError: true, refetch } as Partial<ReturnType<typeof useLibrariesData>>);
    render(<LibrariesView />);
    expect(screen.getByRole('alert')).toHaveTextContent('Could not load libraries.');
  });

  it('says when a search finds nothing', async () => {
    const user = userEvent.setup();
    render(<LibrariesView />);
    await user.type(screen.getByRole('searchbox', { name: 'Search npm packages' }), 'zzqx');
    expect(screen.getByText('No packages match “zzqx”')).toBeInTheDocument();
  });
});
