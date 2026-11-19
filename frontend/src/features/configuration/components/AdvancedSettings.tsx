import { useState } from 'react';
import { useConfigurationActions } from '../hooks';
import { useQuery } from '@tanstack/react-query';
import { toast } from 'sonner';
import { Loader2, Info } from 'lucide-react';
import api from '@/shared/lib';
import { Button, Dialog, DialogBody, DialogFooter, Input } from '@/shared/components/ui';
import { useT } from '@/i18n';

// Slice 3 of issue #764 — the curated-preset UI surface.
//
// The component lists the presets returned by GET /api/presets, asks for
// any operator values a preset declares (absolute file paths), shows the
// redacted diff of the in-memory apply, and delegates the confirmed write
// to /api/settings/raw so the settings.js write path stays single-flight
// and audited (slice 2 of #758). It never writes to disk directly.

// PresetView mirrors the JSON shape returned by the backend handler.
interface PresetView {
  id: string;
  name: string;
  description: string;
  surfaces: string[];
  trust: string;
  channelBoundary: string;
  managedKeys: string[];
  inputs?: string[];
}

interface PresetPreview {
  after: string;
  preview: string;
  replaced: string[] | null;
  inserted: string[] | null;
}

interface AdvancedSettingsProps {
  // The live settings.js source the operator opened with. Used to
  // scope the preview request and as the "back to source" target.
  rawContent: string;
  // Issue #878 — the revision of the document the operator opened. The
  // preset write carries it so a stale preview cannot overwrite newer
  // settings.js.
  expectedRevision?: string;
  // Called after the operator confirms an apply and the parent
  // has written the new source through /api/settings/raw.
  onApplied: () => void;
}

type ApiError = { response?: { status?: number; data?: { error?: { code?: string } } } } | null;

function isRouteMissing(error: unknown): boolean {
  return (error as ApiError)?.response?.status === 404;
}

/** Maps a preview failure to a catalog key; the HTTP status is never shown. */
function previewErrorKey(error: unknown): string {
  const response = (error as ApiError)?.response;
  switch (response?.data?.error?.code) {
    case 'PRESET_NEEDS_INPUT':
      return 'configuration:preset.error.needsInput';
    case 'SOURCE_NOT_EXPORTS':
      return 'configuration:preset.error.sourceNotExports';
    case 'PRESET_NOT_REGISTERED':
      return 'configuration:preset.error.notRegistered';
  }
  if (response?.status === 413) return 'configuration:preset.error.tooLarge';
  return 'configuration:preset.error.generic';
}

export function AdvancedSettings({ rawContent, expectedRevision, onApplied }: AdvancedSettingsProps) {
  const { t } = useT();
  const actions = useConfigurationActions();

  const [activePreset, setActivePreset] = useState<PresetView | null>(null);
  const [values, setValues] = useState<Record<string, string>>({});
  const [preview, setPreview] = useState<PresetPreview | null>(null);
  const [previewError, setPreviewError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [acknowledged, setAcknowledged] = useState(false);

  const presetsQuery = useQuery({
    queryKey: ['presets'],
    queryFn: async () => {
      const r = await api.get<{ success: boolean; data: PresetView[] }>('/presets');
      return r.data.data;
    },
    // A 404 is definitive: this server does not mount the presets routes,
    // so retrying only keeps the spinner up. Other failures keep the
    // default retries.
    retry: (failureCount, error) => !isRouteMissing(error) && failureCount < 3,
  });

  async function generatePreview(preset: PresetView, presetValues: Record<string, string>) {
    setLoading(true);
    setPreview(null);
    setPreviewError(null);
    setAcknowledged(false);
    try {
      const inputs = preset.inputs ?? [];
      const r = await api.post<{ success: boolean; data: PresetPreview }>(
        `/presets/${encodeURIComponent(preset.id)}/preview`,
        inputs.length > 0
          ? { content: rawContent, values: Object.fromEntries(inputs.map((name) => [name, presetValues[name]?.trim() ?? ''])) }
          : { content: rawContent },
      );
      if (!r.data.success) {
        setPreviewError(t('configuration:preset.error.generic'));
        return;
      }
      setPreview(r.data.data);
    } catch (err) {
      setPreviewError(t(previewErrorKey(err)));
    } finally {
      setLoading(false);
    }
  }

  function openPreset(preset: PresetView) {
    setActivePreset(preset);
    setValues({});
    setPreview(null);
    setPreviewError(null);
    setAcknowledged(false);
    if ((preset.inputs ?? []).length === 0) void generatePreview(preset, {});
  }

  function close() {
    if (actions.saveRawSettingsMutation.isPending) return;
    setActivePreset(null);
    setPreview(null);
    setPreviewError(null);
  }

  function apply() {
    if (!activePreset || !preview) return;
    const name = activePreset.name;
    actions.saveRawSettingsMutation.mutate(
      { content: preview.after, expectedRevision },
      {
        onSuccess: () => {
          toast.success(t('configuration:preset.applied', { name }));
          onApplied();
          setActivePreset(null);
          setPreview(null);
        },
        onError: () => {
          setPreviewError(t('configuration:preset.error.applyFailed'));
        },
      },
    );
  }

  if (presetsQuery.isLoading) {
    return (
      <div className="flex items-center gap-2 text-sm text-ds-text-secondary">
        <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" /> {t('configuration:preset.loadingCurated')}
      </div>
    );
  }

  // Presets are optional: an unavailable catalogue is reported calmly, in
  // neutral ink, not as a warning.
  if (presetsQuery.isError || !presetsQuery.data) {
    return (
      <div
        role="status"
        className="flex items-center gap-2 rounded-box border border-ds-border-subtle px-3 py-2 text-sm text-ds-text-secondary"
      >
        <Info className="h-4 w-4 shrink-0" aria-hidden="true" />
        {isRouteMissing(presetsQuery.error) ? t('configuration:preset.unavailable') : t('configuration:preset.loadError')}
      </div>
    );
  }

  const presets = presetsQuery.data;
  const inputs = activePreset?.inputs ?? [];
  const inputsComplete = inputs.every((name) => (values[name] ?? '').trim() !== '');
  const unchanged = preview !== null && preview.preview.trim() === '';
  const saving = actions.saveRawSettingsMutation.isPending;
  const canApply = preview !== null && !unchanged && acknowledged && !saving;

  return (
    <section aria-label={t('configuration:preset.sectionLabel')} className="space-y-3 border-t border-ds-border-subtle pt-4">
      <h2 className="text-sm font-semibold text-ds-text-primary">{t('configuration:preset.title')}</h2>
      <p className="text-sm text-ds-text-secondary">{t('configuration:preset.description')}</p>

      <ul className="divide-y divide-ds-border-subtle border-y border-ds-border-subtle">
        {presets.map((preset) => (
          <li
            key={preset.id}
            role="region"
            aria-label={preset.id}
            className="flex flex-col gap-2 py-3 sm:flex-row sm:items-start sm:justify-between"
          >
            <div className="min-w-0 space-y-1">
              <h3 className="text-sm font-semibold text-ds-text-primary">{preset.name}</h3>
              <p className="text-sm text-ds-text-secondary">{preset.description}</p>
              <p className="flex flex-wrap gap-x-3 gap-y-1 text-xs text-ds-text-muted">
                <span>
                  {t('configuration:preset.touches')}{' '}
                  <span className="font-mono text-ds-text-secondary">{preset.managedKeys.join(', ')}</span>
                </span>
                <span>
                  {t('configuration:preset.surfaces')}{' '}
                  <span className="font-mono text-ds-text-secondary">{preset.surfaces.join(', ')}</span>
                </span>
              </p>
            </div>
            <Button
              type="button"
              variant="secondary"
              size="sm"
              className="shrink-0 self-start"
              onClick={() => openPreset(preset)}
              aria-label={t('configuration:preset.previewAria', { name: preset.name })}
            >
              {t('configuration:preset.preview')}
            </Button>
          </li>
        ))}
      </ul>

      <Dialog
        open={activePreset !== null}
        onClose={close}
        dismissible={!saving}
        size="lg"
        title={activePreset ? t('configuration:preset.previewTitle', { name: activePreset.name }) : ''}
        description={t('configuration:preset.reviewDiff')}
      >
        <DialogBody className="space-y-4">
          {inputs.length > 0 && (
            <form
              className="space-y-3"
              onSubmit={(e) => {
                e.preventDefault();
                if (activePreset && inputsComplete && !loading) void generatePreview(activePreset, values);
              }}
            >
              <p className="text-sm text-ds-text-secondary">{t('configuration:preset.inputsHint')}</p>
              {inputs.map((name) => (
                <Input
                  key={name}
                  id={`preset-input-${name}`}
                  label={t(`configuration:preset.input.${name}`, { defaultValue: name })}
                  value={values[name] ?? ''}
                  onChange={(e) => setValues((current) => ({ ...current, [name]: e.target.value }))}
                  placeholder={t(`configuration:preset.inputPlaceholder.${name}`, { defaultValue: '' })}
                  spellCheck={false}
                  autoComplete="off"
                  className="font-mono"
                />
              ))}
              <Button type="submit" variant="secondary" disabled={!inputsComplete || loading}>
                {t('configuration:preset.generate')}
              </Button>
            </form>
          )}

          {loading && (
            <p className="flex items-center gap-2 text-sm text-ds-text-secondary" role="status">
              <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" />
              {t('configuration:preset.generatingPreview')}
            </p>
          )}

          {previewError && (
            <p role="alert" className="rounded-box border border-ds-danger/40 px-3 py-2 text-sm text-ds-on-tint-error">
              {previewError}
            </p>
          )}

          {preview && unchanged && (
            <p role="status" className="text-sm text-ds-text-secondary">
              {t('configuration:preset.unchanged')}
            </p>
          )}

          {preview && !unchanged && (
            <pre
              role="region"
              aria-label={t('configuration:preset.diffLabel')}
              className="max-h-80 overflow-auto rounded-box border border-ds-border-subtle bg-ds-bg-base p-3 font-mono text-xs"
            >
              {preview.preview}
            </pre>
          )}

          {preview && (
            <label className="flex cursor-pointer select-none items-start gap-3">
              <input
                type="checkbox"
                checked={acknowledged}
                onChange={(e) => setAcknowledged(e.target.checked)}
                disabled={saving}
                className="checkbox checkbox-sm mt-0.5"
              />
              <span className="text-sm text-ds-text-secondary">{t('configuration:preset.acknowledgement')}</span>
            </label>
          )}
        </DialogBody>
        <DialogFooter>
          <Button type="button" variant="ghost" onClick={close} disabled={saving}>
            {t('common:cancel')}
          </Button>
          <Button type="button" variant="solid" onClick={apply} disabled={!canApply} aria-busy={saving}>
            {t('configuration:preset.confirmApply')}
          </Button>
        </DialogFooter>
      </Dialog>
    </section>
  );
}
