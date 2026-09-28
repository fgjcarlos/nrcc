import { Settings } from 'lucide-react';
import { useT } from '@/i18n';
import { NrccTopologyDiagram } from './NrccTopologyDiagram';

/**
 * ConfigurationHeader — issue #766 slice F (W3)
 *
 * Replaces the inline header inside `ConfigurationView`. Renders the
 * page title with the settings icon and embeds the
 * `NrccTopologyDiagram` so the operator sees the
 * `NRCC → settings.js → Node-RED` data flow above the form.
 *
 * The topology diagram is a sibling element of the title block so the
 * header stays readable at narrow viewports — the diagram collapses to
 * its icon-only representation below 480 px.
 */
export interface ConfigurationHeaderProps {
  /** Optional flag: when true, the topology diagram is hidden (e.g.
   *  on narrow viewports where it doesn't fit cleanly). */
  hideTopology?: boolean;
}

export function ConfigurationHeader({ hideTopology }: ConfigurationHeaderProps) {
  const { t } = useT();

  return (
    <header className="flex flex-col gap-6 border-b border-border pb-6">
      <div className="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <p className="text-xs uppercase tracking-[0.24em] text-base-content/50">
            {t('configuration:settings')}
          </p>
          <h1
            id="configuration-view-title"
            className="flex items-center gap-3 text-2xl font-bold text-base-content"
          >
            <Settings className="h-6 w-6" aria-hidden="true" />
            {t('configuration:pageTitle')}
          </h1>
        </div>
      </div>
      {!hideTopology && <NrccTopologyDiagram />}
    </header>
  );
}
