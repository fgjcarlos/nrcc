import { useState } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import { dashboardService } from '../services';

import { queryKeys } from '@/shared/lib/queryKeys';
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

  const [pendingConfirm, setPendingConfirm] = useState(false);
  const [isRestarting, setIsRestarting] = useState(false);
  const [isStartStopping, setIsStartStopping] = useState(false);

  const invalidateRuntimeStatus = () => {
    queryClient.invalidateQueries({ queryKey: queryKeys.runtime.status });
  };

  const getErrorMessage = (error: unknown) =>
    error instanceof Error ? error.message : 'Error desconocido';

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
      successTitle: 'Node-RED reiniciado',
      successMessage: 'El proceso ha arrancado correctamente.',
      errorTitle: 'No se pudo reiniciar Node-RED',
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
      successTitle: 'Node-RED iniciado',
      successMessage: 'El proceso ha arrancado correctamente.',
      errorTitle: 'No se pudo iniciar Node-RED',
      onSuccess: invalidateRuntimeStatus,
      onFinally: () => setIsStartStopping(false),
    });
  };

  const handleStopNodeRed = async () => {
    setIsStartStopping(true);

    await runRuntimeAction({
      action: dashboardService.stopNodeRed,
      successTitle: 'Node-RED detenido',
      successMessage: 'El proceso se ha detenido correctamente.',
      errorTitle: 'No se pudo detener Node-RED',
      onSuccess: invalidateRuntimeStatus,
      onFinally: () => setIsStartStopping(false),
    });
  };

  const handleOpenNodeRed = () => {
    window.open(`http://localhost:${uiPort || 1880}`, '_blank');
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
