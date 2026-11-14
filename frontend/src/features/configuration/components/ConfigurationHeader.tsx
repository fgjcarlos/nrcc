import type { ReactNode } from 'react';
import { useT } from '@/i18n';
import { NrccTopologyDiagram } from './NrccTopologyDiagram';

/**
 * ConfigurationHeader — issue #766 slice F (W3), T9 layout.
 *
 * Page title with one runtime status beside it, optional host details
 * underneath, and the compact `NrccTopologyDiagram` on the right so the
 * operator sees the `NRCC → settings.js → Node-RED` flow once per page.
 */
export interface ConfigurationHeaderProps {
  /** Optional flag: when true, the topology diagram is hidden. */
  hideTopology?: boolean;
  /** One runtime status shown next to the title. */
  status?: ReactNode;
  /** Host details rendered under the title. */
  children?: ReactNode;
}

export function ConfigurationHeader({ hideTopology, status, children }: ConfigurationHeaderProps) {
  const { t } = useT();

  return (
    <header className="flex flex-col gap-3 border-b border-ds-border-subtle pb-3 md:flex-row md:items-end md:justify-between">
      <div className="min-w-0 space-y-1">
        <div className="flex flex-wrap items-baseline gap-x-3 gap-y-1">
          <h1 id="configuration-view-title" className="text-2xl font-bold text-base-content">
            {t('configuration:pageTitle')}
          </h1>
          {status}
        </div>
        {children}
      </div>
      {!hideTopology && <NrccTopologyDiagram compact className="shrink-0 md:w-72" />}
    </header>
  );
}
