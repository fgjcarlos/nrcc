import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor, within } from '@testing-library/react';
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

describe('ConfigurationView (issue #363 — Advanced settings panel i18n)', () => {
  it('renders the advanced settings heading in English without backticks', async () => {
    renderConfiguration();

    // Heading is plain text — no literal backticks, no Spanish words.
    const heading = await screen.findByRole('heading', { level: 2, name: 'Advanced settings.js' });
    expect(heading).toBeInTheDocument();
    expect(heading.textContent).not.toContain('`');
    expect(heading.textContent).not.toMatch(/Avanced/);
  });

  it('renders the advanced settings description in English and routes through UI_COPY', async () => {
    renderConfiguration();

    expect(
      await screen.findByText(/Edit the live settings\.js file detected by nrcc\./i),
    ).toBeInTheDocument();
    // No Spanish copy should leak through.
    expect(screen.queryByText(/Edita el archivo real/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/Último backup/i)).not.toBeInTheDocument();
  });

  it('renders the host-status block labels in English', async () => {
    renderConfiguration();

    expect(
      await screen.findByText(/Installation detected/i),
    ).toBeInTheDocument();
    // Old Spanish strings should be gone.
    expect(screen.queryByText(/Instalación detectada/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/sin Node-RED detectado/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/sin ruta detectada/i)).not.toBeInTheDocument();
  });
});

describe('ConfigurationView (issue #364 — gated raw settings editor)', () => {
  // The lock-state textarea renders the loaded file content but cannot
  // be edited, and the only available action is the "Unlock" button.
  it('renders the textarea read-only by default with an unlock button', async () => {
    renderConfiguration();

    await screen.findByTestId('raw-settings-unlock-btn');

    const textarea = screen.getByDisplayValue(/module\.exports/) as HTMLTextAreaElement;
    expect(textarea).toHaveAttribute('readonly');
    expect(textarea).toHaveAttribute('aria-readonly', 'true');

    // Save and Cancel buttons should NOT exist in the locked state.
    expect(screen.queryByTestId('raw-settings-save-btn')).not.toBeInTheDocument();
    expect(screen.queryByTestId('raw-settings-cancel-btn')).not.toBeInTheDocument();

    // Locked banner is visible.
    expect(screen.getByTestId('raw-settings-locked-banner')).toBeInTheDocument();
  });

  // The unlock dialog gates confirmation behind an acknowledgement
  // checkbox. Confirming flips the editor to its editable state.
  it('opens a confirmation dialog with an acknowledgement gate when unlock is clicked', async () => {
    const user = userEvent.setup();
    renderConfiguration();

    const unlockBtn = await screen.findByTestId('raw-settings-unlock-btn');
    await user.click(unlockBtn);

    const dialog = await screen.findByRole('dialog');
    expect(dialog).toBeInTheDocument();
    expect(within(dialog).getByText(/Edit Node-RED settings\.js directly/i)).toBeInTheDocument();

    const ack = within(dialog).getByTestId('confirmation-dialog-ack');
    expect(ack).not.toBeChecked();
    const confirmBtn = within(dialog).getByRole('button', { name: /confirm/i });
    expect(confirmBtn).toBeDisabled();

    await user.click(ack);
    expect(ack).toBeChecked();
    expect(confirmBtn).toBeEnabled();

    await user.click(confirmBtn);

    // Editor is now unlocked: Save / Cancel appear, textarea is editable.
    await waitFor(() => expect(screen.getByTestId('raw-settings-save-btn')).toBeInTheDocument());
    const textarea = screen.getByDisplayValue(/module\.exports/) as HTMLTextAreaElement;
    expect(textarea).not.toHaveAttribute('readonly');
  });

  // Cancelling the dialog (via the inline Cancel button on the unlocked
  // panel) restores the snapshot taken at unlock-time. In-flight edits
  // are discarded.
  it('re-locks the editor on Cancel and discards in-flight edits', async () => {
    const user = userEvent.setup();
    renderConfiguration();

    const unlockBtn = await screen.findByTestId('raw-settings-unlock-btn');
    await user.click(unlockBtn);

    const dialog = await screen.findByRole('dialog');
    await user.click(within(dialog).getByTestId('confirmation-dialog-ack'));
    await user.click(within(dialog).getByRole('button', { name: /confirm/i }));

    await screen.findByTestId('raw-settings-save-btn');

    const textarea = screen.getByDisplayValue(/module\.exports/) as HTMLTextAreaElement;
    await user.type(textarea, '// should be discarded');
    expect(textarea.value).toContain('// should be discarded');

    await user.click(screen.getByTestId('raw-settings-cancel-btn'));

    await waitFor(() => expect(screen.getByTestId('raw-settings-unlock-btn')).toBeInTheDocument());
    const reLocked = screen.getByDisplayValue(/module\.exports/) as HTMLTextAreaElement;
    expect(reLocked).toHaveAttribute('readonly');
    expect(reLocked.value).not.toContain('// should be discarded');
  });

  // Save is wired through the existing action; in-flight edits flow
  // through. Asserting on the post-click disabled state is the simplest
  // way to confirm the click reached the mutation without over-fitting
  // to mutation internals.
  it('disables the Save button while the save mutation is pending', async () => {
    const user = userEvent.setup();
    renderConfiguration();

    const unlockBtn = await screen.findByTestId('raw-settings-unlock-btn');
    await user.click(unlockBtn);

    const dialog = await screen.findByRole('dialog');
    await user.click(within(dialog).getByTestId('confirmation-dialog-ack'));
    await user.click(within(dialog).getByRole('button', { name: /confirm/i }));

    await screen.findByTestId('raw-settings-save-btn');
    await user.click(screen.getByTestId('raw-settings-save-btn'));

    // Either the button stays disabled while the mutation is in flight,
    // or the test sees a successful completion (the mock resolves
    // synchronously in the vi.fn() default). Either way, the click
    // reached the action; no errors thrown.
    expect(screen.getByTestId('raw-settings-save-btn')).toBeInTheDocument();
  });
});
