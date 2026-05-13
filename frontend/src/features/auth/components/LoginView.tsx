import { useNavigate, useLocation, Link } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { useAuth } from '../hooks/useAuth';
import { authService } from '../services/authService';
import { toast } from 'sonner';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { AlertCircle } from 'lucide-react';

const loginSchema = z.object({
  username: z.string().min(1, 'Username is required'),
  password: z.string().min(1, 'Password is required'),
});

type LoginFormData = z.infer<typeof loginSchema>;

export function LoginView() {
  const navigate = useNavigate();
  const location = useLocation();
  const { login } = useAuth();
  const from = (location.state as { from?: Location })?.from?.pathname || '/dashboard';

  // Check auth status to determine if system is initialized
  const { data: status, isLoading: isStatusLoading } = useQuery({
    queryKey: ['authStatus'],
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
      toast.success('Logged in successfully!');
      navigate(from, { replace: true });
    } catch (err: unknown) {
      const axiosError = err as { response?: { data?: { error?: { message?: string } } } };
      const message = axiosError.response?.data?.error?.message || 'Invalid credentials';
      setError('root', { message });
      toast.error(message);
    }
  };

  // Show loading state while checking auth status
  if (isStatusLoading) {
    return (
      <div className="auth-shell min-h-screen flex items-center justify-center px-4">
        <div className="h-8 w-8 animate-spin rounded-full border-b-2 border-primary"></div>
      </div>
    );
  }

  // If system is not initialized, show "Sistema no inicializado" card
  if (status?.initialized === false) {
    return (
      <div className="auth-shell min-h-screen flex items-center justify-center px-4">
        <div className="surface-card w-full max-w-md space-y-6 border border-border p-8">
          <div className="text-center">
            <div className="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-2xl bg-warning/15">
              <AlertCircle className="h-6 w-6 text-warning" />
            </div>
            <h1 className="text-2xl font-bold text-base-content">Sistema No Inicializado</h1>
            <p className="mt-2 text-base-content/70">
              Aún no se ha configurado el usuario administrador.
            </p>
          </div>

          <div className="pt-4">
            <Link
              to="/setup"
              className="block w-full rounded-xl bg-primary px-4 py-2 text-center font-medium text-primary-content hover:bg-primary/90"
            >
              Crear Usuario Administrador
            </Link>
          </div>
        </div>
      </div>
    );
  }

  // Default: show login form
  return (
    <div className="auth-shell min-h-screen flex items-center justify-center px-4">
      <div className="surface-card w-full max-w-md space-y-6 border border-border p-8">
        <div className="text-center">
          <p className="mb-2 text-xs uppercase tracking-[0.24em] text-base-content/50">Access console</p>
          <h1 className="text-2xl font-bold text-base-content">Node-RED Control Center</h1>
          <p className="mt-2 text-base-content/70">Sign in to your account</p>
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

          <button
            type="submit"
            disabled={isSubmitting}
            className="w-full rounded-xl bg-primary px-4 py-2 text-primary-content hover:bg-primary/90 disabled:cursor-not-allowed disabled:opacity-50"
          >
            {isSubmitting ? 'Signing in...' : 'Sign in'}
          </button>
        </form>
      </div>
    </div>
  );
}
