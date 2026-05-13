import { useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { authService } from '../services/authService';
import { toast } from 'sonner';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';

const setupSchema = z.object({
  username: z
    .string()
    .min(3, 'Username must be 3-32 characters')
    .max(32, 'Username must be 3-32 characters')
    .regex(
      /^[a-zA-Z0-9_]+$/,
      'Username can only contain letters, numbers, and underscores'
    ),
  password: z
    .string()
    .min(8, 'Password must be at least 8 characters'),
  confirmPassword: z.string(),
}).refine((data) => data.password === data.confirmPassword, {
  message: 'Passwords do not match',
  path: ['confirmPassword'],
});

type SetupFormData = z.infer<typeof setupSchema>;

export function SetupView() {
  const navigate = useNavigate();

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
      toast.success('Setup completed successfully!');
      navigate('/dashboard', { replace: true });
    } catch (error: unknown) {
      const err = error as { response?: { status?: number; data?: { error?: { message?: string } } } };
      // If already initialized (403), redirect to login
      if (err.response?.status === 403) {
        toast.error('System already initialized. Redirecting to login...');
        navigate('/login', { replace: true });
        return;
      }
      const message = err.response?.data?.error?.message || 'Setup failed';
      setError('root', { message });
      toast.error(message);
    }
  };

  return (
    <div className="auth-shell min-h-screen flex items-center justify-center px-4">
      <div className="surface-card w-full max-w-md space-y-6 border border-border p-8">
        <div className="text-center">
          <p className="mb-2 text-xs uppercase tracking-[0.24em] text-base-content/50">Bootstrap</p>
          <h1 className="text-2xl font-bold text-base-content">Node-RED Control Center</h1>
          <p className="mt-2 text-base-content/70">Initial Setup</p>
        </div>

        <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
          {errors.root && (
            <div className="rounded-xl border border-error/20 bg-error/10 p-3 text-sm text-error">
              {errors.root.message}
            </div>
          )}

          <div>
            <label htmlFor="username" className="mb-1 block text-sm font-medium text-base-content">
              Username
            </label>
            <input
              id="username"
              type="text"
              {...register('username')}
              className="w-full rounded-xl border border-border bg-base-100/70 px-3 py-2 text-base-content focus:outline-none focus:ring-2 focus:ring-primary"
              placeholder="admin"
            />
            {errors.username && (
              <p className="mt-1 text-sm text-error">{errors.username.message}</p>
            )}
          </div>

          <div>
            <label htmlFor="password" className="mb-1 block text-sm font-medium text-base-content">
              Password
            </label>
            <input
              id="password"
              type="password"
              {...register('password')}
              className="w-full rounded-xl border border-border bg-base-100/70 px-3 py-2 text-base-content focus:outline-none focus:ring-2 focus:ring-primary"
              placeholder="••••••••"
            />
            {errors.password && (
              <p className="mt-1 text-sm text-error">{errors.password.message}</p>
            )}
          </div>

          <div>
            <label htmlFor="confirmPassword" className="mb-1 block text-sm font-medium text-base-content">
              Confirm Password
            </label>
            <input
              id="confirmPassword"
              type="password"
              {...register('confirmPassword')}
              className="w-full rounded-xl border border-border bg-base-100/70 px-3 py-2 text-base-content focus:outline-none focus:ring-2 focus:ring-primary"
              placeholder="••••••••"
            />
            {errors.confirmPassword && (
              <p className="mt-1 text-sm text-error">{errors.confirmPassword.message}</p>
            )}
          </div>

          <button
            type="submit"
            disabled={isSubmitting}
            className="w-full rounded-xl bg-primary px-4 py-2 text-primary-content hover:bg-primary/90 disabled:cursor-not-allowed disabled:opacity-50"
          >
            {isSubmitting ? 'Creating account...' : 'Create account and continue'}
          </button>
        </form>
      </div>
    </div>
  );
}
