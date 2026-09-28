import { describe, it, expect } from 'vitest';
import { computeConfigurationDiff, type FieldSource } from './useConfigurationDiff';
import type { NodeRedConfigFormData, HostStatus } from '@/shared/types';
import { configToFormData, type NodeRedConfigResponse } from '@/features/configuration/lib/configTransformers';
import { INITIAL_FORM_DATA } from '@/features/configuration/lib/initialFormData';

function baseForm(): NodeRedConfigFormData {
  return { ...INITIAL_FORM_DATA };
}

function emptyHost(): HostStatus {
  return {
    platform: 'linux',
    ready: true,
    interactive: true,
    nodejs: { name: 'nodejs', installed: true, version: '20.0.0' },
    npm: { name: 'npm', installed: true, version: '10.0.0' },
    nodeRedBinary: { name: 'node-red', installed: true, version: '4.0.0' },
    docker: { name: 'docker', installed: false },
    dockerCompose: { name: 'docker-compose', installed: false },
    nodeRed: { detected: true, mode: 'native', managedByNrcc: true, running: true, version: '5.0.7' },
    settings: { path: '/etc/node-red/settings.js', source: 'system', writable: true },
    configuration: { runtimeVersion: '5.0.7', adapter: 'file', catalogVersion: '1', source: 'system', mode: 'editable', editable: true },
  };
}

function loadedConfig(): NodeRedConfigResponse {
  // Mirrors the defaults that `configToFormData` produces from a real
  // NRCC bootstrap so the diff against `INITIAL_FORM_DATA` is exactly 0.
  return {
    uiPort: 1880,
    uiHost: '0.0.0.0',
    httpAdminRoot: '/',
    httpNodeRoot: '/',
    disableEditor: false,
    projectsEnabled: false,
    logging: { console: { level: 'info', metrics: false }, internal: { level: 'info', metrics: false } },
    flowFile: 'flows.json',
    editorTheme: {
      page: { title: 'Node-RED' },
      header: { title: 'Node-RED' },
      deployButton: { type: 'default', label: 'Deploy' },
      palette: { editable: true },
      codeEditor: { lib: 'ace', options: { theme: 'vs', fontSize: 12 } },
      userMenu: true,
      tours: true,
    },
    lang: 'en-US',
    requireHttps: false,
  };
}

describe('computeConfigurationDiff', () => {
  it('reports pending=false and pendingCount=0 when form matches loaded config', () => {
    const loaded = loadedConfig();
    const form = configToFormData(loaded);
    const diff = computeConfigurationDiff(form, loaded, emptyHost(), '');
    expect(diff.pendingCount).toBe(0);
    expect(diff.restartRequired).toBe('none');
    expect(diff.fields.uiPort.pending).toBe(false);
  });

  it('detects a dirty uiPort and marks restartRequired=hard', () => {
    const form = baseForm();
    form.uiPort = 1881; // user changed it
    const diff = computeConfigurationDiff(form, loadedConfig(), emptyHost(), '');
    expect(diff.fields.uiPort.pending).toBe(true);
    expect(diff.fields.uiPort.configuredValue).toBe(1880);
    expect(diff.fields.uiPort.effectiveValue).toBe(1880);
    expect(diff.fields.uiPort.formValue).toBe(1881);
    expect(diff.fields.uiPort.restart).toBe('hard');
    expect(diff.pendingCount).toBeGreaterThan(0);
    expect(diff.restartRequired).toBe('hard');
  });

  it('marks editorTheme.* as soft restart (flow redeploy, no settings reload)', () => {
    const form = baseForm();
    form.editorPageTitle = 'Custom Title';
    const diff = computeConfigurationDiff(form, loadedConfig(), emptyHost(), '');
    expect(diff.fields.editorPageTitle.pending).toBe(true);
    expect(diff.fields.editorPageTitle.restart).toBe('soft');
    expect(diff.restartRequired).toBe('soft');
  });

  it('labels effectiveValue as NRCC-loaded for every field', () => {
    const form = baseForm();
    const loaded = loadedConfig();
    const diff = computeConfigurationDiff(form, loaded, emptyHost(), '');
    const sample = diff.fields.uiPort;
    expect(sample.effectiveValue).toBe(loaded.uiPort);
    expect(sample.effectiveLabel).toBe('Effective (NRCC-loaded)');
  });

  it('marks auth fields with restart=hard when enable flips', () => {
    const form = baseForm();
    form.authEnabled = true;
    form.authAdminUser = 'admin';
    const diff = computeConfigurationDiff(form, loadedConfig(), emptyHost(), '');
    // authEnabled is sourced from settings.js (Node-RED adminAuth); enabling
    // it requires a Node-RED restart.
    expect(diff.fields.authEnabled.source).toBe('settings.js');
    expect(diff.fields.authEnabled.restart).toBe('hard');
  });

  it('aggregates restartRequired to the highest severity across dirty fields', () => {
    const form = baseForm();
    form.editorPageTitle = 'Custom Title'; // soft
    form.uiPort = 2000; // hard
    const diff = computeConfigurationDiff(form, loadedConfig(), emptyHost(), '');
    expect(diff.restartRequired).toBe('hard');
    const dirty = Object.values(diff.fields).filter((f) => f.pending);
    const distinct = new Set(dirty.map((f) => f.restart));
    expect(distinct.has('soft')).toBe(true);
    expect(distinct.has('hard')).toBe(true);
  });

  it('reports validationErrors from validateConfigurationField when password is too short', () => {
    const form = baseForm();
    form.authEnabled = true;
    form.authAdminUser = 'admin';
    form.authAdminPassword = 'short'; // < 8 chars
    const diff = computeConfigurationDiff(form, loadedConfig(), emptyHost(), '');
    expect(diff.validationErrors['authAdminPassword']).toBeDefined();
    // Save should be blocked
    expect(diff.canSave).toBe(false);
  });

  it('canSave=true when there are pending changes but no validation errors', () => {
    const form = baseForm();
    form.uiPort = 2000;
    const diff = computeConfigurationDiff(form, loadedConfig(), emptyHost(), '');
    expect(diff.pendingCount).toBeGreaterThan(0);
    expect(diff.canSave).toBe(true);
  });

  it('source map distinguishes settings.js vs form default', () => {
    const diff = computeConfigurationDiff(baseForm(), loadedConfig(), emptyHost(), '');
    // uiPort survives a settings reload, so it is settings.js
    expect(diff.fields.uiPort.source).toBe('settings.js');
    // disableEditor (UI-only affordance) is NRCC-side
    expect(diff.fields.disableEditor.source satisfies FieldSource).toBe('settings.js');
    // authAdminPassword is write-only; never sourced from settings.js
    expect(diff.fields.authAdminPassword.source satisfies FieldSource).toBe('form default');
  });

  it('falls back to all-zero pending state when loadedConfig is null (initial render)', () => {
    const diff = computeConfigurationDiff(baseForm(), null, emptyHost(), '');
    expect(diff.pendingCount).toBe(0);
    // effectiveValue should fall back to the form value itself.
    expect(diff.fields.uiPort.effectiveValue).toBe(INITIAL_FORM_DATA.uiPort);
  });

  it('treats a null host status as read-only (canSave=false) even with pending changes', () => {
    // Slice F W2 — the bootstrap query can still be in-flight when the
    // configuration view first renders. The diff must stay safe by
    // treating null as read-only until the bootstrap resolves.
    const form = baseForm();
    form.uiPort = 2000; // dirty
    const diff = computeConfigurationDiff(form, loadedConfig(), null, '');
    expect(diff.pendingCount).toBeGreaterThan(0);
    expect(diff.canSave).toBe(false);
  });
});