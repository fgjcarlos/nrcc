import { useState, type ReactNode } from 'react';
import { Check, X } from 'lucide-react';
import { useT } from '@/i18n';
import { Button } from '@/shared/components/ui/Button';
import { Table } from '@/shared/components/ui/Table';
import { NrccMark } from '@/shared/components/NrccMark';
import { cn } from '@/shared/lib/utils';
import { contrastRatio, meetsAA } from './lib/contrast';
import { COLOUR_TOKENS, GROUND_NAMES, THEME_PROPS, useComputed } from './lib/tokens';
import { Section, Specimen, Tag } from './parts';

const THEMES = [
  { id: 'corporateLight', label: 'Light' },
  { id: 'corporateDark', label: 'Dark' },
] as const;

type ThemeId = (typeof THEMES)[number]['id'];

/** One hidden probe per theme, so both themes are read side by side whatever the page theme is. */
function useThemeValues() {
  const [lightRef, light] = useComputed<HTMLDivElement>(THEME_PROPS);
  const [darkRef, dark] = useComputed<HTMLDivElement>(THEME_PROPS);
  const probes = (
    <>
      <div ref={lightRef} data-theme="corporateLight" hidden />
      <div ref={darkRef} data-theme="corporateDark" hidden />
    </>
  );
  const values: Record<ThemeId, Record<string, string>> = { corporateLight: light, corporateDark: dark };
  return { values, probes };
}

function Verdict({ ratio, min, against }: { ratio: number | null; min: 4.5 | 3 | null; against: string }) {
  if (ratio === null) return <p className="text-ds-text-muted">not measured</p>;
  const measured = `${ratio.toFixed(2)}:1 vs ${against}`;
  if (min === null) return <p className="text-ds-text-secondary">{measured} · decorative, no AA minimum</p>;
  const pass = meetsAA(ratio, min);
  const Icon = pass ? Check : X;
  return (
    <p className={cn('flex items-center gap-1', pass ? 'text-ds-on-tint-success' : 'text-ds-on-tint-error')}>
      <Icon className="h-3 w-3 shrink-0" aria-hidden="true" />
      {`AA ${pass ? 'pass' : 'fail'} · ${measured} (min ${min})`}
    </p>
  );
}

export function ColourSection() {
  const { values, probes } = useThemeValues();
  return (
    <Section
      title="Colour"
      intro="Conductor colours on a warm drawing sheet: every hue has one job, and the job holds in both themes. Values, hex and contrast are read from src/index.css at runtime; each theme column is drawn on its own ground."
    >
      {probes}
      <Table caption="Colour tokens in both themes" stacked>
        <thead>
          <tr>
            <th scope="col">Token</th>
            <th scope="col">Job</th>
            {THEMES.map((theme) => (
              <th key={theme.id} scope="col">
                {theme.label}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {COLOUR_TOKENS.map((c) => (
            <tr key={c.token}>
              <td data-label="Token">
                <div>
                  <code className="font-mono text-xs font-semibold text-ds-text-primary">{c.token}</code>
                  <p className="text-xs text-ds-text-secondary">{c.name}</p>
                </div>
              </td>
              <td data-label="Job" className="max-w-xs text-ds-text-secondary">
                {c.job}
              </td>
              {THEMES.map((theme) => {
                const v = values[theme.id];
                const value = v[c.token] ?? '';
                return (
                  <td key={theme.id} data-label={theme.label} data-theme={theme.id} className="bg-ds-bg-void text-ds-text-primary">
                    <div className="flex items-center gap-3">
                      <span
                        aria-hidden="true"
                        className="h-8 w-8 shrink-0 rounded-selector border border-ds-border-strong"
                        style={{ backgroundColor: value || undefined }}
                      />
                      <div className="min-w-0 text-xs">
                        <p className="font-mono font-semibold">{value || '—'}</p>
                        <Verdict
                          ratio={contrastRatio(value, v[c.against] ?? '')}
                          min={c.min}
                          against={GROUND_NAMES[c.against] ?? c.against}
                        />
                      </div>
                    </div>
                  </td>
                );
              })}
            </tr>
          ))}
        </tbody>
      </Table>
    </Section>
  );
}

const TYPE_PROPS = ['font-size', 'font-weight', 'line-height', 'font-family'];

/** Text rendered with the real utilities, labelled with what the browser computed. */
function MeasuredText({ className, children }: { className: string; children: ReactNode }) {
  const [ref, v] = useComputed<HTMLParagraphElement>(TYPE_PROPS);
  const family = (v['font-family'] ?? '').split(',')[0].replace(/["']/g, '');
  return (
    <div className="grid gap-1 sm:grid-cols-[minmax(0,1fr)_auto] sm:items-baseline sm:gap-6">
      <p ref={ref} className={cn('min-w-0 text-ds-text-primary', className)}>
        {children}
      </p>
      <Tag>{v['font-size'] ? `${v['font-size']} · ${v['font-weight']} · ${v['line-height']} · ${family}` : 'not measured'}</Tag>
    </div>
  );
}

export function TypeSection() {
  const { t } = useT();
  const scale = [
    { name: 'Title', className: 'text-xl font-bold', use: 'The page title only, one per view.', sample: t('configuration:pageTitle') },
    { name: 'Body', className: 'text-sm', use: 'Form controls, tabs, descriptions, runtime status.', sample: t('configuration:topology.subtitle') },
    { name: 'Body strong', className: 'text-sm font-semibold', use: 'Section headings and contact names.', sample: t('configuration:circuit.title') },
    { name: 'Label', className: 'text-xs', use: 'Field labels, help text, contact state words, proof lines, totals.', sample: t('configuration:circuit.proof.localValid') },
    { name: 'Designator', className: 'font-mono text-xs font-semibold tabular-nums', use: 'REV, K1–K6, RB, settings.js keys, server error codes, version numbers.', sample: 'REV a1b2c3d4 · K1 · RB · uiPort · RUNTIME_READINESS_FAILED' },
  ];
  return (
    <Section
      title="Type"
      intro="Overpass Variable for everything; Overpass Mono for designators. Both self-hosted. Three sizes only: 20, 14 and 12px."
    >
      <Specimen name="Scale" caption="Each line uses the real utilities; the label on the right is what the browser computed.">
        <div className="space-y-4">
          {scale.map((s) => (
            <div key={s.name} className="space-y-1">
              <p className="text-xs text-ds-text-secondary">
                <span className="font-semibold text-ds-text-primary">{s.name}</span> · {s.use}
              </p>
              <MeasuredText className={s.className}>{s.sample}</MeasuredText>
            </div>
          ))}
        </div>
      </Specimen>
      <Specimen name="Weights" caption="Overpass at 14px. Rank with weight before reaching for size.">
        <div className="space-y-2">
          {['font-normal', 'font-medium', 'font-semibold', 'font-bold'].map((weight) => (
            <MeasuredText key={weight} className={cn('text-sm', weight)}>
              {t('configuration:circuit.proof.backupTaken')}
            </MeasuredText>
          ))}
        </div>
      </Specimen>
      <Specimen
        name="Ranking rule"
        caption="Rank by weight, case and rule; no opacity. Never add a fourth size, and never fade text with an opacity mix: use the three ink tokens. pnpm run check:faded-text rejects opacity mixes below 65%."
      >
        <div className="space-y-1 text-sm">
          <p className="text-ds-text-primary">ink · {t('configuration:circuit.proof.serverAccepted')}</p>
          <p className="text-ds-text-secondary">ink-secondary · {t('configuration:circuit.proof.serverAccepted')}</p>
          <p className="text-ds-text-muted">ink-muted · {t('configuration:circuit.proof.serverAccepted')}</p>
        </div>
      </Specimen>
    </Section>
  );
}

const WIDTH = ['width'];
const RADIUS = ['border-top-left-radius'];

function SpaceBar({ className, use }: { className: string; use: string }) {
  const [ref, v] = useComputed<HTMLSpanElement>(WIDTH);
  return (
    <li className="grid grid-cols-[3rem_1.5rem_minmax(0,1fr)] items-center gap-3 text-sm">
      <Tag>{v.width || '—'}</Tag>
      <span ref={ref} aria-hidden="true" className={cn('h-4 bg-ds-text-secondary', className)} />
      <span className="text-ds-text-secondary">{use}</span>
    </li>
  );
}

function RadiusBox({ className, name, use }: { className: string; name: string; use: string }) {
  const [ref, v] = useComputed<HTMLDivElement>(RADIUS);
  // rounded-full computes to calc(infinity * 1px), a huge number of pixels.
  const radius = parseFloat(v['border-top-left-radius'] ?? '') > 9999 ? 'infinite' : v['border-top-left-radius'];
  return (
    <div className="flex items-center gap-3">
      <div ref={ref} aria-hidden="true" className={cn('h-10 w-10 shrink-0 border border-ds-border-strong bg-ds-bg-elevated', className)} />
      <div className="text-xs">
        <p className="font-semibold text-ds-text-primary">
          {name} <Tag>{radius || '—'}</Tag>
        </p>
        <p className="text-ds-text-secondary">{use}</p>
      </div>
    </div>
  );
}

export function SpacingSection() {
  const { values, probes } = useThemeValues();
  return (
    <Section title="Spacing, radii and elevation" intro="A 4px base, near-square drafting corners, and a flat sheet: depth is a ground change or a 1px hairline.">
      {probes}
      <Specimen name="Spacing" caption="Tight inside a group, wider between groups. Widths are measured from the rendered bars.">
        <ul className="space-y-2">
          <SpaceBar className="w-1" use="Inside label rows." />
          <SpaceBar className="w-2" use="Between a control and its neighbours; inside the circuit foot." />
          <SpaceBar className="w-3" use="Between contacts and between header items." />
          <SpaceBar className="w-4" use="Between page blocks." />
          <SpaceBar className="w-6" use="Between columns." />
        </ul>
      </Specimen>
      <Specimen name="Radii" caption="Selectors take 2px, fields, buttons, panels and banners 4px. Only the toggle track and knob are fully round.">
        <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
          <RadiusBox className="rounded-selector" name="selector" use="Checkbox, radio, status chip." />
          <RadiusBox className="rounded-field" name="field" use="Inputs and buttons." />
          <RadiusBox className="rounded-box" name="box" use="Panels, banners, dialogs." />
          <RadiusBox className="rounded-full" name="full" use="Toggle track and knob only." />
        </div>
      </Specimen>
      <Specimen
        name="Elevation"
        caption="The Hairline Rule: elevation is a 1px border plus at most the raised shadow. Need more separation? Change the ground or draw a rule, never a bigger shadow."
      >
        <div className="grid gap-4 sm:grid-cols-2">
          <div className="rounded-box border border-ds-border-default bg-ds-bg-surface p-4 text-sm">
            <p className="font-semibold text-ds-text-primary">Hairline</p>
            <p className="text-xs text-ds-text-secondary">1px hairline on the sheet: panels and dividers.</p>
          </div>
          <div className="rounded-box border border-ds-border-default bg-ds-bg-surface p-4 text-sm shadow-glow">
            <p className="font-semibold text-ds-text-primary">Raised</p>
            <p className="text-xs text-ds-text-secondary">Hairline plus --ds-shadow-raised:</p>
            {THEMES.map((theme) => (
              <p key={theme.id} className="text-xs text-ds-text-secondary">
                {theme.label} <Tag>{values[theme.id]['--ds-shadow-raised'] || '—'}</Tag>
              </p>
            ))}
          </div>
        </div>
      </Specimen>
    </Section>
  );
}

const BLADE = ['transition-duration', 'transition-timing-function'];

export function MotionSection() {
  const [open, setOpen] = useState(true);
  const [ref, v] = useComputed<SVGLineElement>(BLADE);
  const reduce = typeof window.matchMedia === 'function' && window.matchMedia('(prefers-reduced-motion: reduce)').matches;
  return (
    <Section
      title="Motion"
      intro="One authored moment: a contact closing or opening on the change circuit. Motion says a state changed; nothing else moves."
    >
      <Specimen
        name="Contact close"
        caption="The blade rotates over 150ms with an exponential ease-out; in the circuit each contact waits 60ms per index so contacts close in sequence. This blade uses the real .circuit-blade rule."
      >
        <div className="flex flex-wrap items-center gap-6">
          <svg viewBox="0 0 20 20" width="48" height="48" aria-hidden="true" className="text-ds-text-primary">
            <g stroke="currentColor" strokeWidth="1.5" strokeLinecap="square">
              <line x1="10.5" y1="0" x2="10.5" y2="5" />
              <line x1="10.5" y1="15" x2="10.5" y2="20" />
              <line ref={ref} x1="10.5" y1="15" x2="10.5" y2="5" className="circuit-blade" data-open={open ? 'true' : 'false'} />
            </g>
          </svg>
          <Button type="button" variant="secondary" onClick={() => setOpen((o) => !o)}>
            {open ? 'Close contact' : 'Open contact'}
          </Button>
          <div className="text-xs text-ds-text-secondary">
            <p>
              duration <Tag>{v['transition-duration'] || '—'}</Tag>
            </p>
            <p>
              easing <Tag>{v['transition-timing-function'] || '—'}</Tag>
            </p>
          </div>
        </div>
      </Specimen>
      <Specimen
        name="Reduced motion"
        caption="Under prefers-reduced-motion the blade switches instantly, and one global rule caps every transition and animation (skeleton pulse, spinners) at 0.001ms. State still changes; only the movement goes."
      >
        <p className="text-sm text-ds-text-primary">
          {reduce ? 'This browser asks for reduced motion: the durations above read 0.001ms.' : 'This browser does not ask for reduced motion.'}
        </p>
      </Specimen>
    </Section>
  );
}

export function VoiceSection() {
  const { t } = useT();
  const pairs = [
    { good: t('configuration:circuit.proof.rejected'), bad: 'Oops! Something went wrong.', why: 'Name what happened and what it left behind.' },
    { good: t('configuration:circuit.review', { count: 3 }), bad: 'Submit', why: 'A control names its action and its object.' },
    { good: t('configuration:circuit.proof.backupTaken'), bad: 'Backup BK-1042 created', why: 'Never invent a designator the API does not return.' },
    { good: t('configuration:circuit.rollback.restored'), bad: 'Saved successfully!', why: 'Say the real outcome, on the circuit, not in a toast.' },
    { good: t('env-vars:deleteDialogDescription', { key: 'MQTT_BROKER_URL' }), bad: 'Are you sure?', why: 'State the consequence before the operator confirms.' },
  ];
  return (
    <Section title="Voice" intro="From PRODUCT.md and DESIGN.md. Do strings are the live catalogue, so they switch with the language.">
      <Specimen name="Principles" caption="The operator knows Node-RED; write for that person.">
        <ul className="list-disc space-y-1 pl-5 text-sm text-ds-text-primary">
          <li>Show before doing: every change is previewed, explained and reversible before it touches the instance.</li>
          <li>Precise, technical, honest language; no hand-holding and no hiding of state.</li>
          <li>Say every state in words as well as colour and shape.</li>
          <li>Controls name their action; errors name the problem and the recovery.</li>
          <li>No toasts on the gated flow: the change circuit and its live region carry the outcome.</li>
          <li>Identify things with stable designators (REV, K1–K6, RB, settings.js keys), never with decoration.</li>
        </ul>
      </Specimen>
      <Specimen name="Do and don't" caption="Real NRCC strings against the generic copy they replace." className="p-0">
        <Table caption="Voice do and don't" stacked className="border-0">
          <thead>
            <tr>
              <th scope="col">Do</th>
              <th scope="col">Don&apos;t</th>
              <th scope="col">Why</th>
            </tr>
          </thead>
          <tbody>
            {pairs.map((p) => (
              <tr key={p.bad}>
                <td data-label="Do" className="text-ds-text-primary">{p.good}</td>
                <td data-label="Don't" className="text-ds-text-secondary line-through">{p.bad}</td>
                <td data-label="Why" className="text-ds-text-secondary">{p.why}</td>
              </tr>
            ))}
          </tbody>
        </Table>
      </Specimen>
    </Section>
  );
}

const TONES = ['accent', 'primary', 'muted'] as const;
const SIZES = [
  { size: 'sm', glyph: 16 },
  { size: 'md', glyph: 24 },
  { size: 'lg', glyph: 32 },
] as const;
const FAVICON = `${import.meta.env.BASE_URL}favicon.svg`;

export function LogoSection() {
  return (
    <Section
      title="Logo"
      intro="The closed IEC normally-open contact: two terminal rings with the blade lying on the line, drawn on a 16-unit grid. Wordmark: NRCC in Overpass."
    >
      <Specimen name="NrccMark tones and sizes" caption="accent in the header, primary in the sidebar, muted inside dense rows. Glyph sizes are 1×, 1.5× and 2× the grid; 128px is the presentation size.">
        <div className="space-y-4">
          {TONES.map((tone) => (
            <div key={tone} className="flex flex-wrap items-center gap-4">
              <span className="w-16"><Tag>{tone}</Tag></span>
              {SIZES.map(({ size, glyph }) => (
                <div key={size} className="flex items-center gap-2">
                  <NrccMark tone={tone} size={size} />
                  <Tag>{`${size} · ${glyph}px`}</Tag>
                </div>
              ))}
            </div>
          ))}
          <div className="flex flex-wrap items-center gap-4">
            <span className="w-16"><Tag>128px</Tag></span>
            <NrccMark tone="primary" size="lg" className="h-40 w-40 [&_svg]:h-32 [&_svg]:w-32" />
            <span className="text-xl font-bold text-ds-text-primary">NRCC</span>
          </div>
        </div>
      </Specimen>
      <Specimen name="Favicon tile" caption="Ink on a drawing-white tile, so it reads on light and dark browser tabs. Same glyph as docs/assets/nrcc-mark.svg.">
        <div className="flex flex-wrap items-end gap-6">
          {[16, 32, 128].map((px) => (
            <div key={px} className="space-y-1">
              <img src={FAVICON} width={px} height={px} alt={px === 16 ? 'NRCC favicon' : ''} />
              <Tag>{`${px}px`}</Tag>
            </div>
          ))}
        </div>
      </Specimen>
      <Specimen
        name="Clear space and minimum size (proposed)"
        caption="Not yet in DESIGN.md. Proposed from the grid: keep one terminal-ring diameter (4 units, a quarter of the glyph) clear on every side, and never draw the glyph below 16px (1×), where the rings fill in."
      >
        <div className="flex flex-wrap items-center gap-8">
          <div className="border border-ds-border-strong p-8">
            <img src={FAVICON} width={128} height={128} alt="" />
          </div>
          <div className="flex items-center gap-2">
            <img src={FAVICON} width={16} height={16} alt="" />
            <Tag>16px minimum</Tag>
          </div>
        </div>
      </Specimen>
    </Section>
  );
}
