import { useLayoutEffect, useRef, useState } from 'react';

/**
 * Colour tokens shown on the styleguide. Only names, jobs and pairings live
 * here; every value is read from src/index.css at runtime. Jobs and names
 * come from DESIGN.md (Colors); `against` is the intended ground or ink and
 * `min` the AA threshold (4.5 text, 3 non-text, null when decorative).
 */
export interface ColourToken {
  token: string;
  name: string;
  job: string;
  against: string;
  min: 4.5 | 3 | null;
}

export const COLOUR_TOKENS: ColourToken[] = [
  { token: '--ds-bg-void', name: 'drawing-white / drawing-negative', job: 'Ground: the page.', against: '--ds-text-primary', min: 4.5 },
  { token: '--ds-bg-base', name: 'drawing-sunken', job: 'Ground: table headers, code blocks, the off toggle track.', against: '--ds-text-primary', min: 4.5 },
  { token: '--ds-bg-surface', name: 'sheet', job: 'Ground: panels, the change circuit column, button grounds.', against: '--ds-text-primary', min: 4.5 },
  { token: '--ds-bg-elevated', name: 'sheet-raised', job: 'Ground: inputs, dialogs, overlays.', against: '--ds-text-primary', min: 4.5 },
  { token: '--ds-text-primary', name: 'ink', job: 'Ink: primary text.', against: '--ds-bg-void', min: 4.5 },
  { token: '--ds-text-secondary', name: 'ink-secondary', job: 'Ink: secondary text, field labels, proof lines.', against: '--ds-bg-void', min: 4.5 },
  { token: '--ds-text-muted', name: 'ink-muted', job: 'Ink: muted text, placeholders, waiting contacts.', against: '--ds-bg-void', min: 4.5 },
  { token: '--color-primary', name: 'conductor-blue', job: 'Action: acts and selects. Primary button edge, selected tab, focus, checked toggle.', against: '--ds-bg-surface', min: 3 },
  { token: '--ds-on-tint-primary', name: 'conductor-blue-text', job: 'Action: label text on a plain or tinted surface.', against: '--ds-bg-surface', min: 4.5 },
  { token: '--ds-brand-hover', name: 'conductor-blue-hover', job: 'Action: solid button hover.', against: '--ds-bg-surface', min: 3 },
  { token: '--ds-brand-dim', name: 'conductor-blue-dim', job: 'Action: the quiet selection ground.', against: '--ds-text-primary', min: 4.5 },
  { token: '--color-info', name: 'pending-brown', job: 'Pending: a step energising or waiting for the operator.', against: '--ds-bg-surface', min: 3 },
  { token: '--ds-on-tint-info', name: 'pending-brown (text)', job: 'Pending: state words and chip labels.', against: '--ds-bg-surface', min: 4.5 },
  { token: '--color-success', name: 'verified-green', job: 'Verified: a passed gate, the "Editable" runtime status.', against: '--ds-bg-surface', min: 3 },
  { token: '--ds-on-tint-success', name: 'verified-green-text', job: 'Verified: state words and chip labels.', against: '--ds-bg-surface', min: 4.5 },
  { token: '--ds-verified-fill', name: 'verified-fill', job: 'Verified: the square beside a closed contact. A fill, never text; its ink stroke carries the contrast.', against: '--ds-bg-surface', min: null },
  { token: '--color-error', name: 'critical-red', job: 'Critical: failed contact, rollback branch, destructive edge, errors. Nothing else.', against: '--ds-bg-surface', min: 3 },
  { token: '--ds-on-tint-error', name: 'critical-red-text', job: 'Critical: error text and labels.', against: '--ds-bg-surface', min: 4.5 },
  { token: '--color-warning', name: 'warning-amber', job: 'Warning: a gate held by the operator (Open contact, read-only). Fill under an ink label.', against: '--color-warning-content', min: 4.5 },
  { token: '--ds-on-tint-warning', name: 'warning-amber-text', job: 'Warning: state words and the read-only notice.', against: '--ds-bg-surface', min: 4.5 },
  { token: '--ds-border-default', name: 'hairline', job: 'Line: decorative dividers and panel edges, never a control edge.', against: '--ds-bg-void', min: null },
  { token: '--ds-border-strong', name: 'control-border', job: 'Line: every control edge and the circuit conductor (WCAG 1.4.11).', against: '--ds-bg-base', min: 3 },
  { token: '--ds-rail-bg', name: 'rail', job: 'Rail: the ink navigation rail ground.', against: '--ds-rail-text', min: 4.5 },
];

export const GROUND_NAMES: Record<string, string> = {
  '--ds-text-primary': 'ink',
  '--ds-bg-void': 'ground',
  '--ds-bg-base': 'sunken',
  '--ds-bg-surface': 'sheet',
  '--color-warning-content': 'its label',
  '--ds-rail-text': 'rail text',
};

/** Every custom property the colour, shape and elevation tables read per theme. */
export const THEME_PROPS = [
  ...new Set([
    ...COLOUR_TOKENS.flatMap(({ token, against }) => [token, against]),
    '--ds-shadow-raised',
    '--radius-selector',
    '--radius-field',
    '--radius-box',
  ]),
];

/** Slug for a section heading, shared by the index links and the section ids. */
export const sectionId = (title: string) => title.toLowerCase().replace(/[^a-z]+/g, '-');

/**
 * Reads computed CSS properties (custom properties included) of the ref'd
 * element after layout. `props` must be a stable module-level array.
 */
export function useComputed<T extends Element>(props: readonly string[]) {
  const ref = useRef<T>(null);
  const [values, setValues] = useState<Record<string, string>>({});
  useLayoutEffect(() => {
    if (!ref.current) return;
    const style = getComputedStyle(ref.current);
    setValues(Object.fromEntries(props.map((p) => [p, style.getPropertyValue(p).trim()])));
  }, [props]);
  return [ref, values] as const;
}
