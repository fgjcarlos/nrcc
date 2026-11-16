import { useId, type AriaAttributes } from 'react';

/**
 * Layout of a shared text control. Edge, radius, ground and every state
 * (hover, focus, aria-invalid, disabled, read-only) come from the
 * unlayered text-control rule in src/index.css, so raw fields and these
 * primitives draw the same boundary.
 */
export const fieldControlClass = 'block w-full px-3 py-2 text-sm';

interface FieldOptions {
  id?: string;
  label?: string;
  /** `true` marks the control invalid; a string is also its error message. */
  error?: boolean | string;
  'aria-describedby'?: string;
  'aria-invalid'?: AriaAttributes['aria-invalid'];
}

export type Field = ReturnType<typeof useField>;

/** Ids and ARIA wiring shared by Input, Select and Textarea. */
export function useField({ id, label, error, 'aria-describedby': describedBy, 'aria-invalid': ariaInvalid }: FieldOptions) {
  const autoId = useId();
  const controlId = id || (label ? label.toLowerCase().replace(/\s+/g, '-') : autoId);
  const message = typeof error === 'string' && error ? error : undefined;
  const errorId = message ? `${controlId}-error` : undefined;
  return {
    controlId,
    message,
    errorId,
    controlProps: {
      id: controlId,
      'aria-invalid': error ? true : ariaInvalid,
      'aria-describedby': [describedBy, errorId].filter(Boolean).join(' ') || undefined,
    },
  };
}
