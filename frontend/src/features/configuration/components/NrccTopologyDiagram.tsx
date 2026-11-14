import { useT } from '@/i18n';

/**
 * NrccTopologyDiagram — issue #766 slice F (W3)
 *
 * Inline SVG diagram that visualises the data flow NRCC edits:
 *
 *   NRCC  ──►  settings.js  ──►  Node-RED + Dashboard
 *
 * The diagram is theme-aware (it inherits the active DaisyUI theme via
 * Tailwind's stroke / fill utility classes, which the design-system
 * tokens map in `frontend/src/index.css`). No image asset; no new
 * illustration system.
 *
 * Width: fluid, scales to container. Height: fixed at 96 px on desktop,
 * collapses the labels to icons on < 480 px viewports.
 *
 * @example
 *   <NrccTopologyDiagram />
 */
export interface NrccTopologyDiagramProps {
  /** Optional accessible label override. Defaults to the i18n topology.title. */
  ariaLabel?: string;
  /** Optional extra className for layout (margins, max-width, etc.). */
  className?: string;
}

export function NrccTopologyDiagram({
  ariaLabel,
  className,
}: NrccTopologyDiagramProps) {
  const { t } = useT();
  const label = ariaLabel ?? t('configuration:topology.title');

  return (
    <figure
      role="img"
      aria-label={label}
      className={['flex flex-col gap-2', className].filter(Boolean).join(' ')}
      data-testid="nrcc-topology-diagram"
    >
      <figcaption className="text-xs uppercase tracking-[0.24em] text-base-content/50">
        {t('configuration:topology.title')}
      </figcaption>
      <svg
        viewBox="0 0 480 96"
        className="h-24 w-full max-w-xl"
        role="presentation"
        aria-hidden="true"
      >
        <defs>
          <marker
            id="nrcc-topology-arrow"
            viewBox="0 0 10 10"
            refX="9"
            refY="5"
            markerWidth="6"
            markerHeight="6"
            orient="auto-start-reverse"
          >
            <path d="M 0 0 L 10 5 L 0 10 z" className="fill-base-content/60" />
          </marker>
        </defs>

        {/* NRCC node */}
        <g transform="translate(8,16)">
          <rect
            x="0"
            y="0"
            width="120"
            height="64"
            rx="12"
            className="fill-ds-bg-elevated stroke-ds-border-default"
            strokeWidth="1"
          />
          <text
            x="60"
            y="32"
            textAnchor="middle"
            className="fill-ds-text-primary"
            fontSize="14"
            fontWeight="600"
          >
            NRCC
          </text>
          <text
            x="60"
            y="50"
            textAnchor="middle"
            className="fill-ds-text-muted hidden md:block"
            fontSize="10"
          >
            {t('configuration:topology.nodeNrccSubtitle')}
          </text>
        </g>

        {/* Arrow: NRCC → settings.js */}
        <g>
          <line
            x1="132"
            y1="48"
            x2="178"
            y2="48"
            className="stroke-base-content/60"
            strokeWidth="1.5"
            markerEnd="url(#nrcc-topology-arrow)"
          />
          <text
            x="155"
            y="40"
            textAnchor="middle"
            className="fill-ds-text-muted hidden md:block"
            fontSize="9"
          >
            {t('configuration:topology.arrowNrccSettings')}
          </text>
        </g>

        {/* settings.js node */}
        <g transform="translate(184,16)">
          <rect
            x="0"
            y="0"
            width="120"
            height="64"
            rx="12"
            className="fill-ds-bg-surface stroke-ds-border-default"
            strokeWidth="1"
          />
          <text
            x="60"
            y="32"
            textAnchor="middle"
            className="fill-ds-text-primary"
            fontSize="13"
            fontWeight="600"
          >
            settings.js
          </text>
          <text
            x="60"
            y="50"
            textAnchor="middle"
            className="fill-ds-text-muted hidden md:block"
            fontSize="9"
          >
            {t('configuration:topology.nodeSettingsSubtitle')}
          </text>
        </g>

        {/* Arrow: settings.js → Node-RED */}
        <g>
          <line
            x1="308"
            y1="48"
            x2="354"
            y2="48"
            className="stroke-base-content/60"
            strokeWidth="1.5"
            markerEnd="url(#nrcc-topology-arrow)"
          />
          <text
            x="331"
            y="40"
            textAnchor="middle"
            className="fill-ds-text-muted hidden md:block"
            fontSize="9"
          >
            {t('configuration:topology.arrowSettingsNodeRed')}
          </text>
        </g>

        {/* Node-RED + Dashboard node */}
        <g transform="translate(360,16)">
          <rect
            x="0"
            y="0"
            width="112"
            height="64"
            rx="12"
            className="fill-ds-bg-elevated stroke-ds-border-default"
            strokeWidth="1"
          />
          <text
            x="56"
            y="32"
            textAnchor="middle"
            className="fill-ds-text-primary"
            fontSize="13"
            fontWeight="600"
          >
            Node-RED
          </text>
          <text
            x="56"
            y="50"
            textAnchor="middle"
            className="fill-ds-text-muted hidden md:block"
            fontSize="9"
          >
            + Dashboard
          </text>
        </g>
      </svg>
      <p className="text-xs text-base-content/60">
        {t('configuration:topology.subtitle')}
      </p>
    </figure>
  );
}
