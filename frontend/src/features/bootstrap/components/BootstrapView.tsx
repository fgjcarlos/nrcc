import { useEnvironment } from '@/features/bootstrap/hooks/useEnvironment';
import {
  AlertCircle,
  CheckCircle2,
  AlertTriangle,
  Loader,
  FileText,
  Settings,
  Container,
} from 'lucide-react';
import { cn } from '@/shared/lib';
import { useT } from '@/i18n';
import type { NodeRedEnvironment, SettingsDocument } from '@/shared/types';

const NodeRedEnvironmentCard = ({ env }: { env: NodeRedEnvironment }) => {
  const { t } = useT();
  const modeLabel = {
    native: t('bootstrap:modeNative'),
    docker: t('bootstrap:modeDocker'),
    none: t('bootstrap:modeNone'),
    unknown: t('bootstrap:modeUnknown'),
  }[env.mode] || env.mode;

  const modeColor = {
    native: 'text-info',
    docker: 'text-primary',
    none: 'text-warning',
    unknown: 'text-error',
  }[env.mode] || 'text-base-content';

  return (
    <div className="surface-card border border-border rounded-lg p-6">
      <div className="flex items-center gap-3 mb-4">
        <Container className="w-5 h-5 text-body-secondary" />
        <h3 className="font-semibold text-base-content">{t('bootstrap:nodeRedEnvironment')}</h3>
      </div>
      
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div>
          <p className="text-xs uppercase tracking-wider text-base-content/50 mb-1">{t('bootstrap:mode')}</p>
          <p className={cn('text-lg font-semibold', modeColor)}>{modeLabel}</p>
        </div>
        
        <div>
          <p className="text-xs uppercase tracking-wider text-base-content/50 mb-1">{t('bootstrap:status')}</p>
          <div className="flex items-center gap-2">
            {env.running ? (
              <>
                <CheckCircle2 className="w-5 h-5 text-success" />
                <span className="text-lg font-semibold text-success">{t('bootstrap:running')}</span>
              </>
            ) : (
              <>
                <AlertCircle className="w-5 h-5 text-warning" />
                <span className="text-lg font-semibold text-warning">{t('bootstrap:notRunning')}</span>
              </>
            )}
          </div>
        </div>

        {env.version && (
          <div>
            <p className="text-xs uppercase tracking-wider text-base-content/50 mb-1">{t('bootstrap:version')}</p>
            <p className="text-sm text-base-content">{env.version}</p>
          </div>
        )}

        {env.settingsPath && (
          <div>
            <p className="text-xs uppercase tracking-wider text-base-content/50 mb-1">{t('bootstrap:settingsPath')}</p>
            <p className="text-xs text-base-content/70 truncate font-mono" title={env.settingsPath}>
              {env.settingsPath}
            </p>
          </div>
        )}

        {env.userDir && (
          <div>
            <p className="text-xs uppercase tracking-wider text-base-content/50 mb-1">{t('bootstrap:userDirectory')}</p>
            <p className="text-xs text-base-content/70 truncate font-mono" title={env.userDir}>
              {env.userDir}
            </p>
          </div>
        )}

        {env.containerName && (
          <div>
            <p className="text-xs uppercase tracking-wider text-base-content/50 mb-1">{t('bootstrap:containerName')}</p>
            <p className="text-sm text-base-content font-mono">{env.containerName}</p>
          </div>
        )}
      </div>
    </div>
  );
};

const SettingsCard = ({ settings }: { settings: SettingsDocument }) => {
  const { t } = useT();
  return (
  <div className="surface-card border border-border rounded-lg p-6">
    <div className="flex items-center gap-3 mb-4">
      <FileText className="w-5 h-5 text-body-secondary" />
      <h3 className="font-semibold text-base-content">{t('bootstrap:settingsFile')}</h3>
    </div>

    <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
      <div>
        <p className="text-xs uppercase tracking-wider text-base-content/50 mb-1">{t('bootstrap:path')}</p>
        <p className="text-xs text-base-content/70 truncate font-mono" title={settings.path}>
          {settings.path}
        </p>
      </div>

      <div>
        <p className="text-xs uppercase tracking-wider text-base-content/50 mb-1">{t('bootstrap:source')}</p>
        <p className="text-sm text-base-content">{settings.source}</p>
      </div>

      <div>
        <p className="text-xs uppercase tracking-wider text-base-content/50 mb-1">{t('bootstrap:writable')}</p>
        <div className="flex items-center gap-2">
          {settings.writable ? (
            <>
              <CheckCircle2 className="w-4 h-4 text-success" />
              <span className="text-sm text-success font-medium">{t('common:yes')}</span>
            </>
          ) : (
            <>
              <AlertCircle className="w-4 h-4 text-error" />
              <span className="text-sm text-error font-medium">{t('common:no')}</span>
            </>
          )}
        </div>
      </div>

      {settings.backupPath && (
        <div>
          <p className="text-xs uppercase tracking-wider text-base-content/50 mb-1">{t('bootstrap:backupPath')}</p>
          <p className="text-xs text-base-content/70 truncate font-mono" title={settings.backupPath}>
            {settings.backupPath}
          </p>
        </div>
      )}
    </div>
  </div>
  );
};

const RecommendationsCard = ({ recommendations }: { recommendations: string[] }) => {
  const { t } = useT();
  return (
  <div className="surface-card border border-border rounded-lg p-6">
    <div className="flex items-center gap-3 mb-4">
      <AlertTriangle className="w-5 h-5 text-warning" />
      <h3 className="font-semibold text-base-content">{t('bootstrap:recommendations')}</h3>
    </div>

    <div className="space-y-2">
      {recommendations.map((rec, idx) => (
        <div key={idx} className="flex gap-3">
          <div className="flex-shrink-0 pt-0.5">
            <AlertTriangle className="w-4 h-4 text-warning" />
          </div>
          <p className="text-sm text-base-content/70">{rec}</p>
        </div>
      ))}
    </div>
  </div>
  );
};

export function BootstrapView() {
  const { t } = useT();
  const { data: hostStatus, isLoading, error } = useEnvironment();

  return (
    <div className="space-y-6 p-6">
      {/* Header */}
      <div className="flex items-end justify-between gap-4">
        <div>
          <p className="text-xs uppercase tracking-[0.28em] text-base-content/50">{t('bootstrap:systemLabel')}</p>
          <h1 className="text-3xl font-bold tracking-tight text-base-content">{t('bootstrap:bootstrapTitle')}</h1>
        </div>
      </div>

      {isLoading && (
        <div className="flex items-center justify-center py-12">
          <Loader className="w-6 h-6 animate-spin text-primary" />
        </div>
      )}

      {error && (
        <div className="alert alert-error">
          <AlertTriangle className="w-5 h-5" />
          <div>
            <h3 className="font-semibold">{t('bootstrap:errorLoadingStatus')}</h3>
            <p className="text-sm">{error instanceof Error ? error.message : t('common:unknownError')}</p>
          </div>
        </div>
      )}

      {hostStatus && (
        <>
          {/* Overall Status */}
          <div className="surface-card border border-border rounded-lg p-6">
            <div className="flex items-center gap-3 mb-4">
              <Settings className="w-5 h-5 text-body-secondary" />
              <h2 className="text-xl font-semibold text-base-content">{t('bootstrap:systemStatus')}</h2>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div>
                <p className="text-xs uppercase tracking-wider text-base-content/50 mb-1">{t('bootstrap:platform')}</p>
                <p className="text-lg font-semibold text-base-content">{hostStatus.platform}</p>
              </div>

              <div>
                <p className="text-xs uppercase tracking-wider text-base-content/50 mb-1">{t('bootstrap:overallStatus')}</p>
                <div className="flex items-center gap-2">
                  {hostStatus.ready ? (
                    <>
                      <CheckCircle2 className="w-5 h-5 text-success" />
                      <span className="text-lg font-semibold text-success">{t('bootstrap:ready')}</span>
                    </>
                  ) : (
                    <>
                      <AlertCircle className="w-5 h-5 text-error" />
                      <span className="text-lg font-semibold text-error">{t('bootstrap:notReady')}</span>
                    </>
                  )}
                </div>
              </div>
            </div>
          </div>

          {/* Node-RED Environment */}
          <NodeRedEnvironmentCard env={hostStatus.nodeRed} />

          {/* Settings */}
          <SettingsCard settings={hostStatus.settings} />

          {/* Recommendations */}
          {hostStatus.recommendations && hostStatus.recommendations.length > 0 && (
            <RecommendationsCard recommendations={hostStatus.recommendations} />
          )}
        </>
      )}
    </div>
  );
}
