import { Fragment, type ReactNode } from 'react';
import { useEnvironment } from '@/features/bootstrap/hooks/useEnvironment';
import type { HostStatusWithCodes } from '@/features/bootstrap/types';
import { ErrorState, Skeleton } from '@/shared/components/feedback';
import { StatusChip } from '@/shared/components/ui';
import { useT } from '@/i18n';

const designator = 'font-mono text-xs font-semibold tabular-nums text-ds-text-primary';

/** A path in mono that wraps after a slash; mid-segment only if one segment cannot fit at all. */
function PathDesignator({ value }: { value: string }) {
  return (
    <code className={`${designator} wrap-anywhere`}>
      {value.split('/').map((part, index) => (
        <Fragment key={index}>
          {index > 0 && <>/<wbr /></>}
          {part}
        </Fragment>
      ))}
    </code>
  );
}

function Fact({ label, children, wide }: { label: string; children: ReactNode; wide?: boolean }) {
  return (
    <div className={wide ? 'min-w-0 space-y-1 sm:col-span-2' : 'min-w-0 space-y-1'}>
      <dt className="text-xs text-ds-text-secondary">{label}</dt>
      <dd>{children}</dd>
    </div>
  );
}

function Section({ id, title, children }: { id: string; title: string; children: ReactNode }) {
  return (
    <section aria-labelledby={id} className="space-y-3 py-4">
      <h2 id={id} className="text-sm font-semibold text-ds-text-primary">{title}</h2>
      {children}
    </section>
  );
}

function HostDetails({ host }: { host: HostStatusWithCodes }) {
  const { t, i18n } = useT();
  const env = host.nodeRed;
  const settings = host.settings;
  const recommendations = host.recommendations ?? [];
  const modeKey = { native: 'modeNative', docker: 'modeDocker', none: 'modeNone', unknown: 'modeUnknown' }[env.mode];
  const sourceKey = { env: 'sourceEnv', detected: 'sourceDetected', 'nrcc-data': 'sourceNrccData' }[settings.source];

  // Known codes come from the catalog; anything else keeps the backend text.
  const recommendationText = (message: string, index: number) => {
    const key = `bootstrap:recommendation.${host.recommendationCodes?.[index]}`;
    return host.recommendationCodes?.[index] && i18n.exists(key) ? t(key) : message;
  };

  return (
    <div className="divide-y divide-ds-border-subtle">
      <Section id="bootstrap-runtime-title" title={t('bootstrap:runtimeTitle')}>
        <dl className="grid gap-x-8 gap-y-3 sm:grid-cols-2">
          <Fact label={t('bootstrap:mode')}>
            <StatusChip size="sm" variant="neutral">{modeKey ? t(`bootstrap:${modeKey}`) : env.mode}</StatusChip>
          </Fact>
          {env.detected && (
            <Fact label={t('bootstrap:process')}>
              <StatusChip size="sm" variant={env.running ? 'success' : 'warning'}>
                {t(env.running ? 'bootstrap:running' : 'bootstrap:notRunning')}
              </StatusChip>
            </Fact>
          )}
          <Fact label={t('bootstrap:version')}>
            {env.version
              ? <span className={designator}>{env.version}</span>
              : <span className="text-sm text-ds-text-secondary">{t('bootstrap:notReported')}</span>}
          </Fact>
          <Fact label={t('bootstrap:platform')}>
            <span className={designator}>{host.platform}</span>
          </Fact>
          {env.containerName && (
            // Native installs report their systemd unit in containerName (internal/service/host.go).
            <Fact label={t(env.mode === 'docker' ? 'bootstrap:containerName' : 'bootstrap:serviceUnit')}>
              <span className={`${designator} wrap-anywhere`}>{env.containerName}</span>
            </Fact>
          )}
          {env.userDir && (
            <Fact label={t('bootstrap:userDirectory')} wide>
              <PathDesignator value={env.userDir} />
            </Fact>
          )}
          {env.settingsPath && (
            <Fact label={t('bootstrap:defaultSettingsPath')} wide>
              <PathDesignator value={env.settingsPath} />
              {env.settingsPath !== settings.path && (
                <p className="mt-1 max-w-[75ch] text-xs text-ds-text-secondary">{t('bootstrap:defaultSettingsHint')}</p>
              )}
            </Fact>
          )}
        </dl>
      </Section>

      <Section id="bootstrap-settings-title" title={t('bootstrap:editedTitle')}>
        <dl className="grid gap-x-8 gap-y-3 sm:grid-cols-2">
          <Fact label={t('bootstrap:path')} wide>
            <PathDesignator value={settings.path} />
          </Fact>
          <Fact label={t('bootstrap:source')}>
            {sourceKey
              ? <span className="text-sm text-ds-text-primary">{t(`bootstrap:${sourceKey}`)}</span>
              : <span className={designator}>{settings.source}</span>}
          </Fact>
          <Fact label={t('bootstrap:writeAccess')}>
            <StatusChip size="sm" variant={settings.writable ? 'success' : 'warning'}>
              {t(settings.writable ? 'bootstrap:writable' : 'bootstrap:notWritable')}
            </StatusChip>
          </Fact>
          <Fact label={t('bootstrap:latestBackup')} wide>
            {settings.backupPath
              ? <PathDesignator value={settings.backupPath} />
              : <span className="text-sm text-ds-text-secondary">{t('bootstrap:noBackup')}</span>}
          </Fact>
        </dl>
      </Section>

      {recommendations.length > 0 && (
        <Section id="bootstrap-recommendations-title" title={t('bootstrap:recommendations')}>
          <ul className="max-w-[75ch] list-disc space-y-2 pl-5 text-sm text-ds-text-primary">
            {recommendations.map((message, index) => (
              <li key={index}>{recommendationText(message, index)}</li>
            ))}
          </ul>
        </Section>
      )}
    </div>
  );
}

export function BootstrapView() {
  const { t } = useT();
  const { data: hostStatus, isLoading, error, refetch } = useEnvironment();

  return (
    <div className="space-y-4">
      <header className="flex flex-col gap-3 border-b border-ds-border-subtle pb-3 sm:flex-row sm:items-end sm:justify-between">
        <div className="min-w-0 space-y-1">
          <h1 className="text-xl font-bold text-ds-text-primary">{t('bootstrap:bootstrapTitle')}</h1>
          <p className="max-w-[75ch] text-pretty text-sm text-ds-text-secondary">{t('bootstrap:intro')}</p>
        </div>
        {hostStatus && (
          <StatusChip size="sm" variant={hostStatus.ready ? 'success' : 'warning'} className="shrink-0 self-start sm:self-auto">
            {t(hostStatus.ready ? 'bootstrap:ready' : 'bootstrap:notReady')}
          </StatusChip>
        )}
      </header>

      {isLoading && <Skeleton rows={6} className="h-8" />}

      {error && (
        <ErrorState
          title={t('bootstrap:errorTitle')}
          description={t('bootstrap:errorDescription')}
          onRetry={() => void refetch()}
        />
      )}

      {hostStatus && <HostDetails host={hostStatus} />}
    </div>
  );
}
