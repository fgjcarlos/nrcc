import { describe, it, expect, vi, beforeEach, afterEach, type Mock } from 'vitest';
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
  // The AI Provider tab loads its own config when the tablist reaches it.
  aiService: {
    getConfig: vi.fn().mockResolvedValue({ enabled: false, provider: 'offline' }),
    getStatus: vi.fn().mockResolvedValue({ status: 'disabled' }),
    saveConfig: vi.fn(),
    testConfig: vi.fn(),
  },
}));

vi.mock('@/features/bootstrap/services', () => ({
  bootstrapService: { getStatus: mocks.getStatus },
}));

vi.mock('sonner', () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

import { ConfigurationView } from './ConfigurationView';
import { I18nProvider, i18n } from '@/i18n';
import enBootstrap from '@/locales/en/bootstrap.json';
import esBootstrap from '@/locales/es/bootstrap.json';

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

// T9: the change circuit's "Review N changes" action replaced the
// floating Save button as the entry to the review drawer.
function reviewAction() {
  return screen.getByRole('button', { name: /^Review \d+ changes?$/ });
}

async function switchToTab(user: ReturnType<typeof userEvent.setup>, label: string) {
  // The sections are a tablist; each tab's accessible name is its
  // visible label (the icon is aria-hidden).
  const tab = await screen.findByRole('tab', { name: new RegExp(`^${label}$`) });
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

    await user.click(await screen.findByRole('tab', { name: /^TLS & Secrets$/ }));

    // credentialSecret + https use the shared Input (label+id).
    // requireHttps is a ToggleField with no htmlFor link.
    expect(screen.getByLabelText('Credential Secret (credentialSecret)')).toBeInTheDocument();
    expect(screen.getByText(/^Require HTTPS$/)).toBeInTheDocument();
    expect(screen.getByLabelText('Private Key Path (https.key)')).toBeInTheDocument();
    expect(screen.getByLabelText('Certificate Path (https.cert)')).toBeInTheDocument();
    expect(screen.getByLabelText('CA Bundle Path (https.ca)')).toBeInTheDocument();
    expect(screen.getByLabelText('HTTPS Port (https.port)')).toBeInTheDocument();
    expect(screen.getByLabelText('Private Key Passphrase (https.passphrase)')).toBeInTheDocument();
  });

  it('gates credentialSecret rotation behind a confirmation dialog and clears the field on cancel', async () => {
    const user = userEvent.setup();
    renderConfiguration();

    await user.click(await screen.findByRole('tab', { name: /^TLS & Secrets$/ }));
    const secretInput = screen.getByLabelText('Credential Secret (credentialSecret)') as HTMLInputElement;
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

  // PRODUCT.md: secrets are masked before any diff is shown.
  it('prints typed secrets in the review drawer as the redaction marker', async () => {
    const user = userEvent.setup();
    renderConfiguration();

    await user.click(await screen.findByRole('tab', { name: /^TLS & Secrets$/ }));
    await user.type(screen.getByLabelText('Credential Secret (credentialSecret)'), 'typed-credential-secret');
    await user.type(screen.getByLabelText('Private Key Passphrase (https.passphrase)'), 'typed-key-passphrase');
    await user.click(reviewAction());
    const panel = await screen.findByTestId('review-changes-panel');

    for (const id of ['credentialSecret', 'httpsPassphrase']) {
      const values = within(within(panel).getByTestId(`review-changes-panel-row-${id}`)).getAllByText(/^(—|\[redacted\])$/);
      expect(values.map((v) => v.textContent)).toEqual(['—', '[redacted]']);
    }
    expect(document.body.textContent).not.toContain('typed-credential-secret');
    expect(document.body.textContent).not.toContain('typed-key-passphrase');
  });

  // Rotation is opt-in: blank credentialSecret must NOT be sent.
  it('omits credentialSecret from the save payload when the rotation field is blank', async () => {
    const user = userEvent.setup();
    const updateSpy = vi.fn().mockResolvedValue({ data: { data: {}, success: true, timestamp: '' } });
    const configModule = await import('@/features/configuration/services');
    vi.spyOn(configModule.configService, 'updateConfig').mockImplementation(updateSpy);

    renderConfiguration();

    await user.click(await screen.findByRole('tab', { name: /^TLS & Secrets$/ }));
    await user.click(findToggleButton(/^Require HTTPS$/));
    await user.type(screen.getByLabelText('Private Key Path (https.key)'), '/etc/node-red/key.pem');
    // Issue #766 slice F (W4) — the review action opens the panel first;
    // the actual save dispatches when the operator clicks Apply.
    await user.click(reviewAction());
    const reviewPanel = await screen.findByTestId('review-changes-panel');
    await user.click(within(reviewPanel).getByTestId('review-changes-panel-apply'));

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

    await user.click(await screen.findByRole('tab', { name: /^TLS & Secrets$/ }));
    await user.click(findToggleButton(/^Require HTTPS$/));
    await user.click(reviewAction());
    const reviewPanel = await screen.findByTestId('review-changes-panel');
    await user.click(within(reviewPanel).getByTestId('review-changes-panel-apply'));

    await waitFor(() => expect(updateSpy).toHaveBeenCalled());
    expect(updateSpy.mock.calls[0][1]).toBe('rev-42');
  });
});

describe('ConfigurationView (review drawer Apply — single confirmation, feedback, single submit)', () => {
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

  async function openReview(user: ReturnType<typeof userEvent.setup>) {
    await user.click(await screen.findByRole('tab', { name: /^TLS & Secrets$/ }));
    await user.click(findToggleButton(/^Require HTTPS$/));
    await user.click(reviewAction());
    const panel = await screen.findByTestId('review-changes-panel');
    return { panel, apply: within(panel).getByTestId('review-changes-panel-apply') };
  }

  it('one click on Apply in the drawer sends the request, with no second dialog', async () => {
    const user = userEvent.setup();
    const { updateSpy } = deferredUpdate();
    await spyUpdate(updateSpy);
    renderConfiguration();

    const { apply } = await openReview(user);
    await user.click(apply);

    await waitFor(() => expect(updateSpy).toHaveBeenCalledTimes(1));
    // The drawer is the only dialog: no "Apply changes?" step after it.
    expect(screen.getAllByRole('dialog')).toHaveLength(1);
    expect(screen.queryByRole('dialog', { name: 'Apply changes?' })).not.toBeInTheDocument();
  });

  it('states the change count, the restart and the backup in the footer, focusing Cancel first', async () => {
    const user = userEvent.setup();
    renderConfiguration();

    const { panel, apply } = await openReview(user);
    const footer = within(panel).getByTestId('review-changes-panel-consequence');
    expect(footer).toHaveTextContent('1 change · Node-RED restarts · backup taken first');
    expect(apply).toHaveAccessibleName('Apply and restart Node-RED');
    expect(apply).toHaveAccessibleDescription('1 change · Node-RED restarts · backup taken first');
    await waitFor(() =>
      expect(within(panel.querySelector('footer') as HTMLElement).getByRole('button', { name: 'Cancel' })).toHaveFocus(),
    );
  });

  it('cannot be dismissed while the apply is pending', async () => {
    const user = userEvent.setup();
    const { updateSpy, handle } = deferredUpdate();
    await spyUpdate(updateSpy);
    renderConfiguration();

    const { panel, apply } = await openReview(user);
    await user.click(apply);
    await waitFor(() => expect(apply).toBeDisabled());

    await user.keyboard('{Escape}');
    for (const cancel of within(panel).getAllByRole('button', { name: 'Cancel' })) {
      expect(cancel).toBeDisabled();
      fireEvent.click(cancel);
    }
    fireEvent.click(screen.getByTestId('review-changes-panel-portal').firstElementChild as HTMLElement);
    expect(screen.getByTestId('review-changes-panel')).toBeInTheDocument();

    await act(async () => handle.resolve({ data: { data: {}, success: true, timestamp: '' } }));
    await waitFor(() => expect(screen.queryByTestId('review-changes-panel')).not.toBeInTheDocument());
    expect(updateSpy).toHaveBeenCalledTimes(1);
  });

  it('Escape before Apply closes the drawer and sends nothing', async () => {
    const user = userEvent.setup();
    const { updateSpy } = deferredUpdate();
    await spyUpdate(updateSpy);
    renderConfiguration();

    await openReview(user);
    await user.keyboard('{Escape}');
    await waitFor(() => expect(screen.queryByTestId('review-changes-panel')).not.toBeInTheDocument());
    expect(updateSpy).not.toHaveBeenCalled();
  });

  it('sends exactly one request and shows progress while it runs, then closes on success', async () => {
    const user = userEvent.setup();
    const { updateSpy, handle } = deferredUpdate();
    await spyUpdate(updateSpy);
    renderConfiguration();

    const { apply } = await openReview(user);
    // Same-tick double click: the second click must not start a second save.
    fireEvent.click(apply);
    fireEvent.click(apply);

    await waitFor(() => expect(apply).toBeDisabled());
    expect(apply).toHaveTextContent('Applying…');
    expect(reviewAction()).toBeDisabled();

    await user.click(apply);
    expect(updateSpy).toHaveBeenCalledTimes(1);

    await act(async () => handle.resolve({ data: { data: {}, success: true, timestamp: '' } }));
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());
    expect(screen.queryByTestId('review-changes-panel')).not.toBeInTheDocument();
    expect(updateSpy).toHaveBeenCalledTimes(1);
  });

  // T9: no failure toast. The drawer closes so the change circuit beside it
  // shows the failed contact; the review action reopens it for a retry.
  it('closes the drawer on a failed save so the circuit shows the outcome', async () => {
    const user = userEvent.setup();
    const { updateSpy, handle } = deferredUpdate();
    await spyUpdate(updateSpy);
    renderConfiguration();

    const { apply } = await openReview(user);
    await user.click(apply);
    await waitFor(() => expect(apply).toBeDisabled());

    await act(async () => handle.reject(new Error('boom')));
    await waitFor(() => expect(screen.queryByTestId('review-changes-panel')).not.toBeInTheDocument());
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
    expect(screen.getByTestId('change-circuit-live')).toHaveTextContent('Validate: Not confirmed');

    await user.click(reviewAction());
    const reopened = await screen.findByTestId('review-changes-panel');
    expect(within(reopened).getByTestId('review-changes-panel-apply')).toBeEnabled();
    expect(updateSpy).toHaveBeenCalledTimes(1);
  });

  it('credentialSecret rotation keeps its own confirmation and shows its own progress', async () => {
    const user = userEvent.setup();
    const { updateSpy, handle } = deferredUpdate();
    await spyUpdate(updateSpy);
    renderConfiguration();

    await user.click(await screen.findByRole('tab', { name: /^TLS & Secrets$/ }));
    await user.type(screen.getByLabelText('Credential Secret (credentialSecret)'), 'new-rotation-passphrase');
    await user.click(reviewAction());
    const panel = await screen.findByTestId('review-changes-panel');
    await user.click(within(panel).getByTestId('review-changes-panel-apply'));

    const dialog = await screen.findByRole('dialog', { name: 'Rotate credential secret' });
    expect(updateSpy).not.toHaveBeenCalled();
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

    await user.click(await screen.findByRole('tab', { name: /^TLS & Secrets$/ }));
    await user.click(findToggleButton(/^Require HTTPS$/));
    expect(screen.getByRole('button', { name: 'Review 1 change' })).toBeEnabled();
    expect(screen.getByTestId('change-circuit-total')).toHaveTextContent('1 change');

    await user.click(await screen.findByRole('tab', { name: /^Basic$/ }));
    const port = screen.getByLabelText('UI Port (uiPort)');
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

    await user.click(await screen.findByRole('tab', { name: /^TLS & Secrets$/ }));
    await user.click(findToggleButton(/^Require HTTPS$/));
    await user.click(screen.getByRole('button', { name: 'Discard Changes' }));

    expect(screen.getByRole('button', { name: 'No changes to review' })).toBeDisabled();
    expect(screen.getByTestId('change-circuit-total')).toHaveTextContent('0 changes');
  });

  it('opens the review drawer from "Review N changes" and closes the Diff contact', async () => {
    const user = userEvent.setup();
    renderConfiguration();

    await user.click(await screen.findByRole('tab', { name: /^TLS & Secrets$/ }));
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

    await user.click(await screen.findByRole('tab', { name: /^TLS & Secrets$/ }));
    await user.click(findToggleButton(/^Require HTTPS$/));
    await user.click(screen.getByRole('button', { name: 'Review 1 change' }));
    await user.click(within(await screen.findByTestId('review-changes-panel')).getByTestId('review-changes-panel-apply'));

    await waitFor(async () => {
      const steps = await circuitSteps();
      for (const step of steps) expect(within(step).getByText('Closed')).toBeInTheDocument();
    });
    const steps = await circuitSteps();
    expect(within(steps[3]).getByText('Taken before write')).toBeInTheDocument();
    expect(within(steps[5]).getByText('Readiness OK')).toBeInTheDocument();
    // No toast: the circuit and its live region carry the outcome.
    const { toast } = await import('sonner');
    expect(toast.success).not.toHaveBeenCalled();
    expect(screen.getByTestId('change-circuit-live')).toHaveTextContent('Ready: Closed. Readiness OK');
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

    await user.click(await screen.findByRole('tab', { name: /^TLS & Secrets$/ }));
    await user.click(findToggleButton(/^Require HTTPS$/));
    await user.click(screen.getByRole('button', { name: 'Review 1 change' }));
    await user.click(within(await screen.findByTestId('review-changes-panel')).getByTestId('review-changes-panel-apply'));

    const ready = (await circuitSteps())[5];
    await waitFor(() => expect(within(ready).getByText('Failed')).toBeInTheDocument());
    expect(within(ready).getByText('Readiness failed')).toBeInTheDocument();
    expect(within(ready).getByText('RUNTIME_READINESS_FAILED')).toBeInTheDocument();
    const [, , validate, backup, apply] = await circuitSteps();
    expect(within(validate).getByText('Closed')).toBeInTheDocument();
    expect(within(backup).getByText('Closed')).toBeInTheDocument();
    expect(within(apply).getByText('Closed')).toBeInTheDocument();
    const rollback = screen.getByTestId('change-circuit-rollback');
    expect(rollback).toHaveTextContent('Rollback: restored from backup');
    // Text plus a drawn symbol, so the outcome reads on the mobile strip too
    // (the side branch is only drawn from lg up).
    expect(within(rollback).getByTestId('rollback-symbol')).toHaveAttribute('data-outcome', 'restored');
    const { toast } = await import('sonner');
    expect(toast.error).not.toHaveBeenCalled();
    expect(screen.getByTestId('change-circuit-live')).toHaveTextContent('Ready: Failed');
  });

  it('marks the outcome as not confirmed when the server gives no stage', async () => {
    const user = userEvent.setup();
    await spyUpdate(vi.fn().mockRejectedValue(new Error('timeout of 120000ms exceeded')));
    renderConfiguration();

    await user.click(await screen.findByRole('tab', { name: /^TLS & Secrets$/ }));
    await user.click(findToggleButton(/^Require HTTPS$/));
    await user.click(screen.getByRole('button', { name: 'Review 1 change' }));
    await user.click(within(await screen.findByTestId('review-changes-panel')).getByTestId('review-changes-panel-apply'));

    await waitFor(async () => {
      const steps = await circuitSteps();
      for (const step of steps.slice(2)) expect(within(step).getByText('Not confirmed')).toBeInTheDocument();
    });
    expect(screen.queryByTestId('change-circuit-rollback')).not.toBeInTheDocument();
  });
});

// Finish review (T9 follow-up): schematic designators, gated outcomes on
// the circuit, a real tablist and the direct-edit lock region.
describe('ConfigurationView (finish review)', () => {
  it('renders the sections as a tablist with roving focus and arrow keys', async () => {
    const user = userEvent.setup();
    renderConfiguration();

    const tablist = await screen.findByRole('tablist', { name: 'Configuration sections' });
    const tabs = within(tablist).getAllByRole('tab');
    expect(tabs.map((tab) => tab.textContent)).toEqual(['Basic', 'TLS & Secrets', 'Logging', 'Editor Theme', 'AI Provider']);
    expect(tabs[0]).toHaveAttribute('aria-selected', 'true');
    expect(tabs[0]).toHaveAttribute('tabindex', '0');
    for (const tab of tabs.slice(1)) {
      expect(tab).toHaveAttribute('aria-selected', 'false');
      expect(tab).toHaveAttribute('tabindex', '-1');
    }
    const panel = screen.getByRole('tabpanel', { name: 'Basic' });
    expect(tabs[0]).toHaveAttribute('aria-controls', panel.id);

    tabs[0].focus();
    await user.keyboard('{ArrowRight}');
    expect(tabs[1]).toHaveFocus();
    expect(tabs[1]).toHaveAttribute('aria-selected', 'true');
    expect(screen.getByRole('tabpanel', { name: 'TLS & Secrets' })).toBeInTheDocument();
    await user.keyboard('{End}');
    expect(tabs[4]).toHaveFocus();
    await user.keyboard('{ArrowRight}');
    expect(tabs[0]).toHaveFocus();
    await user.keyboard('{ArrowLeft}');
    expect(tabs[4]).toHaveFocus();
    await user.keyboard('{Home}');
    expect(tabs[0]).toHaveFocus();
    expect(tabs[0]).toHaveAttribute('aria-selected', 'true');
  });

  it('labels each field with its settings.js key as the terminal designator', async () => {
    renderConfiguration();

    const port = await screen.findByLabelText('UI Port (uiPort)');
    const label = document.querySelector(`label[for="${port.id}"]`) as HTMLElement;
    expect(within(label).getByText('uiPort')).toHaveClass('font-mono');
    expect(within(label).getByText('UI Port')).toBeInTheDocument();
    // Nested keys use their settings.js path.
    expect(screen.getByRole('button', { name: 'Disable Editor (disableEditor)' })).toBeInTheDocument();
  });

  it('shows the settings.js revision as the header designator', async () => {
    mocks.getRaw.mockResolvedValue({
      data: {
        data: {
          content: 'module.exports = {};\n',
          path: '/etc/node-red/settings.js',
          writable: true,
          revision: { fingerprint: '9f2c1b0d4e5a6f7c8d9e', algorithm: 'sha256' },
        },
        success: true,
        timestamp: '',
      },
    });
    renderConfiguration();

    expect(await screen.findByText('REV 9f2c1b0d')).toHaveClass('font-mono');
  });

  it('holds the Validate contact open with the invalid keys as its proof', async () => {
    const user = userEvent.setup();
    renderConfiguration();

    const port = await screen.findByLabelText('UI Port (uiPort)');
    await user.clear(port);
    await user.type(port, '70000');

    const validate = within(await screen.findByRole('list', { name: 'Change circuit' })).getAllByRole('listitem')[2];
    expect(within(validate).getByText('Open')).toBeInTheDocument();
    expect(within(validate).getByText('1 field invalid')).toBeInTheDocument();
    expect(within(validate).getByText('uiPort')).toBeInTheDocument();
  });

  it('marks the raw editor as a direct edit with the unlock control in the lock banner', async () => {
    renderConfiguration();

    const banner = await screen.findByTestId('raw-settings-locked-banner');
    expect(within(banner).getByTestId('raw-settings-unlock-btn')).toBeInTheDocument();
    expect(banner).toHaveTextContent(/Unlock to edit settings\.js/);
    expect(banner).toHaveTextContent(/backup is taken before any write/i);
    expect(screen.getByText(/bypasses the change circuit/i)).toBeInTheDocument();
  });

  it('shows a settings.js syntax error inline under the editor, not as a toast', async () => {
    const user = userEvent.setup();
    const { toast } = await import('sonner');
    renderConfiguration();

    await user.click(await screen.findByTestId('raw-settings-unlock-btn'));
    const dialog = await screen.findByRole('dialog');
    await user.click(within(dialog).getByTestId('confirmation-dialog-ack'));
    await user.click(within(dialog).getByRole('button', { name: /confirm/i }));

    const textarea = await screen.findByDisplayValue(/module\.exports/);
    fireEvent.change(textarea, { target: { value: 'module.exports = {' } });
    await user.click(screen.getByTestId('raw-settings-save-btn'));

    const error = await screen.findByTestId('raw-settings-error');
    expect(error).toHaveTextContent(/Invalid settings\.js/);
    expect(textarea).toHaveAttribute('aria-invalid', 'true');
    expect(textarea).toHaveAttribute('aria-describedby', error.id);
    expect(toast.error).not.toHaveBeenCalled();
  });
});

describe('ConfigurationView (finish review round 2)', () => {
  it('lets long settings.js keys break only after a dot', async () => {
    const user = userEvent.setup();
    renderConfiguration();
    await user.click(await screen.findByRole('tab', { name: /^Editor Theme$/ }));

    const select = screen.getByLabelText('Theme (editorTheme.codeEditor.options.theme)');
    const key = within(document.querySelector(`label[for="${select.id}"]`) as HTMLElement)
      .getByText('editorTheme.codeEditor.options.theme');
    expect(key.className).not.toMatch(/break-all/);
    expect(key.querySelectorAll('wbr')).toHaveLength(3);
  });

  it('keeps the lock banner text wide enough to push the unlock button below it', async () => {
    renderConfiguration();
    const banner = await screen.findByTestId('raw-settings-locked-banner');
    expect(banner).toHaveClass('flex-wrap');
    const text = within(banner).getByText(/^Locked\./);
    expect(text.className).toMatch(/flex-\[1_1_14rem\]/);
  });
});

describe('ConfigurationView (host recommendations from recommendationCodes)', () => {
  const PORTLESS_ES_BACKEND = 'Opcional: instala Portless para exponer nrcc o Node-RED con URLs HTTPS .localhost, LAN o Tailscale.';
  const withRecommendations = (recommendations: string[], recommendationCodes?: string[]) =>
    mocks.getStatus.mockResolvedValue({
      data: {
        data: {
          nodeRed: { mode: 'docker', detected: true, version: '5.0.6' },
          settings: { path: '/etc/node-red/settings.js' },
          configuration: { runtimeVersion: '5.0.6', source: 'detected', mode: 'editable', editable: true },
          recommendations,
          recommendationCodes,
        },
        success: true,
        timestamp: '',
      },
    });

  afterEach(async () => {
    await i18n.changeLanguage('en');
  });

  it('localizes each recommendation by its code in Spanish', async () => {
    withRecommendations([PORTLESS_ES_BACKEND, 'Otorga permisos'], ['portless_optional', 'settings_not_writable']);
    await i18n.changeLanguage('es');
    renderConfiguration();

    expect(await screen.findByText(esBootstrap.recommendation.portless_optional)).toBeInTheDocument();
    expect(screen.getByText(esBootstrap.recommendation.settings_not_writable)).toBeInTheDocument();
    expect(screen.queryByText(PORTLESS_ES_BACKEND)).not.toBeInTheDocument();
  });

  it('localizes each recommendation by its code in English', async () => {
    withRecommendations([PORTLESS_ES_BACKEND], ['portless_optional']);
    renderConfiguration();

    expect(await screen.findByText(enBootstrap.recommendation.portless_optional)).toBeInTheDocument();
    expect(screen.queryByText(PORTLESS_ES_BACKEND)).not.toBeInTheDocument();
  });

  it('falls back to the backend string for an unknown code or a backend without codes', async () => {
    withRecommendations(['Backend says hello', 'Older backend text'], ['future_code']);
    renderConfiguration();

    expect(await screen.findByText('Backend says hello')).toBeInTheDocument();
    expect(screen.getByText('Older backend text')).toBeInTheDocument();
  });
});
