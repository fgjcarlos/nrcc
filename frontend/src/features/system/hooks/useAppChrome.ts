import { useQuery } from '@tanstack/react-query';
import { useEnvironment } from '@/features/bootstrap/hooks/useEnvironment';
import { dashboardService } from '@/features/dashboard/services';
import { queryKeys } from '@/shared/lib/queryKeys';

/**
 * useAppChrome: the runtime context the persistent header shows.
 *
 * The runtime status comes from the same /api/bootstrap/status query
 * (queryKeys.bootstrap.status) that Configuration renders as
 * "Node-RED <version> · Editable", with the same derivation, so the
 * header can never contradict the page. It subscribes to the query: the
 * former cache peek never re-rendered, and read the cache one `.data`
 * level too deep, so it stayed "unknown / Mode pending" forever.
 */
export type RuntimeStatus =
  | { state: 'loading' }
  | { state: 'unknown' }
  | { state: 'known'; version: string | undefined; editable: boolean; reason: string | undefined };

export interface AppChromeData {
  runtime: RuntimeStatus;
  /** NRCC edge deployment flag from /api/system/info; undefined until known. */
  edgeMode: boolean | undefined;
}

export function useAppChrome(): AppChromeData {
  const host = useEnvironment();
  const system = useQuery({
    queryKey: queryKeys.system.info,
    queryFn: () => dashboardService.getSystemInfo(),
  });

  const status = host.data;
  let runtime: RuntimeStatus;
  if (status) {
    runtime = {
      state: 'known',
      version: status.configuration?.runtimeVersion || status.nodeRed?.version || undefined,
      // Same rule as ConfigurationView: only an explicit `false` is read-only.
      editable: status.configuration?.editable !== false,
      reason: status.configuration?.reason,
    };
  } else {
    runtime = { state: host.isPending ? 'loading' : 'unknown' };
  }

  return { runtime, edgeMode: system.data?.data?.data?.edgeMode };
}
