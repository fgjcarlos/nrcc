import { useEffect } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { authService } from '../services/authService';
import { toast } from 'sonner';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { useT } from '@/i18n';
import { Button, Input } from '@/shared/components/ui';
import { AuthAlert, AuthPage } from './LoginView';

// Messages are catalog keys; the view translates them at render.
const setupSchema = z.object({
  username: z
    .string()
    .min(3, 'auth:setup.errors.usernameLength')
    .max(32, 'auth:setup.errors.usernameLength')
    .regex(
      /^[a-zA-Z0-9_]+$/,
      'auth:setup.errors.usernameChars'
    ),
  password: z
    .string()
    .min(8, 'auth:setup.errors.passwordLength'),
  confirmPassword: z.string(),
}).refine((data) => data.password === data.confirmPassword, {
  message: 'auth:setup.errors.passwordMismatch',
  path: ['confirmPassword'],
});

type SetupFormData = z.infer<typeof setupSchema>;

export function SetupView() {
  const navigate = useNavigate();
  const { t } = useT();

  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
    setError,
  } = useForm<SetupFormData>({
    resolver: zodResolver(setupSchema),
  });

  // Check if already initialized
  useEffect(() => {
    authService.getStatus()
      .then(status => {
        if (status.initialized) {
          navigate('/login', { replace: true });
        }
      })
      .catch(() => {
        // Allow setup if status check fails
      });
  }, [navigate]);

  const onSubmit = async (data: SetupFormData) => {
    try {
      await authService.setup(data.username, data.password);
      toast.success(t('auth:setup.success'));
      navigate('/dashboard', { replace: true });
    } catch (error: unknown) {
      const err = error as { response?: { status?: number; data?: { error?: { code?: string } } } };
      // If already initialized (403), redirect to login
      if (err.response?.status === 403) {
        toast.error(t('auth:setup.errors.alreadyConfiguredRedirect'));
        navigate('/login', { replace: true });
        return;
      }
      // Copy by outcome: no response = network, then 409, 400 weak password, 429, anything else.
      const status = err.response?.status;
      const code = err.response?.data?.error?.code;
      if (status === 409) {
        setError('root', { type: 'alreadyConfigured', message: t('auth:setup.errors.alreadyConfigured') });
        return;
      }
      const message =
        status === undefined ? t('auth:setup.errors.network')
        : code === 'WEAK_PASSWORD' ? t('auth:setup.errors.weakPassword')
        : status === 429 ? t('auth:setup.errors.rateLimited')
        : t('auth:setup.errors.unexpected', { status });
      // Inline only: the alert carries it, a toast would announce it twice.
      setError('root', { message });
    }
  };

  return (
    <AuthPage>
      <h1 className="text-balance text-xl font-bold text-ds-text-primary">{t('auth:setup.title')}</h1>
      <p className="text-sm text-ds-text-secondary">{t('auth:setup.intro')}</p>

      <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
        {errors.root && (
          <AuthAlert>
            <p>{errors.root.message}</p>
            {errors.root.type === 'alreadyConfigured' && (
              <Link to="/login" className="inline-flex min-h-11 items-center font-semibold underline underline-offset-2">
                {t('auth:setup.goToSignIn')}
              </Link>
            )}
          </AuthAlert>
        )}

        <div className="space-y-1">
          <Input
            id="username"
            type="text"
            label={t('auth:setup.usernameLabel')}
            autoComplete="username"
            autoCapitalize="none"
            aria-describedby="username-hint"
            className="min-h-11"
            error={errors.username?.message && t(errors.username.message)}
            {...register('username')}
          />
          <p id="username-hint" className="text-xs text-ds-text-secondary">{t('auth:setup.usernameHint')}</p>
        </div>

        <div className="space-y-1">
          <Input
            id="password"
            type="password"
            label={t('auth:setup.passwordLabel')}
            autoComplete="new-password"
            aria-describedby="password-hint"
            className="min-h-11"
            error={errors.password?.message && t(errors.password.message)}
            {...register('password')}
          />
          <p id="password-hint" className="text-xs text-ds-text-secondary">{t('auth:setup.passwordHint')}</p>
        </div>

        <Input
          id="confirmPassword"
          type="password"
          label={t('auth:setup.confirmPasswordLabel')}
          autoComplete="new-password"
          className="min-h-11"
          error={errors.confirmPassword?.message && t(errors.confirmPassword.message)}
          {...register('confirmPassword')}
        />

        <Button
          type="submit"
          variant="solid"
          size="lg"
          className="w-full"
          disabled={isSubmitting}
          aria-busy={isSubmitting}
        >
          {isSubmitting ? t('auth:setup.creatingAccount') : t('auth:setup.createAccount')}
        </Button>
      </form>
    </AuthPage>
  );
}
