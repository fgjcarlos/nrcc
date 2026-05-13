import type { HostStatus, RuntimeStatus } from '@/shared/types';

export function getStatusBadgeClass(status: RuntimeStatus | string) {
  switch (status) {
    case 'running':
      return 'badge-success';
    case 'stopped':
    case 'error':
      return 'badge-error';
    default:
      return 'badge-warning';
  }
}

export function getHostWarningMessage(host: HostStatus) {
  return [
    !host.nodejs.installed ? 'Node.js no está instalado.' : '',
    !host.nodeRed.detected ? 'Node-RED aún no fue detectado.' : '',
    !host.settings.writable ? 'nrcc no puede escribir sobre settings.js.' : '',
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

export function getDeploymentLabel(mode?: HostStatus['nodeRed']['mode']) {
  switch (mode) {
    case 'docker':
      return 'Docker';
    case 'native':
      return 'Nativo';
    default:
      return 'Sin detectar';
  }
}
