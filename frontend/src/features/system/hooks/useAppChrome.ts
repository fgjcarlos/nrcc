import { useQuery } from '@tanstack/react-query';
import { systemService } from '@/features/dashboard/services/systemService';
import { bootstrapService } from '@/features/bootstrap/services/bootstrapService';
import { queryKeys } from '@/shared/lib/queryKeys';

/**
 * useAppChrome — issue #766 slice C
 *
 * Returns the runtime context the persistent header needs:
 * - `nodeRedVersion` from `/api/system/info` (Node-RED detected version)
 * - `edgeMode` from `/api/system/info` (NRCC edge deployment flag)
 * - `configuration` from `/api/bootstrap/status` (ConfigurationCapabilities)
 *   carries `editable`, `mode`, `runtimeVersion`, and a `reason` that
 *   the compat chip surfaces as its accessible description.
 *
 * Refetch cadence matches the dashboard's own hooks (systemInfo 10s,
 * bootstrap 30s) so the header chip strip stays in sync without
 * adding new polling traffic. Each field is `undefined` until its
 * respective query resolves; chips render the neutral "loading"
 * palette during the brief window.
 */

export interface AppChromeData {
  nodeRedVersion: string | undefined;
  edgeMode: boolean | undefined;
  configurationEditable: boolean | undefined;
  configurationMode: 'editable' | 'read-only' | undefined;
  configurationReason: string | undefined;
  configurationRuntimeVersion: string | undefined;
}

export function useAppChrome(): AppChromeData {
  const systemQuery = useQuery({
    queryKey: queryKeys.system.info,
    queryFn: () => systemService.getInfo(),
    refetchInterval: 10_000,
  });

  const bootstrapQuery = useQuery({
    queryKey: queryKeys.bootstrap.status,
    queryFn: () => bootstrapService.getStatus(),
    refetchInterval: 30_000,
  });

  const systemInfo = systemQuery.data?.data?.data;
  const hostStatus = bootstrapQuery.data?.data?.data;

  return {
    nodeRedVersion: systemInfo?.nodeRedVersion,
    edgeMode: systemInfo?.edgeMode,
    configurationEditable: hostStatus?.configuration?.editable,
    configurationMode: hostStatus?.configuration?.mode,
    configurationReason: hostStatus?.configuration?.reason,
    configurationRuntimeVersion: hostStatus?.configuration?.runtimeVersion,
  };
}
