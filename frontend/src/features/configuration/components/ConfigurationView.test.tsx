import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter } from 'react-router-dom';

// vi.mock factories are hoisted above all imports, so the mocked
// service handles have to be created via vi.hoisted before the mocks
// reference them.
const mocks = vi.hoisted(() => ({
  getConfig: vi.fn(),
  getRaw: vi.fn(),
  getStatus: vi.fn(),
}));

vi.mock('@/features/configuration/services', () => ({
  configService: {
    getConfig: mocks.getConfig,
    updateConfig: vi.fn(),
    validateConfig: vi.fn(),
    getDefaultConfig: vi.fn(),
  },
  fileService: { uploadImage: vi.fn(), deleteImage: vi.fn(), listImages: vi.fn() },
  settingsService: { getRaw: mocks.getRaw, saveRaw: vi.fn() },
}));

vi.mock('@/features/bootstrap/services', () => ({
  bootstrapService: { getStatus: mocks.getStatus },
}));

vi.mock('sonner', () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

import { ConfigurationView } from './ConfigurationView';

function renderConfiguration() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <MemoryRouter>
      <QueryClientProvider client={queryClient}>
        <ConfigurationView />
      </QueryClientProvider>
    </MemoryRouter>,
  );
}

const baseConfig = {
  uiPort: 1880,
  uiHost: '0.0.0.0',
  httpAdminRoot: '/',
  httpNodeRoot: '/',
  disableEditor: false,
  // No adminAuth / nodeHttpAuth / staticAuth — start false
  projectsEnabled: false,
  logging: { console: { level: 'info', metrics: false }, internal: { level: 'info', metrics: false } },
  flowFile: 'flows.json',
  editorTheme: {
    page: { title: 'Node-RED' },
    header: { title: 'Node-RED' },
    palette: { catalogues: [] },
    code: { lib: 'ace', theme: 'vs' },
  },
};

beforeEach(() => {
  vi.clearAllMocks();
  mocks.getConfig.mockResolvedValue({ data: { data: baseConfig, success: true, timestamp: '' } });
  mocks.getRaw.mockResolvedValue({ data: { data: { content: 'module.exports = {};\n', path: '/etc/node-red/settings.js', writable: true }, success: true, timestamp: '' } });
  mocks.getStatus.mockResolvedValue({ data: { data: { nodeRed: { mode: 'docker', detected: true }, settings: { path: '/etc/node-red/settings.js' } }, success: true, timestamp: '' } });
});

async function switchToTab(user: ReturnType<typeof userEvent.setup>, label: string) {
  // The tab buttons render an icon + the label. The accessible name
  // for the <button> is the visible label (the icon is aria-hidden via
  // lucide-react), so getByRole with `name` is the stable selector.
  const tab = await screen.findByRole('button', { name: new RegExp(`^${label}$`) });
  await user.click(tab);
}

// ToggleField renders the label as a sibling <label> inside a nested
// <div>, with the actual switch as a <button> in a sibling <div>.
// The accessible name of the button is empty, so we have to find the
// label and climb to the row container to reach the toggle.
function findToggleButton(labelText: RegExp): HTMLButtonElement {
  const label = screen.getByText(labelText);
  // Walk up until we find an element that contains a <button> sibling
  // — the row is the closest <div> whose direct or descendant children
  // include both the label and the toggle button.
  let row: HTMLElement | null = label.parentElement;
  while (row && !row.querySelector('button')) {
    row = row.parentElement;
  }
  if (!row) throw new Error(`No toggle row found for label matching ${labelText}`);
  const button = row.querySelector('button');
  if (!button) throw new Error(`No toggle button next to label matching ${labelText}`);
  return button as HTMLButtonElement;
}

describe('ConfigurationView (issue #366 — toggles were reset on every render)', () => {
  it('keeps the Enable Admin Auth toggle flipped on after a re-render', async () => {
    const user = userEvent.setup();
    renderConfiguration();

    // The "Basic" tab is the default; switch to "Authentication" first.
    await switchToTab(user, 'Authentication');

    const toggle = findToggleButton(/^Enable Admin Auth$/);
    // Initial state: fixture has no adminAuth, so the toggle starts off.
    expect(toggle.className).toContain('bg-muted');

    await user.click(toggle);
    await waitFor(() => expect(toggle.className).toContain('bg-primary'));

    // Trigger a re-render by clicking the top-level Save button. This
    // re-runs the useEffect in ConfigurationView that used to clobber
    // form state on every render before the fix landed.
    const saveButton = screen.getByText('Save', { selector: 'button' });
    await user.click(saveButton).catch(() => undefined);

    // After the re-render, the toggle should still be on. Before the
    // fix, this would have been reset to its loaded value.
    expect(toggle.className).toContain('bg-primary');
  });

  it('keeps the Editor Theme palette toggle flipped off after a re-render', async () => {
    const user = userEvent.setup();
    renderConfiguration();

    await switchToTab(user, 'Editor Theme');

    const toggle = findToggleButton(/^Allow Node Installation$/);
    // Fixture has no editorTheme.palette.editable, the transformer
    // defaults it to true, so the toggle starts in the "on" state.
    expect(toggle.className).toContain('bg-primary');

    await user.click(toggle);
    await waitFor(() => expect(toggle.className).toContain('bg-muted'));

    const saveButton = screen.getByText('Save', { selector: 'button' });
    await user.click(saveButton).catch(() => undefined);

    expect(toggle.className).toContain('bg-muted');
  });
});
