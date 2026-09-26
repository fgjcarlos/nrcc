import { useQueryClient } from '@tanstack/react-query';
import { queryKeys } from '@/shared/lib/queryKeys';
import type { SystemInfo, HostStatus } from '@/shared/types';

/**
 * useAppChrome — issue #766 slice C
 *
 * Returns the runtime context the persistent header needs:
 * - `nodeRedVersion` from /api/system/info (Node-RED detected version)
 * - `edgeMode` from /api/system/info (NRCC edge deployment flag)
 * - `configuration` from /api/bootstrap/status (ConfigurationCapabilities)
 *
 * Implementation note (re: PR #839 follow-up): reading via
 * `queryClient.getQueryData` instead of `useQuery` subscriptions avoids
 * introducing a third parallel subscription for keys that are already
 * managed by useDashboardData (Overview) and useConfigurationData
 * (Settings + Security). Three useQuery calls sharing the same keys
 * raced with Playwright's page.route override (one of the three would
 * fire before the route registered and pre-populate the cache with the
 * default mock, freezing editable=false on Settings/Security). Reading
 * from the cache means whichever hook first populates the key wins —
 * and Header stays in sync via queryClient's existing subscription
 * notifications without adding a third subscription.
 *
 * Fields are `undefined` until the dashboard or settings page populates
 * the shared cache. Chips render the neutral loading palette during
 * that window, which is fine for an authenticated operator navigation.
 */
export interface AppChromeData {
  nodeRedVersion: string | undefined;
  edgeMode: boolean | undefined;
  configurationEditable: boolean | undefined;
  configurationMode: 'editable' | 'read-only' | undefined;
  configurationReason: string | undefined;
  configurationRuntimeVersion: string | undefined;
}

type SystemInfoCached = { data?: { data?: SystemInfo } };
type HostStatusCached = { data?: { data?: HostStatus } };

export function useAppChrome(): AppChromeData {
  const queryClient = useQueryClient();
  const systemCached = queryClient.getQueryData(queryKeys.system.info) as SystemInfoCached | undefined;
  const hostCached = queryClient.getQueryData(queryKeys.bootstrap.status) as HostStatusCached | undefined;

  const systemInfo = systemCached?.data?.data;
  const hostStatus = hostCached?.data?.data;

  return {
    nodeRedVersion: systemInfo?.nodeRedVersion,
    edgeMode: systemInfo?.edgeMode,
    configurationEditable: hostStatus?.configuration?.editable,
    configurationMode: hostStatus?.configuration?.mode,
    configurationReason: hostStatus?.configuration?.reason,
    configurationRuntimeVersion: hostStatus?.configuration?.runtimeVersion,
  };
}
