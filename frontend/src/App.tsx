import { lazy, Suspense, useEffect, useState, type ComponentType, type ReactNode } from 'react';
import { BrowserRouter, Navigate, Routes, Route, useNavigate } from 'react-router-dom';
import { setNavigator } from '@/shared/lib/navigation';
import { AlertCircle, Loader2 } from 'lucide-react';
import { Layout } from '@/shared/components/layout/Layout';
import { ProtectedRoute } from '@/shared/components/ProtectedRoute';
import { useAuth } from '@/features/auth/hooks/useAuth';
import { authService } from '@/features/auth/services/authService';
import { ErrorBoundary } from '@/shared/components/layout/ErrorBoundary';
import { Button } from '@/shared/components/ui/Button';
import { useT, I18nProvider } from '@/i18n';

function lazyNamed<T extends ComponentType<object>>(
  importer: () => Promise<Record<string, T>>,
  exportName: string,
) {
  return lazy(async () => ({ default: (await importer())[exportName] }));
}

// Public pages (no layout)
const SetupView = lazyNamed(
  () => import('@/features/auth/components/SetupView'),
  'SetupView',
);
const LoginView = lazyNamed(
  () => import('@/features/auth/components/LoginView'),
  'LoginView',
);

// Feature views (protected routes with layout)
const DashboardView = lazyNamed(
  () => import('@/features/dashboard/components/DashboardView'),
  'DashboardView',
);
const ConfigurationView = lazyNamed(
  () => import('@/features/configuration/components/ConfigurationView'),
  'ConfigurationView',
);
const ProfileView = lazyNamed(
  () => import('@/features/auth/components/ProfileView'),
  'ProfileView',
);
const UsersView = lazyNamed(
  () => import('@/features/auth/components/UsersView'),
  'UsersView',
);
const UpdatesView = lazyNamed(
  () => import('@/features/updates/components/UpdatesView'),
  'UpdatesView',
);
const LibrariesView = lazyNamed(
  () => import('@/features/libraries/components/LibrariesView'),
  'LibrariesView',
);
const BootstrapView = lazyNamed(
  () => import('@/features/bootstrap/components/BootstrapView'),
  'BootstrapView',
);
const EnvVarsView = lazyNamed(
  () => import('@/features/env-vars/components/EnvVarsView'),
  'EnvVarsView',
);
const BackupsView = lazyNamed(
  () => import('@/features/backups/components/BackupsView'),
  'BackupsView',
);

function RouteLoadingFallback({ label }: { label: string }) {
  const { t } = useT();
  return (
    <div className="flex min-h-[24rem] flex-col items-center justify-center gap-3 rounded-box border border-base-300 bg-base-100 p-8 text-center shadow-sm">
      <Loader2 className="h-8 w-8 animate-spin text-primary" aria-hidden="true" />
      <div>
        <p className="font-medium text-base-content">{t('common:loadingWithLabel', { label })}</p>
        <p className="text-sm text-base-content/60">{t('common:preparingSection')}</p>
      </div>
    </div>
  );
}

function RouteErrorFallback({ label, onRetry }: { label: string; onRetry: () => void }) {
  const { t } = useT();
  return (
    <div className="rounded-box border border-error/20 bg-error/8 p-6 shadow-sm">
      <div className="flex flex-col gap-4 sm:flex-row sm:items-start">
        <AlertCircle className="mt-0.5 h-6 w-6 flex-shrink-0 text-error" aria-hidden="true" />
        <div className="flex-1">
          <h2 className="text-lg font-semibold text-base-content">{t('common:unableToLoad', { label })}</h2>
          <p className="mt-1 text-sm text-base-content/70">
            {t('common:routeLoadFailed')}
          </p>
        </div>
        <Button type="button" onClick={onRetry} variant="secondary" size="sm">
          {t('common:tryAgain')}
        </Button>
      </div>
    </div>
  );
}

function RouteBoundary({ label, children }: { label: string; children: ReactNode }) {
  const [attempt, setAttempt] = useState(0);

  return (
    <ErrorBoundary
      key={`${label}-${attempt}`}
      fallback={<RouteErrorFallback label={label} onRetry={() => setAttempt((current) => current + 1)} />}
    >
      <Suspense fallback={<RouteLoadingFallback label={label} />}>{children}</Suspense>
    </ErrorBoundary>
  );
}

function routeElement(label: string, view: ReactNode, requiredRole?: 'admin') {
  return (
    <RouteBoundary label={label}>
      <ProtectedRoute requiredRole={requiredRole}>{view}</ProtectedRoute>
    </RouteBoundary>
  );
}

function RootRedirect() {
  const { isAuthenticated, isLoading } = useAuth();
  const [serverInitialized, setServerInitialized] = useState<boolean | null>(null);

  useEffect(() => {
    let cancelled = false;

    void authService
      .getStatus()
      .then(({ initialized }) => {
        if (!cancelled) {
          setServerInitialized(initialized);
        }
      })
      .catch(() => {
        if (!cancelled) {
          setServerInitialized(true);
        }
      });

    return () => {
      cancelled = true;
    };
  }, []);

  if (isLoading || serverInitialized === null) {
    return <RouteLoadingFallback label="home" />;
  }

  if (!serverInitialized) {
    return <Navigate to="/setup" replace />;
  }

  return <Navigate to={isAuthenticated ? '/overview' : '/login'} replace />;
}

function publicRouteElement(label: string, view: ReactNode) {
  return <RouteBoundary label={label}>{view}</RouteBoundary>;
}

function AppRoutes() {
  return (
    <Routes>
      {/* Public routes without layout */}
      <Route path="/" element={<RootRedirect />} />
      <Route path="/setup" element={publicRouteElement('setup', <SetupView />)} />
      <Route path="/login" element={publicRouteElement('login', <LoginView />)} />

      {/* Protected routes with layout */}
      <Route path="/" element={<Layout />}>
        <Route path="overview" element={routeElement('overview', <DashboardView />)} />
        <Route path="dashboard" element={<Navigate to="/overview" replace />} />
        <Route path="configuration" element={routeElement('configuration', <ConfigurationView />)} />
        <Route path="profile" element={routeElement('profile', <ProfileView />)} />
        <Route path="settings/users" element={routeElement('users', <UsersView />, 'admin')} />
        <Route path="maintenance/updates" element={routeElement('maintenance updates', <UpdatesView />, 'admin')} />
        <Route path="maintenance/libraries" element={routeElement('maintenance libraries', <LibrariesView />, 'admin')} />
        <Route path="updates" element={<Navigate to="/maintenance/updates" replace />} />
        <Route path="libraries" element={<Navigate to="/maintenance/libraries" replace />} />
        <Route path="flows" element={<Navigate to="/overview" replace />} />
        <Route path="flows/versions" element={<Navigate to="/backups" replace />} />
        <Route path="flows/:id" element={<Navigate to="/overview" replace />} />
        <Route path="bootstrap" element={routeElement('bootstrap', <BootstrapView />)} />
        <Route path="environment" element={routeElement('environment variables', <EnvVarsView />)} />
        <Route path="backups" element={routeElement('recovery', <BackupsView />)} />
        <Route path="files" element={<Navigate to="/overview" replace />} />
      </Route>
    </Routes>
  );
}

// Registers React Router's navigate with the navigation bridge so non-React
// code (the axios interceptor) can redirect without a full page reload.
function NavigatorRegistrar() {
  const navigate = useNavigate();
  useEffect(() => {
    setNavigator((path) => navigate(path));
    return () => setNavigator(null);
  }, [navigate]);
  return null;
}

function App() {
  return (
    <I18nProvider>
      <BrowserRouter>
        <ErrorBoundary>
          <NavigatorRegistrar />
          <AppRoutes />
        </ErrorBoundary>
      </BrowserRouter>
    </I18nProvider>
  );
}

export default App;
