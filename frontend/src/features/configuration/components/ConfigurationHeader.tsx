import type { ReactNode } from 'react';
import { useT } from '@/i18n';

/**
 * ConfigurationHeader — issue #766 slice F (W3), T9 layout.
 *
 * Page title with its designator (the settings.js revision) and one
 * runtime status beside it, host details underneath. The topology
 * diagram lives on the Security page only; at header size its labels
 * were illegible and competed with the runtime status.
 */
export interface ConfigurationHeaderProps {
  /** Mono designator for the configuration source, e.g. "REV a1b2c3d4". */
  designator?: ReactNode;
  /** One runtime status shown next to the title. */
  status?: ReactNode;
  /** Host details rendered under the title. */
  children?: ReactNode;
}

export function ConfigurationHeader({ designator, status, children }: ConfigurationHeaderProps) {
  const { t } = useT();

  return (
    <header className="min-w-0 space-y-1 border-b border-ds-border-subtle pb-3">
      <div className="flex flex-wrap items-baseline gap-x-3 gap-y-1">
        <h1 id="configuration-view-title" className="text-xl font-bold text-ds-text-primary">
          {t('configuration:pageTitle')}
        </h1>
        {designator}
        {status}
      </div>
      {children}
    </header>
  );
}
