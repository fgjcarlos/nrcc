/**
 * useConfigurationSave — issue #766 slice F W2.
 *
 * Save gate. Wraps the existing `useConfigurationActions` mutation and
 * blocks the network call when the per-field diff says the form is
 * not saveable (no pending changes, validation errors, or backend in
 * read-only mode).
 *
 * Replaces the aggregate `validateAuthFields` from
 * `useConfigurationActions` — the per-field validator in
 * `validateConfigurationField` is now the source of truth.
 *
 * The hook returns a structured `SaveOutcome` so the caller can show
 * a targeted error (no pending changes / validation errors / backend
 * failure) without the toast noise the old aggregate helper caused.
 */

import { useMemo } from 'react';
import type { NodeRedConfigFormData } from '@/shared/types';
import { formDataToConfigPayload } from '../lib/configTransformers';
import { useConfigurationActions } from './useConfigurationActions';
import type { ConfigurationDiffSummary } from './useConfigurationDiff';

export type SaveOutcome =
  | { ok: true }
  | { ok: false; reason: 'no-pending-changes' }
  | { ok: false; reason: 'validation-errors'; errors: Record<string, string> }
  | { ok: false; reason: 'read-only' }
  | { ok: false; reason: 'mutation-failed'; error: unknown };

interface UseConfigurationSaveInput {
  formData: NodeRedConfigFormData;
  diff: ConfigurationDiffSummary;
}

export function useConfigurationSave({ formData, diff }: UseConfigurationSaveInput) {
  const { saveConfigMutation } = useConfigurationActions();

  // Stabilise the gate so a re-render with the same inputs does not
  // produce a new function reference and re-trigger the Save button
  // effect chain.
  return useMemo(() => {
    return {
      async save(): Promise<SaveOutcome> {
        // canSave is the composite gate (pendingCount>0, no validation
        // errors, editable). When it is false, identify the most
        // specific reason so the caller can show a targeted message.
        if (!diff.canSave) {
          if (diff.pendingCount === 0) {
            return { ok: false, reason: 'no-pending-changes' };
          }
          const errorKeys = Object.keys(diff.validationErrors);
          if (errorKeys.length > 0) {
            return { ok: false, reason: 'validation-errors', errors: diff.validationErrors };
          }
          // canSave was false but neither pendingCount nor errors
          // explains it → backend is read-only.
          return { ok: false, reason: 'read-only' };
        }

        const payload = formDataToConfigPayload(formData);
        try {
          await saveConfigMutation.mutateAsync(payload);
          return { ok: true };
        } catch (error) {
          return { ok: false, reason: 'mutation-failed', error };
        }
      },
      isPending: saveConfigMutation.isPending,
    };
  }, [formData, diff, saveConfigMutation]);
}
