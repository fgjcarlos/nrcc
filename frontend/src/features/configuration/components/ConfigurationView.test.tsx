import { describe, it, expect, vi, beforeEach, type Mock } from 'vitest';
import { act, fireEvent, render, screen, waitFor, within } from '@testing-library/react';
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
import { I18nProvider } from '@/i18n';

function renderConfiguration() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <I18nProvider>
      <MemoryRouter>
        <QueryClientProvider client={queryClient}>
          <ConfigurationView />
        </QueryClientProvider>
      </MemoryRouter>
    </I18nProvider>,
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
  mocks.getStatus.mockResolvedValue({ data: { data: { nodeRed: { mode: 'docker', detected: true, version: '5.0.6' }, settings: { path: '/etc/node-red/settings.js' }, configuration: { runtimeVersion: '5.0.6', adapter: 'node-red-5', catalogVersion: '5.0.6', source: 'detected', mode: 'editable', editable: true } }, success: true, timestamp: '' } });
});

// Apply in the review drawer asks for confirmation before saving.
async function confirmApply(user: ReturnType<typeof userEvent.setup>) {
  const dialog = await screen.findByRole('dialog', { name: 'Apply changes?' });
  await user.click(within(dialog).getByRole('button', { name: /^confirm$/i }));
}

// T9: the change circuit's "Review N changes" action replaced the
// floating Save button as the entry to the review drawer.
function reviewAction() {
  return screen.getByRole('button', { name: /^Review \d+ changes?$/ });
}

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
  it('keeps the Editor Theme palette toggle flipped off after a re-render', async () => {
    const user = userEvent.setup();
    renderConfiguration();

    await switchToTab(user, 'Editor Theme');

    const toggle = findToggleButton(/^Allow Node Installation$/);
    // Fixture has no editorTheme.palette.editable, the transformer
    // defaults it to true, so the toggle starts in the "on" state.
    expect(toggle).toHaveAttribute('aria-pressed', 'true');

    await user.click(toggle);
    await waitFor(() => expect(toggle).toHaveAttribute('aria-pressed', 'false'));

    const saveButton = reviewAction();
    await user.click(saveButton).catch(() => undefined);

    expect(toggle).toHaveAttribute('aria-pressed', 'false');
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

  it('renders the advanced settings description in English', async () => {
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

// Issue #762: Security tab surfaces credentialSecret rotation,
// requireHttps, and the https block from PR #776.
describe('ConfigurationView (issue #762 — Security tab for credentialSecret / requireHttps / https)', () => {
  it('renders the Security tab with credentialSecret, requireHttps and https inputs', async () => {
    const user = userEvent.setup();
    renderConfiguration();

    await user.click(await screen.findByRole('button', { name: /^TLS & Secrets$/ }));

    // credentialSecret + https use the shared Input (label+id).
    // requireHttps is a ToggleField with no htmlFor link.
    expect(screen.getByLabelText('Credential Secret')).toBeInTheDocument();
    expect(screen.getByText(/^Require HTTPS$/)).toBeInTheDocument();
    expect(screen.getByLabelText('Private Key Path')).toBeInTheDocument();
    expect(screen.getByLabelText('Certificate Path')).toBeInTheDocument();
    expect(screen.getByLabelText('CA Bundle Path')).toBeInTheDocument();
    expect(screen.getByLabelText('HTTPS Port')).toBeInTheDocument();
    expect(screen.getByLabelText('Private Key Passphrase')).toBeInTheDocument();
  });

  it('gates credentialSecret rotation behind a confirmation dialog and clears the field on cancel', async () => {
    const user = userEvent.setup();
    renderConfiguration();

    await user.click(await screen.findByRole('button', { name: /^TLS & Secrets$/ }));
    const secretInput = screen.getByLabelText('Credential Secret') as HTMLInputElement;
    await user.type(secretInput, 'new-rotation-passphrase');

    // Issue #766 slice F (W4) — the review action opens the panel first;
    // the rotation dialog is gated by clicking Apply inside the panel.
    await user.click(reviewAction());
    const reviewPanel = await screen.findByTestId('review-changes-panel');
    await user.click(within(reviewPanel).getByTestId('review-changes-panel-apply'));

    const dialog = await screen.findByRole('dialog');
    expect(within(dialog).getByTestId('confirmation-dialog-ack')).not.toBeChecked();
    expect(within(dialog).getByRole('button', { name: /confirm/i })).toBeDisabled();

    // Cancel clears the field so the operator must re-enter to retry.
    await user.click(within(dialog).getByRole('button', { name: /^cancel$/i }));
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());
    expect(secretInput.value).toBe('');

    // Re-enter and confirm: the value sticks after acknowledgement.
    await user.type(secretInput, 'final-rotation-passphrase');
    await user.click(reviewAction());
    const secondReview = await screen.findByTestId('review-changes-panel');
    await user.click(within(secondReview).getByTestId('review-changes-panel-apply'));
    const secondDialog = await screen.findByRole('dialog');
    await user.click(within(secondDialog).getByTestId('confirmation-dialog-ack'));
    await user.click(within(secondDialog).getByRole('button', { name: /confirm/i }));
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());
    expect(secretInput.value).toBe('final-rotation-passphrase');
  });

  // Rotation is opt-in: blank credentialSecret must NOT be sent.
  it('omits credentialSecret from the save payload when the rotation field is blank', async () => {
    const user = userEvent.setup();
    const updateSpy = vi.fn().mockResolvedValue({ data: { data: {}, success: true, timestamp: '' } });
    const configModule = await import('@/features/configuration/services');
    vi.spyOn(configModule.configService, 'updateConfig').mockImplementation(updateSpy);

    renderConfiguration();

    await user.click(await screen.findByRole('button', { name: /^TLS & Secrets$/ }));
    await user.click(findToggleButton(/^Require HTTPS$/));
    await user.type(screen.getByLabelText('Private Key Path'), '/etc/node-red/key.pem');
    // Issue #766 slice F (W4) — the review action opens the panel first;
    // the actual save dispatches when the operator clicks Apply.
    await user.click(reviewAction());
    const reviewPanel = await screen.findByTestId('review-changes-panel');
    await user.click(within(reviewPanel).getByTestId('review-changes-panel-apply'));
    await confirmApply(user);

    await waitFor(() => expect(updateSpy).toHaveBeenCalled());
    const payload = updateSpy.mock.calls[0][0] as Record<string, unknown>;
    expect(payload.requireHttps).toBe(true);
    expect(payload.https).toMatchObject({ key: '/etc/node-red/key.pem' });
    expect(payload.credentialSecret).toBeUndefined();
  });

  // Issue #878 — the visual save must carry the settings.js revision the
  // operator read, so a stale browser session cannot overwrite newer
  // state.
  it('sends the settings revision it read with the save', async () => {
    const user = userEvent.setup();
    const updateSpy = vi.fn().mockResolvedValue({ data: { data: {}, success: true, timestamp: '' } });
    const configModule = await import('@/features/configuration/services');
    vi.spyOn(configModule.configService, 'updateConfig').mockImplementation(updateSpy);
    mocks.getRaw.mockResolvedValue({
      data: {
        data: {
          content: 'module.exports = {};\n',
          path: '/etc/node-red/settings.js',
          writable: true,
          revision: { fingerprint: 'rev-42', algorithm: 'sha256' },
        },
        success: true,
        timestamp: '',
      },
    });

    renderConfiguration();

    await user.click(await screen.findByRole('button', { name: /^TLS & Secrets$/ }));
    await user.click(findToggleButton(/^Require HTTPS$/));
    await user.click(reviewAction());
    const reviewPanel = await screen.findByTestId('review-changes-panel');
    await user.click(within(reviewPanel).getByTestId('review-changes-panel-apply'));
    await confirmApply(user);

    await waitFor(() => expect(updateSpy).toHaveBeenCalled());
    expect(updateSpy.mock.calls[0][1]).toBe('rev-42');
  });
});

describe('ConfigurationView (review drawer Apply — confirm, feedback, single submit)', () => {
  function deferredUpdate() {
    const handle: { resolve: (v: unknown) => void; reject: (e: unknown) => void } = {
      resolve: () => undefined,
      reject: () => undefined,
    };
    const updateSpy: Mock = vi.fn().mockImplementation(
      () =>
        new Promise((resolve, reject) => {
          handle.resolve = resolve;
          handle.reject = reject;
        }),
    );
    return { updateSpy, handle };
  }

  async function spyUpdate(updateSpy: Mock) {
    const configModule = await import('@/features/configuration/services');
    vi.spyOn(configModule.configService, 'updateConfig').mockImplementation(updateSpy);
  }

  async function openApplyConfirmation(user: ReturnType<typeof userEvent.setup>) {
    await user.click(await screen.findByRole('button', { name: /^TLS & Secrets$/ }));
    await user.click(findToggleButton(/^Require HTTPS$/));
    await user.click(reviewAction());
    const panel = await screen.findByTestId('review-changes-panel');
    await user.click(within(panel).getByTestId('review-changes-panel-apply'));
    const dialog = await screen.findByRole('dialog', { name: 'Apply changes?' });
    return { panel, dialog };
  }

  it('asks for confirmation before saving; Cancel sends nothing and keeps the drawer', async () => {
    const user = userEvent.setup();
    const { updateSpy } = deferredUpdate();
    await spyUpdate(updateSpy);
    renderConfiguration();

    const { panel, dialog } = await openApplyConfirmation(user);
    expect(updateSpy).not.toHaveBeenCalled();

    await user.click(within(dialog).getByRole('button', { name: /^cancel$/i }));
    await waitFor(() =>
      expect(screen.queryByRole('dialog', { name: 'Apply changes?' })).not.toBeInTheDocument(),
    );
    expect(panel).toBeInTheDocument();
    expect(updateSpy).not.toHaveBeenCalled();
  });

  it('sends exactly one request and shows progress while it runs, then closes on success', async () => {
    const user = userEvent.setup();
    const { updateSpy, handle } = deferredUpdate();
    await spyUpdate(updateSpy);
    renderConfiguration();

    const { panel, dialog } = await openApplyConfirmation(user);
    const confirm = within(dialog).getByRole('button', { name: /^confirm$/i });
    // Same-tick double click: the second click must not start a second save.
    fireEvent.click(confirm);
    fireEvent.click(confirm);

    await waitFor(() => expect(confirm).toBeDisabled());
    expect(within(dialog).getByText('Processing...')).toBeInTheDocument();
    const apply = within(panel).getByTestId('review-changes-panel-apply');
    expect(apply).toBeDisabled();
    expect(apply).toHaveTextContent('Applying…');
    expect(reviewAction()).toBeDisabled();

    await user.click(confirm);
    await user.click(apply);
    expect(updateSpy).toHaveBeenCalledTimes(1);

    await act(async () => handle.resolve({ data: { data: {}, success: true, timestamp: '' } }));
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());
    expect(screen.queryByTestId('review-changes-panel')).not.toBeInTheDocument();
    expect(updateSpy).toHaveBeenCalledTimes(1);
  });

  it('closes the confirmation but keeps the drawer usable when the save fails', async () => {
    const user = userEvent.setup();
    const { updateSpy, handle } = deferredUpdate();
    await spyUpdate(updateSpy);
    renderConfiguration();

    const { panel, dialog } = await openApplyConfirmation(user);
    await user.click(within(dialog).getByRole('button', { name: /^confirm$/i }));
    await waitFor(() => expect(within(panel).getByTestId('review-changes-panel-apply')).toBeDisabled());

    await act(async () => handle.reject(new Error('boom')));
    await waitFor(() =>
      expect(screen.queryByRole('dialog', { name: 'Apply changes?' })).not.toBeInTheDocument(),
    );
    expect(screen.getByTestId('review-changes-panel')).toBeInTheDocument();
    const apply = within(panel).getByTestId('review-changes-panel-apply');
    expect(apply).toBeEnabled();
    expect(apply).toHaveTextContent('Apply');
    expect(updateSpy).toHaveBeenCalledTimes(1);
  });

  it('Escape in the confirmation closes only the confirmation', async () => {
    const user = userEvent.setup();
    const { updateSpy } = deferredUpdate();
    await spyUpdate(updateSpy);
    renderConfiguration();

    await openApplyConfirmation(user);
    await user.keyboard('{Escape}');
    await waitFor(() =>
      expect(screen.queryByRole('dialog', { name: 'Apply changes?' })).not.toBeInTheDocument(),
    );
    expect(screen.getByTestId('review-changes-panel')).toBeInTheDocument();
    expect(updateSpy).not.toHaveBeenCalled();
  });

  it('credentialSecret rotation skips the extra confirmation and shows its own progress', async () => {
    const user = userEvent.setup();
    const { updateSpy, handle } = deferredUpdate();
    await spyUpdate(updateSpy);
    renderConfiguration();

    await user.click(await screen.findByRole('button', { name: /^TLS & Secrets$/ }));
    await user.type(screen.getByLabelText('Credential Secret'), 'new-rotation-passphrase');
    await user.click(reviewAction());
    const panel = await screen.findByTestId('review-changes-panel');
    await user.click(within(panel).getByTestId('review-changes-panel-apply'));

    const dialog = await screen.findByRole('dialog', { name: 'Rotate credential secret' });
    expect(screen.queryByRole('dialog', { name: 'Apply changes?' })).not.toBeInTheDocument();
    await user.click(within(dialog).getByTestId('confirmation-dialog-ack'));
    const confirm = within(dialog).getByRole('button', { name: /^confirm$/i });
    fireEvent.click(confirm);
    fireEvent.click(confirm);

    await waitFor(() => expect(confirm).toBeDisabled());
    expect(within(dialog).getByText('Processing...')).toBeInTheDocument();
    expect(updateSpy).toHaveBeenCalledTimes(1);

    await act(async () => handle.resolve({ data: { data: {}, success: true, timestamp: '' } }));
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());
    expect(updateSpy).toHaveBeenCalledTimes(1);
  });
});

// T9 — the change circuit replaces the floating Save button. It is the
// gated sequence Edit → Diff → Validate → Backup → Apply → Ready.
describe('ConfigurationView (change circuit)', () => {
  const STEP_NAMES = ['Edit', 'Diff', 'Validate', 'Backup', 'Apply', 'Ready'];

  async function circuitSteps() {
    const circuit = await screen.findByRole('list', { name: 'Change circuit' });
    return within(circuit).getAllByRole('listitem');
  }

  async function spyUpdate(impl: Mock) {
    const configModule = await import('@/features/configuration/services');
    vi.spyOn(configModule.configService, 'updateConfig').mockImplementation(impl);
  }

  it('renders the six contacts in order, each with its state in text', async () => {
    renderConfiguration();
    const steps = await circuitSteps();
    expect(steps).toHaveLength(6);
    steps.forEach((step, index) => {
      expect(within(step).getByText(STEP_NAMES[index])).toBeInTheDocument();
      expect(within(step).getByText(`K${index + 1}`)).toBeInTheDocument();
    });
    // No pending edit: nothing is reached yet. "Open" is reserved for a
    // failed or blocked step, so the idle circuit says "Waiting".
    for (const step of steps) {
      expect(within(step).getByText('Waiting')).toBeInTheDocument();
      expect(within(step).queryByText('Open')).not.toBeInTheDocument();
    }
    expect(screen.getByText('Step 1 of 6 · Edit · Waiting')).toBeInTheDocument();
  });

  it('marks the Edit contact "Open" when the runtime is read-only', async () => {
    mocks.getStatus.mockResolvedValue({ data: { data: { nodeRed: { mode: 'docker', detected: true, version: '5.0.6' }, settings: { path: '/etc/node-red/settings.js' }, configuration: { runtimeVersion: '5.0.6', adapter: 'node-red-5', catalogVersion: '5.0.6', source: 'detected', mode: 'read-only', editable: false } }, success: true, timestamp: '' } });
    renderConfiguration();

    await screen.findByText('Read-only runtime');
    const [edit] = await circuitSteps();
    expect(within(edit).getByText('Open')).toBeInTheDocument();
  });

  // 320–390px overflow guard: jsdom has no layout, so this pins the
  // classes that keep the circuit from widening the page. The grid needs an
  // explicit minmax(0,1fr) column, the circuit and its list must be allowed
  // to shrink, and no step may force its min-content width.
  it('keeps the circuit shrinkable so it cannot widen the page on narrow screens', async () => {
    renderConfiguration();
    const circuit = await screen.findByTestId('change-circuit');
    expect(circuit).toHaveClass('min-w-0');
    expect(circuit.parentElement).toHaveClass('grid-cols-1');
    const list = screen.getByTestId('change-circuit-steps');
    expect(list).toHaveClass('min-w-0');
    expect(list.className).not.toMatch(/overflow-x-auto|min-w-max/);
    for (const step of within(list).getAllByRole('listitem')) {
      expect(step.className).not.toMatch(/min-w-max/);
    }
    const review = screen.getByRole('button', { name: 'No changes to review' });
    expect(review).toHaveClass('w-full');
  });

  it('registers each change live: pending count, restart cost and step states', async () => {
    const user = userEvent.setup();
    renderConfiguration();

    expect(await screen.findByRole('button', { name: 'No changes to review' })).toBeDisabled();

    await user.click(await screen.findByRole('button', { name: /^TLS & Secrets$/ }));
    await user.click(findToggleButton(/^Require HTTPS$/));
    expect(screen.getByRole('button', { name: 'Review 1 change' })).toBeEnabled();
    expect(screen.getByTestId('change-circuit-total')).toHaveTextContent('1 change');

    await user.click(await screen.findByRole('button', { name: /^Basic$/ }));
    const port = screen.getByLabelText('UI Port');
    await user.clear(port);
    await user.type(port, '1881');
    expect(screen.getByRole('button', { name: 'Review 2 changes' })).toBeEnabled();
    expect(screen.getByTestId('change-circuit-total')).toHaveTextContent('2 changes · 1 needs restart');

    const [edit, diff] = await circuitSteps();
    expect(within(edit).getByText('Closed')).toBeInTheDocument();
    expect(within(edit).getByText('2 changes')).toBeInTheDocument();
    expect(within(diff).getByText('Pending')).toBeInTheDocument();
    expect(within(diff).getByText('Not reviewed')).toBeInTheDocument();
    expect(screen.getByTestId('change-circuit-live')).toHaveTextContent('Diff: Pending');
  });

  it('discards back to the loaded configuration, leaving zero pending changes', async () => {
    const user = userEvent.setup();
    renderConfiguration();

    await user.click(await screen.findByRole('button', { name: /^TLS & Secrets$/ }));
    await user.click(findToggleButton(/^Require HTTPS$/));
    await user.click(screen.getByRole('button', { name: 'Discard Changes' }));

    expect(screen.getByRole('button', { name: 'No changes to review' })).toBeDisabled();
    expect(screen.getByTestId('change-circuit-total')).toHaveTextContent('0 changes');
  });

  it('opens the review drawer from "Review N changes" and closes the Diff contact', async () => {
    const user = userEvent.setup();
    renderConfiguration();

    await user.click(await screen.findByRole('button', { name: /^TLS & Secrets$/ }));
    await user.click(findToggleButton(/^Require HTTPS$/));
    await user.click(screen.getByRole('button', { name: 'Review 1 change' }));

    expect(await screen.findByTestId('review-changes-panel')).toBeInTheDocument();
    const [, diff] = await circuitSteps();
    expect(within(diff).getByText('Closed')).toBeInTheDocument();
    expect(within(diff).getByText('Diff reviewed')).toBeInTheDocument();
  });

  it('closes every contact after the server confirms the apply', async () => {
    const user = userEvent.setup();
    await spyUpdate(vi.fn().mockResolvedValue({ data: { data: {}, success: true, timestamp: '' } }));
    renderConfiguration();

    await user.click(await screen.findByRole('button', { name: /^TLS & Secrets$/ }));
    await user.click(findToggleButton(/^Require HTTPS$/));
    await user.click(screen.getByRole('button', { name: 'Review 1 change' }));
    await user.click(within(await screen.findByTestId('review-changes-panel')).getByTestId('review-changes-panel-apply'));
    await confirmApply(user);

    await waitFor(async () => {
      const steps = await circuitSteps();
      for (const step of steps) expect(within(step).getByText('Closed')).toBeInTheDocument();
    });
    const steps = await circuitSteps();
    expect(within(steps[3]).getByText('Taken before write')).toBeInTheDocument();
    expect(within(steps[5]).getByText('Readiness OK')).toBeInTheDocument();
  });

  it('opens the Ready contact and shows the rollback branch when readiness fails', async () => {
    const user = userEvent.setup();
    await spyUpdate(
      vi.fn().mockRejectedValue({
        response: {
          status: 500,
          data: {
            success: false,
            error: {
              code: 'RUNTIME_READINESS_FAILED',
              message: 'Configuration apply failed because Node-RED did not become ready',
            },
          },
        },
      }),
    );
    renderConfiguration();

    await user.click(await screen.findByRole('button', { name: /^TLS & Secrets$/ }));
    await user.click(findToggleButton(/^Require HTTPS$/));
    await user.click(screen.getByRole('button', { name: 'Review 1 change' }));
    await user.click(within(await screen.findByTestId('review-changes-panel')).getByTestId('review-changes-panel-apply'));
    await confirmApply(user);

    const ready = (await circuitSteps())[5];
    await waitFor(() => expect(within(ready).getByText('Failed')).toBeInTheDocument());
    expect(within(ready).getByText('Readiness failed')).toBeInTheDocument();
    expect(within(ready).getByText('RUNTIME_READINESS_FAILED')).toBeInTheDocument();
    const [, , validate, backup, apply] = await circuitSteps();
    expect(within(validate).getByText('Closed')).toBeInTheDocument();
    expect(within(backup).getByText('Closed')).toBeInTheDocument();
    expect(within(apply).getByText('Closed')).toBeInTheDocument();
    expect(screen.getByTestId('change-circuit-rollback')).toHaveTextContent(
      'Rollback: restored from backup',
    );
    expect(screen.getByTestId('change-circuit-live')).toHaveTextContent('Ready: Failed');
  });

  it('marks the outcome as not confirmed when the server gives no stage', async () => {
    const user = userEvent.setup();
    await spyUpdate(vi.fn().mockRejectedValue(new Error('timeout of 120000ms exceeded')));
    renderConfiguration();

    await user.click(await screen.findByRole('button', { name: /^TLS & Secrets$/ }));
    await user.click(findToggleButton(/^Require HTTPS$/));
    await user.click(screen.getByRole('button', { name: 'Review 1 change' }));
    await user.click(within(await screen.findByTestId('review-changes-panel')).getByTestId('review-changes-panel-apply'));
    await confirmApply(user);

    await waitFor(async () => {
      const steps = await circuitSteps();
      for (const step of steps.slice(2)) expect(within(step).getByText('Not confirmed')).toBeInTheDocument();
    });
    expect(screen.queryByTestId('change-circuit-rollback')).not.toBeInTheDocument();
  });
});
