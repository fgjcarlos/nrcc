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

  // Helper: derive loaded config from query result
  const loadedConfig = configQuery.data?.data?.data;
  const initialFormData: NodeRedConfigFormData | null = loadedConfig
    ? configToFormData(loadedConfig as NodeRedConfigResponse)
    : null;

  // Helper: derive raw settings content
  const rawSettingsContent = rawSettingsQuery.data?.data?.data?.content || '';
  const settingsDoc = rawSettingsQuery.data?.data?.data;

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
