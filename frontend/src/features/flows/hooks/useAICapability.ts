import { useQuery } from '@tanstack/react-query';
import { flowService, getAICapabilityUnavailableMessage } from '@/features/flows/services/flowService';

export const aiCapabilityQueryKey = ['ai-capability'] as const;

export function useAICapability() {
  const query = useQuery({
    queryKey: aiCapabilityQueryKey,
    queryFn: flowService.getAICapability,
    retry: false,
  });

  const isReady = query.data?.status === 'ready';
  const message = query.isLoading
    ? 'Checking AI provider availability…'
    : getAICapabilityUnavailableMessage(query.data);

  return {
    ...query,
    isReady,
    message,
  };
}
