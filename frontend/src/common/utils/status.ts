import type { GlobalStatus } from '../types'
import type { RuntimeStatus } from '../../api'

export function buildGlobalStatus(runtime: RuntimeStatus | undefined, runtimeError: unknown, systemError: unknown): GlobalStatus {
  if (runtimeError || systemError) {
    return {
      title: 'Degraded',
      detail: 'Some dashboard checks failed. Review the active page notices for details.',
      tone: 'warn',
    }
  }

  if (!runtime) {
    return {
      title: 'Unknown',
      detail: 'Waiting for runtime data from the local control center.',
      tone: 'neutral',
    }
  }

  if (runtime.running && runtime.healthy) {
    return {
      title: 'Operational',
      detail: 'Node-RED is running and responding to health checks.',
      tone: 'ok',
    }
  }

  if (runtime.running) {
    return {
      title: 'Needs attention',
      detail: 'Node-RED is running but health checks are not passing yet.',
      tone: 'warn',
    }
  }

  return {
    title: 'Stopped',
    detail: 'Node-RED is not running. Restart the runtime from the dashboard when ready.',
    tone: 'warn',
  }
}

export function getStatusBadgeClass(status: string) {
  switch (status) {
    case 'pass':
      return 'status-pass'
    case 'warn':
      return 'status-warn'
    case 'fail':
      return 'status-fail'
    default:
      return 'status-unknown'
  }
}
