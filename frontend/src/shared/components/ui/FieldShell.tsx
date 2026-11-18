import { type ReactNode } from 'react';
import { type Field } from './useField';

/** Label, control and error message, in the FormFields label style (12px secondary). */
export function FieldShell({ field, label, children }: { field: Field; label?: string; children: ReactNode }) {
  return (
    <div className="space-y-1">
      {label && (
        <label htmlFor={field.controlId} className="block text-xs text-ds-text-secondary">
          {label}
        </label>
      )}
      {children}
      {field.message && (
        <p id={field.errorId} className="text-xs text-error">
          {field.message}
        </p>
      )}
    </div>
  );
}
