import { useMemo } from 'react';
import { useQuery } from '@tanstack/react-query';
import { configService, settingsService } from '../services';
import { bootstrapService } from '@/features/bootstrap/services';
import type { HostStatus, NodeRedConfigFormData } from '@/shared/types';
import { configToFormData, type NodeRedConfigResponse } from '../lib/configTransformers';

import { queryKeys } from '@/shared/lib/queryKeys';
export function useConfigurationData() {
  // Fetch current config
  const configQuery = useQuery({
    queryKey: queryKeys.config.root,
    queryFn: () => configService.getConfig(),
  });

  // Fetch bootstrap status
  const bootstrapQuery = useQuery({
    queryKey: queryKeys.bootstrap.status,
    queryFn: async () => {
      const response = await bootstrapService.getStatus();
      return response.data?.data as HostStatus;
    },
  });

  // Fetch raw settings
  const rawSettingsQuery = useQuery({
    queryKey: queryKeys.config.rawSettings,
    queryFn: () => settingsService.getRaw(),
  });

  // The raw query data is the only value we want to track for memo
  // invalidation. `configToFormData` returns a fresh object every call,
  // so without `useMemo` its reference changes on every render and the
  // `useEffect` in ConfigurationView that syncs form state would
  // re-fire on every interaction, clobbering the user's in-flight edits
  // (toggles not flipping, fields reverting). The same applies to the
  // raw settings content. See issue #366.
  const loadedConfig = configQuery.data?.data?.data;
  const initialFormData = useMemo<NodeRedConfigFormData | null>(
    () =>
      loadedConfig
        ? configToFormData(loadedConfig as NodeRedConfigResponse)
        : null,
    [loadedConfig],
  );

  const settingsDoc = rawSettingsQuery.data?.data?.data;
  // settingsDoc is an object reference from react-query, which is stable
  // per fetch. `content` is a primitive string, also stable.
  const rawSettingsContent = useMemo(
    () => settingsDoc?.content || '',
    [settingsDoc],
  );

  // Helper: derive host info
  const hostStatus = bootstrapQuery.data;

  return {
    // Config query state
    configLoading: configQuery.isLoading,
    configError: configQuery.isError,
    initialFormData,

    // Bootstrap query state
    bootstrapLoading: bootstrapQuery.isLoading,
    hostStatus,

    // Settings query state
    settingsLoading: rawSettingsQuery.isLoading,
    rawSettingsContent,
    settingsDoc,

    // Refetch functions
    refetchConfig: configQuery.refetch,
    refetchSettings: rawSettingsQuery.refetch,
  };
}
