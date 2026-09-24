import type { HostStatus } from '@/shared/types';

// All display strings here resolve through the dashboard catalog so the
// runtime falls back to EN with a deterministic ES translation. The
// functions accept a t() handle so unit tests and component call sites
// both resolve against the active i18n instance.
export type DashboardT = (key: string) => string;

export function getHostWarningMessage(host: HostStatus, t: DashboardT) {
  return [
    !host.nodejs.installed ? t('dashboard:hostWarning.nodejsMissing') : '',
    !host.nodeRed.detected ? t('dashboard:hostWarning.nodeRedNotDetected') : '',
    !host.settings.writable ? t('dashboard:hostWarning.settingsNotWritable') : '',
  ]
    .filter(Boolean)
    .join(' ');
}

export function getSystemHealthIssues(host: HostStatus) {
  return [
    !host.nodejs.installed ? 'Node.js not installed' : null,
    !host.npm.installed ? 'npm not installed' : null,
    !host.nodeRedBinary.installed ? 'Node-RED binary not found' : null,
    !host.docker.installed ? 'Docker not installed' : null,
    !host.nodeRed.detected ? 'Node-RED environment not detected' : null,
    !host.settings.writable ? 'Settings file not writable' : null,
  ].filter((issue): issue is string => Boolean(issue));
}

export function getDeploymentLabel(mode: HostStatus['nodeRed']['mode'] | undefined, t: DashboardT) {
  switch (mode) {
    case 'docker':
      return t('dashboard:deploymentDocker');
    case 'native':
      return t('dashboard:deploymentNative');
    default:
      return t('dashboard:deploymentUnknown');
  }
}
