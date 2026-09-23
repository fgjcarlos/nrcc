import { useState } from 'react';
import { useConfigurationActions } from '../hooks';
import { useQuery } from '@tanstack/react-query';
import { toast } from 'sonner';
import { Loader2, AlertTriangle, Lock, ShieldCheck } from 'lucide-react';
import api from '@/shared/lib';
import { ConfirmationDialog } from '@/shared/components/ConfirmationDialog';
import { useT } from '@/i18n';

// Slice 3 of issue #764 — the curated-preset UI surface.
//
// The component lists the presets returned by GET /api/presets, lets
// the operator preview the redacted diff of a preset apply, and
// surfaces a "back to source" escape hatch that reverts to the live
// settings.js the operator opened with. The actual disk write is
// delegated to the existing /api/settings/raw endpoint so the
// settings.js write path stays single-flight and audited (slice 2 of
// #758); this component never writes to disk directly.

// PresetView mirrors the JSON shape returned by the backend handler.
interface PresetView {
  id: string;
  name: string;
  description: string;
  surfaces: string[];
  trust: string;
  channelBoundary: string;
  managedKeys: string[];
}

interface PresetPreview {
  after: string;
  preview: string;
  replaced: string[];
  inserted: string[];
}

interface AdvancedSettingsProps {
  // The live settings.js source the operator opened with. Used to
  // scope the preview request and as the "back to source" target.
  rawContent: string;
  // Called after the operator confirms an apply and the parent
  // has written the new source through /api/settings/raw.
  onApplied: () => void;
}

export function AdvancedSettings({ rawContent, onApplied }: AdvancedSettingsProps) {
  const [activePreset, setActivePreset] = useState<PresetView | null>(null);
  const [preview, setPreview] = useState<PresetPreview | null>(null);
  const [previewError, setPreviewError] = useState<string | null>(null);

  const { t } = useT();


  const presetsQuery = useQuery({
    queryKey: ['presets'],
    queryFn: async () => {
      const r = await api.get<{ success: boolean; data: PresetView[] }>('/presets');
      return r.data.data;
    },
  });

  async function openPreview(preset: PresetView) {
    setActivePreset(preset);
    setPreview(null);
    setPreviewError(null);
    try {
      const r = await api.post<{ success: boolean; data: PresetPreview }>(
        `/presets/${encodeURIComponent(preset.id)}/preview`,
        { content: rawContent },
      );
      if (!r.data.success) {
        setPreviewError('Preset preview failed.');
        return;
      }
      setPreview(r.data.data);
    } catch (err) {
      setPreviewError((err as Error).message || 'Preset preview failed.');
    }
  }

  function cancelPreview() {
    setActivePreset(null);
    setPreview(null);
    setPreviewError(null);
  }

  const actions = useConfigurationActions();

  function confirmPreview() {
    if (!activePreset || !preview) return;
    // Route the confirmed write through the existing /api/settings/raw
    // path so it goes through the slice-2 ApplyCoordinator (single-
    // flight, audit-logged, source-revision-checked). The preview was
    // generated against the same rawContent the parent mounted with,
    // so preview.after is a valid write target.
    actions.saveRawSettingsMutation.mutate(preview.after, {
      onSuccess: () => {
        toast.success(`Applied ${activePreset.name}.`);
        onApplied();
        cancelPreview();
      },
      onError: (err) => {
        toast.error(`Failed to apply preset: ${(err as Error).message}`);
      },
    });
  }

  if (presetsQuery.isLoading) {
    return (
      <div className="flex items-center gap-2 text-sm text-base-content/65">
        <Loader2 className="h-4 w-4 animate-spin" /> Loading curated presets…
      </div>
    );
  }

  if (presetsQuery.isError || !presetsQuery.data) {
    return (
      <div
        role="status"
        className="flex items-center gap-2 rounded-xl border border-warning/40 bg-warning/10 px-3 py-2 text-sm text-warning"
      >
        <AlertTriangle className="h-4 w-4" /> Presets could not be loaded.
      </div>
    );
  }

  const presets = presetsQuery.data;

  return (
    <section aria-label="Curated presets" className="surface-card space-y-4 p-6">
      <header className="flex items-center gap-2">
        <ShieldCheck className="h-5 w-5 text-primary" />
        <h2 className="text-lg font-semibold">Curated presets</h2>
      </header>
      <p className="text-sm text-base-content/65">
        Vetted high-value recipes for advanced Node-RED 5 settings. Each preset
        declares the editor / HTTP / Socket.IO surfaces it touches and never
        rewrites operator-owned code outside its declared scope.
      </p>

      <ul className="grid grid-cols-1 gap-4 md:grid-cols-2">
        {presets.map((preset) => (
          <li
            key={preset.id}
            role="region"
            aria-label={preset.id}
            className="rounded-2xl border border-border bg-base-100/60 p-4"
          >
            <div className="flex items-center justify-between">
              <h3 className="text-base font-semibold">{preset.name}</h3>
              <span
                className="rounded-full bg-base-200/60 px-2 py-0.5 text-xs text-base-content/65"
                title={`Channel boundary: ${preset.channelBoundary}`}
              >
                <Lock className="mr-1 inline h-3 w-3" />
                {preset.channelBoundary}
              </span>
            </div>
            <p className="mt-1 text-sm text-base-content/65">{preset.description}</p>
            <div className="mt-2 flex flex-wrap gap-2 text-xs">
              {preset.surfaces.map((surface) => (
                <span
                  key={surface}
                  className="rounded-full border border-border bg-base-200/40 px-2 py-0.5 text-base-content/65"
                >
                  {surface}
                </span>
              ))}
              <span className="rounded-full bg-primary/10 px-2 py-0.5 text-primary">
                trust: {preset.trust}
              </span>
            </div>
            <div className="mt-3 flex justify-end">
              <button
                type="button"
                onClick={() => openPreview(preset)}
                className="action-btn-secondary"
                aria-label={`preview ${preset.name}`}
              >
                Preview
              </button>
            </div>
          </li>
        ))}
      </ul>

      <ConfirmationDialog
        isOpen={activePreset !== null}
        title={activePreset ? `Preview: ${activePreset.name}` : ''}
        description={
          previewError
            ? previewError
            : preview
              ? 'Review the redacted diff. Confirm to write the patched source through /api/settings/raw (or cancel to discard).'
              : 'Generating redacted preview…'
        }
        onConfirm={preview ? confirmPreview : () => undefined}
        onCancel={cancelPreview}
        confirmText={t('configuration:preset.confirmApply')}
        acknowledgement={t('configuration:preset.acknowledgement')}
      >
        {preview && (
          <pre
            role="region"
            aria-label="Redacted transaction preview"
            className="max-h-80 overflow-auto rounded-xl border border-border bg-base-200/40 p-3 font-mono text-xs"
          >
            {preview.preview}
          </pre>
        )}
      </ConfirmationDialog>
    </section>
  );
}
