import { BrowserRouter, Routes, Route } from 'react-router-dom';
import { Layout } from '@/shared/components/layout/Layout';
import { ProtectedRoute } from '@/shared/components/ProtectedRoute';
import { ErrorBoundary } from '@/shared/components/layout/ErrorBoundary';
import { ToastViewport } from './shared/components/ui/ToastViewport';

// Public pages (no layout)
import { LandingView, LoginView, SetupView } from '@/features/auth/components';

// Feature views (protected routes with layout)
import { DashboardView } from '@/features/dashboard';
import { ConfigurationView } from '@/features/configuration/components/ConfigurationView';
import { LogsView } from '@/features/logs/components/LogsView';
import { DockerView } from '@/features/docker/components/DockerView';
import { UsersView } from '@/features/auth/components/UsersView';
import { UpdatesView } from '@/features/updates/components/UpdatesView';
import { LibrariesView } from '@/features/libraries/components/LibrariesView';
import { FlowsView } from '@/features/flows/components/FlowsView';
import { FlowDetailView } from '@/features/flows/components/FlowDetailView';
import { EnvVarsView } from '@/features/env-vars/components/EnvVarsView';
import { BackupsView } from '@/features/backups/components/BackupsView';
import { BootstrapView } from '@/features/bootstrap/components/BootstrapView';

function AppRoutes() {
  return (
    <Routes>
      {/* Public routes without layout */}
      <Route path="/" element={<LandingView />} />
      <Route path="/setup" element={<SetupView />} />
      <Route path="/login" element={<LoginView />} />

      {/* Protected routes with layout */}
      <Route path="/" element={<Layout />}>
        <Route
          path="dashboard"
          element={
            <ProtectedRoute>
              <DashboardView />
            </ProtectedRoute>
          }
        />
        <Route
          path="configuration"
          element={
            <ProtectedRoute>
              <ConfigurationView />
            </ProtectedRoute>
          }
        />
        <Route
          path="logs"
          element={
            <ProtectedRoute>
              <LogsView />
            </ProtectedRoute>
          }
        />
        <Route
          path="docker"
          element={
            <ProtectedRoute>
              <DockerView />
            </ProtectedRoute>
          }
        />
        <Route
          path="settings/users"
          element={
            <ProtectedRoute requiredRole="admin">
              <UsersView />
            </ProtectedRoute>
          }
        />
        <Route
          path="updates"
          element={
            <ProtectedRoute>
              <UpdatesView />
            </ProtectedRoute>
          }
        />
        <Route
          path="libraries"
          element={
            <ProtectedRoute>
              <LibrariesView />
            </ProtectedRoute>
          }
        />
        <Route
          path="flows"
          element={
            <ProtectedRoute>
              <FlowsView />
            </ProtectedRoute>
          }
        />
        <Route
          path="flows/:id"
          element={
            <ProtectedRoute>
              <FlowDetailView />
            </ProtectedRoute>
          }
        />
        <Route
          path="bootstrap"
          element={
            <ProtectedRoute>
              <BootstrapView />
            </ProtectedRoute>
          }
        />
        <Route
          path="environment"
          element={
            <ProtectedRoute>
              <EnvVarsView />
            </ProtectedRoute>
          }
        />
        <Route
          path="backups"
          element={
            <ProtectedRoute>
              <BackupsView />
            </ProtectedRoute>
          }
        />
      </Route>
    </Routes>
  );
}

function App() {
  return (
    <BrowserRouter>
      <ErrorBoundary>
        <AppRoutes />
        <ToastViewport />
      </ErrorBoundary>
    </BrowserRouter>
  );
}

export default App;
