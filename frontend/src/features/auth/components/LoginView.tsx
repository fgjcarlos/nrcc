import { useNavigate, useLocation, Link } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { useAuth } from '../hooks/useAuth';
import { authService } from '../services/authService';
import { toast } from 'sonner';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { AlertCircle } from 'lucide-react';
import { type ReactNode } from 'react';

import { queryKeys } from '@/shared/lib/queryKeys';
import { useT } from '@/i18n';
import { Button, Input, buttonVariants } from '@/shared/components/ui';
import { NrccMark } from '@/shared/components/NrccMark';
// Messages are catalog keys; the view translates them at render.
const loginSchema = z.object({
  username: z.string().min(1, 'auth:login.errors.usernameRequired'),
  password: z.string().min(1, 'auth:login.errors.passwordRequired'),
});

type LoginFormData = z.infer<typeof loginSchema>;

/**
 * Public auth page: the mark with its wordmark and purpose over one
 * drawing sheet. Shared by Login and Setup; a calm single column.
 */
// children is optional so App's lazyNamed() accepts every export as a component.
export function AuthPage({ children }: { children?: ReactNode }) {
  const { t } = useT();
  return (
    <main className="auth-shell flex min-h-screen justify-center px-4 py-8 sm:items-center">
      <div className="w-full max-w-sm space-y-6">
        <div className="flex items-center gap-3">
          {/* The wordmark beside it already names the product. */}
          <span aria-hidden="true" className="flex">
            <NrccMark tone="primary" size="sm" className="text-ds-text-primary" />
          </span>
          <div className="min-w-0">
            <p className="text-sm font-bold tracking-wide text-ds-text-primary">NRCC</p>
            <p className="text-pretty text-xs text-ds-text-secondary">{t('auth:brand.purpose')}</p>
          </div>
        </div>
        <section className="space-y-4 rounded-box border border-ds-border-default bg-ds-bg-surface p-4 sm:p-6">
          {children}
        </section>
      </div>
    </main>
  );
}

/** Inline form-level failure; the form keeps the user's input. */
export function AuthAlert({ children }: { children?: ReactNode }) {
  return (
    <div role="alert" className="flex gap-2 rounded-box border border-error px-3 py-2 text-sm text-error">
      <AlertCircle aria-hidden="true" className="mt-0.5 h-4 w-4 shrink-0" />
      <div className="min-w-0 space-y-1">{children}</div>
    </div>
  );
}

export function LoginView() {
  const { t } = useT();
  const navigate = useNavigate();
  const location = useLocation();
  const { login } = useAuth();
  const from = (location.state as { from?: Location })?.from?.pathname || '/dashboard';

  // Check auth status to determine if system is initialized
  const { data: status, isLoading: isStatusLoading } = useQuery({
    queryKey: queryKeys.auth.status,
    queryFn: authService.getStatus,
    retry: false,
    staleTime: 0,
  });

  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
    setError,
  } = useForm<LoginFormData>({
    resolver: zodResolver(loginSchema),
  });

  const onSubmit = async (data: LoginFormData) => {
    try {
      await login(data.username, data.password);
      toast.success(t('auth:login.success'));
      navigate(from, { replace: true });
    } catch (err: unknown) {
      // Copy by outcome: no response = network, then 401, 429, anything else.
      const status = (err as { response?: { status?: number } })?.response?.status;
      const message =
        status === undefined ? t('auth:login.errors.network')
        : status === 401 ? t('auth:login.errors.invalidCredentials')
        : status === 429 ? t('auth:login.errors.rateLimited')
        : t('auth:login.errors.unexpected', { status });
      // Inline only: the alert below the heading carries it, a toast would announce it twice.
      setError('root', { message });
    }
  };

  if (isStatusLoading) {
    return (
      <AuthPage>
        <p role="status" className="text-sm text-ds-text-secondary">{t('auth:login.checkingStatus')}</p>
      </AuthPage>
    );
  }

  if (status?.initialized === false) {
    return (
      <AuthPage>
        <h1 className="text-balance text-xl font-bold text-ds-text-primary">{t('auth:login.systemNotInitialized')}</h1>
        <p className="text-sm text-ds-text-secondary">{t('auth:login.setupAction')}</p>
        <Link to="/setup" className={buttonVariants({ variant: 'solid', size: 'lg', className: 'w-full' })}>
          {t('auth:login.createAdminAction')}
        </Link>
      </AuthPage>
    );
  }

  return (
    <AuthPage>
      <h1 className="text-balance text-xl font-bold text-ds-text-primary">{t('auth:login.title')}</h1>
      <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
        {errors.root && <AuthAlert><p>{errors.root.message}</p></AuthAlert>}

        <Input
          id="username"
          type="text"
          label={t('auth:login.usernameLabel')}
          autoComplete="username"
          autoCapitalize="none"
          className="min-h-11"
          error={errors.username?.message && t(errors.username.message)}
          {...register('username')}
        />

        <Input
          id="password"
          type="password"
          label={t('auth:login.passwordLabel')}
          autoComplete="current-password"
          className="min-h-11"
          error={errors.password?.message && t(errors.password.message)}
          {...register('password')}
        />

        <Button
          type="submit"
          variant="solid"
          size="lg"
          className="w-full"
          disabled={isSubmitting}
          aria-busy={isSubmitting}
        >
          {isSubmitting ? t('auth:login.submitting') : t('auth:login.submit')}
        </Button>
      </form>
    </AuthPage>
  );
}
