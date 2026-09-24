import { useState } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import { dashboardService } from '../services';

import { queryKeys } from '@/shared/lib/queryKeys';
import { useT } from '@/i18n';

interface UseDashboardActionsOptions {
  uiPort?: number;
}

interface RuntimeActionOptions {
  action: () => Promise<unknown>;
  successTitle: string;
  successMessage: string;
  errorTitle: string;
  onSuccess?: () => void;
  onError?: () => void;
  onFinally?: () => void;
}

export function useDashboardActions({ uiPort }: UseDashboardActionsOptions) {
  const queryClient = useQueryClient();
  const { t } = useT();

  const [pendingConfirm, setPendingConfirm] = useState(false);
  const [isRestarting, setIsRestarting] = useState(false);
  const [isStartStopping, setIsStartStopping] = useState(false);

  const invalidateRuntimeStatus = () => {
    queryClient.invalidateQueries({ queryKey: queryKeys.runtime.status });
  };

  const getErrorMessage = (error: unknown) =>
    error instanceof Error ? error.message : t('dashboard:actions.unknownError');

  const pushRuntimeSuccessToast = (title: string, message: string) => {
    toast.success(title, { description: message });
  };

  const pushRuntimeErrorToast = (title: string, error: unknown) => {
    toast.error(title, {
      description: getErrorMessage(error),
      duration: 8000,
    });
  };

  const runRuntimeAction = async ({
    action,
    successTitle,
    successMessage,
    errorTitle,
    onSuccess,
    onError,
    onFinally,
  }: RuntimeActionOptions) => {
    try {
      await action();
      onSuccess?.();
      pushRuntimeSuccessToast(successTitle, successMessage);
      return true;
    } catch (error) {
      pushRuntimeErrorToast(errorTitle, error);
      onError?.();
      return false;
    } finally {
      onFinally?.();
    }
  };

  const handleRestartConfirm = async () => {
    setPendingConfirm(false);
    setIsRestarting(true);

    const restarted = await runRuntimeAction({
      action: dashboardService.restartNodeRed,
      successTitle: t('dashboard:actions.restartSuccessTitle'),
      successMessage: t('dashboard:actions.restartSuccessMessage'),
      errorTitle: t('dashboard:actions.restartErrorTitle'),
      onSuccess: invalidateRuntimeStatus,
      onError: () => setIsRestarting(false),
    });

    if (!restarted) {
      return;
    }

    setTimeout(() => {
      setIsRestarting(false);
      invalidateRuntimeStatus();
    }, 6000);
  };

  const handleStartNodeRed = async () => {
    setIsStartStopping(true);

    await runRuntimeAction({
      action: dashboardService.startNodeRed,
      successTitle: t('dashboard:actions.startSuccessTitle'),
      successMessage: t('dashboard:actions.startSuccessMessage'),
      errorTitle: t('dashboard:actions.startErrorTitle'),
      onSuccess: invalidateRuntimeStatus,
      onFinally: () => setIsStartStopping(false),
    });
  };

  const handleStopNodeRed = async () => {
    setIsStartStopping(true);

    await runRuntimeAction({
      action: dashboardService.stopNodeRed,
      successTitle: t('dashboard:actions.stopSuccessTitle'),
      successMessage: t('dashboard:actions.stopSuccessMessage'),
      errorTitle: t('dashboard:actions.stopErrorTitle'),
      onSuccess: invalidateRuntimeStatus,
      onFinally: () => setIsStartStopping(false),
    });
  };

  const handleOpenNodeRed = () => {
    // Validate uiPort is a positive integer in the valid TCP port range
    // before constructing the loopback URL. Defends against misconfigured
    // backend responses or env injection. Mirrors the same guard used by
    // the Open Node-RED Editor command in CommandPalette so the two entry
    // points cannot drift apart.
    const parsed = Number.parseInt(String(uiPort), 10);
    const safePort = Number.isInteger(parsed) && parsed >= 1 && parsed <= 65535 ? parsed : 1880;
    window.open(`http://localhost:${safePort}`, '_blank', 'noopener,noreferrer');
  };

  return {
    pendingConfirm,
    isRestarting,
    isStartStopping,
    setPendingConfirm,
    handleRestartConfirm,
    handleStartNodeRed,
    handleStopNodeRed,
    handleOpenNodeRed,
  };
}
