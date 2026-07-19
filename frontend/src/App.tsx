import { lazy, Suspense, useEffect, useState, type ComponentType, type ReactNode } from 'react';
import { BrowserRouter, Routes, Route, useNavigate } from 'react-router-dom';
import { setNavigator } from '@/shared/lib/navigation';
import { AlertCircle, Loader2 } from 'lucide-react';
import { Layout } from '@/shared/components/layout/Layout';
import { ProtectedRoute } from '@/shared/components/ProtectedRoute';
import { ErrorBoundary } from '@/shared/components/layout/ErrorBoundary';
import { Button } from '@/shared/components/ui/Button';

function lazyNamed<T extends ComponentType<object>>(
  importer: () => Promise<Record<string, T>>,
  exportName: string,
) {
  return lazy(async () => ({ default: (await importer())[exportName] }));
}

// Public pages (no layout)
const LandingView = lazyNamed(
  () => import('@/features/auth/components/LandingView'),
  'LandingView',
);
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
const FlowsView = lazyNamed(
  () => import('@/features/flows/components/FlowsView'),
  'FlowsView',
);
const FlowVersionsView = lazyNamed(
  () => import('@/features/flows/components/FlowVersionsView'),
  'FlowVersionsView',
);
const FlowDetailView = lazyNamed(
  () => import('@/features/flows/components/FlowDetailView'),
  'FlowDetailView',
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
const FilesView = lazyNamed(
  () => import('@/features/files/components/FilesView'),
  'FilesView',
);

function RouteLoadingFallback({ label }: { label: string }) {
  return (
    <div className="flex min-h-[24rem] flex-col items-center justify-center gap-3 rounded-box border border-base-300 bg-base-100 p-8 text-center shadow-sm">
      <Loader2 className="h-8 w-8 animate-spin text-primary" aria-hidden="true" />
      <div>
        <p className="font-medium text-base-content">Loading {label}</p>
        <p className="text-sm text-base-content/60">Preparing this section...</p>
      </div>
    </div>
  );
}

function RouteErrorFallback({ label, onRetry }: { label: string; onRetry: () => void }) {
  return (
    <div className="rounded-box border border-error/20 bg-error/8 p-6 shadow-sm">
      <div className="flex flex-col gap-4 sm:flex-row sm:items-start">
        <AlertCircle className="mt-0.5 h-6 w-6 flex-shrink-0 text-error" aria-hidden="true" />
        <div className="flex-1">
          <h2 className="text-lg font-semibold text-base-content">Unable to load {label}</h2>
          <p className="mt-1 text-sm text-base-content/70">
            This route failed to load. You can retry without leaving the current page.
          </p>
        </div>
        <Button type="button" onClick={onRetry} variant="secondary" size="sm">
          Try again
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

function publicRouteElement(label: string, view: ReactNode) {
  return <RouteBoundary label={label}>{view}</RouteBoundary>;
}

function AppRoutes() {
  return (
    <Routes>
      {/* Public routes without layout */}
      <Route path="/" element={publicRouteElement('home', <LandingView />)} />
      <Route path="/setup" element={publicRouteElement('setup', <SetupView />)} />
      <Route path="/login" element={publicRouteElement('login', <LoginView />)} />

      {/* Protected routes with layout */}
      <Route path="/" element={<Layout />}>
        <Route path="dashboard" element={routeElement('dashboard', <DashboardView />)} />
        <Route path="configuration" element={routeElement('configuration', <ConfigurationView />)} />
        <Route path="profile" element={routeElement('profile', <ProfileView />)} />
        <Route path="settings/users" element={routeElement('users', <UsersView />, 'admin')} />
        <Route path="updates" element={routeElement('updates', <UpdatesView />)} />
        <Route path="libraries" element={routeElement('libraries', <LibrariesView />)} />
        <Route path="flows" element={routeElement('flows', <FlowsView />)} />
        <Route path="flows/versions" element={routeElement('flow versions', <FlowVersionsView />)} />
        <Route path="flows/:id" element={routeElement('flow details', <FlowDetailView />)} />
        <Route path="bootstrap" element={routeElement('bootstrap', <BootstrapView />)} />
        <Route path="environment" element={routeElement('environment variables', <EnvVarsView />)} />
        <Route path="backups" element={routeElement('backups', <BackupsView />)} />
        <Route path="files" element={routeElement('files', <FilesView />)} />
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
    <BrowserRouter>
      <ErrorBoundary>
        <NavigatorRegistrar />
        <AppRoutes />
      </ErrorBoundary>
    </BrowserRouter>
  );
}

export default App;
