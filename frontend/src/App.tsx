import { useEffect } from 'react'
import { Navigate, Route, Routes, useLocation, useNavigate } from 'react-router-dom'
import { AnimatePresence } from 'framer-motion'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { api } from './api'
import { ThemeProvider, PageTransition, ErrorBoundary, InlineNotice } from './common/components'
import { useAuth } from './features/auth/useAuth'
import { AuthScreen } from './features/auth/AuthScreen'
import { LoadingScreen } from './features/auth/LoadingScreen'
import { useToasts } from './features/shell/useToasts'
import { useGlobalStatus } from './features/shell/useGlobalStatus'
import { DashboardShell } from './features/shell/DashboardShell'
import { ToastViewport } from './features/shell/ToastViewport'
import { OverviewPage } from './features/overview/OverviewPage'
import { LogsPage } from './features/logs/LogsPage'
import { EnvironmentPage } from './features/environment/EnvironmentPage'
import { BackupsPage } from './features/backups/BackupsPage'
import { FlowsPage } from './features/flows/FlowsPage'
import { LibrariesPage } from './features/libraries/LibrariesPage'
import { UpdatesPage } from './features/updates/UpdatesPage'
import { DiagnosticsPage } from './features/diagnostics/DiagnosticsPage'
import { ConfigPage } from './features/config/ConfigPage'
import { UsersPage } from './features/users/UsersPage'

function AccessDeniedPage({ userRole, onLogout, logoutBusy }: { userRole: string; onLogout: () => void; logoutBusy: boolean }) {
  return (
    <PageTransition>
      <main id="auth-main" tabIndex={-1} className="min-h-screen bg-base-100 px-6 py-16">
        <div className="mx-auto flex max-w-2xl flex-col gap-6">
          <header>
            <p className="text-xs uppercase tracking-[0.28em] text-base-content/50">Access denied</p>
            <h1 className="mt-2 text-3xl font-semibold text-base-content">This first slice is admin-only</h1>
            <p className="mt-3 text-sm text-base-content/70">
              Signed in as <span className="font-semibold">{userRole}</span>. Viewer and operator rollout is intentionally deferred, so this account cannot use the control center yet.
            </p>
          </header>
          <InlineNotice
            tone="info"
            title="Administrator required"
            detail="Ask an administrator to promote this account or sign in with an existing administrator account."
          />
          <div>
            <button className="btn btn-primary" type="button" onClick={onLogout} disabled={logoutBusy}>
              {logoutBusy ? 'Signing out...' : 'Sign out'}
            </button>
          </div>
        </div>
      </main>
    </PageTransition>
  )
}

function AppContent() {
  const navigate = useNavigate()
  const location = useLocation()
  const queryClient = useQueryClient()
  const { toasts, pushToast, dismissToast } = useToasts()
  const { user, isLoading, authMode, setAuthMode, authMessage, loginMutation, registerMutation, logoutMutation } = useAuth(pushToast)
  const isAdmin = user?.role === 'admin'
  const globalStatus = useGlobalStatus()

  // Set up auth navigation effects
  useEffect(() => {
    if (user && location.pathname === '/login') {
      navigate('/app/overview', { replace: true })
    }
    if (!user && location.pathname.startsWith('/app')) {
      navigate('/login', { replace: true })
    }
  }, [user, location.pathname, navigate])

  useEffect(() => {
    const focusTargetId = location.pathname.startsWith('/app') ? 'main-content' : 'auth-main'
    const frame = window.requestAnimationFrame(() => {
      const target = document.getElementById(focusTargetId)
      target?.focus({ preventScroll: true })
    })

    return () => window.cancelAnimationFrame(frame)
  }, [location.pathname])

  // Pass pushToast callback to auth mutations for toast notifications
  useEffect(() => {
    if (loginMutation.isSuccess) {
      pushToast({
        tone: 'success',
        title: 'Signed in',
        detail: 'The local administrator session is active.',
      })
    }
  }, [loginMutation.isSuccess])

  useEffect(() => {
    if (registerMutation.isSuccess) {
      pushToast({
        tone: 'success',
        title: 'Administrator created',
        detail: 'Bootstrap completed and the local session is ready.',
      })
    }
  }, [registerMutation.isSuccess])

  useEffect(() => {
    if (logoutMutation.isSuccess) {
      pushToast({
        tone: 'info',
        title: 'Signed out',
        detail: 'The local session has been closed.',
      })
    }
  }, [logoutMutation.isSuccess])

  // operationsQuery kept for ConfigPage (not in refactor scope)
  const operationsQuery = useQuery({
    queryKey: ['operations-status'],
    queryFn: api.operationsStatus,
    enabled: !!user && isAdmin,
    refetchInterval: 2000,
  })

  if (isLoading) {
    return (
      <>
        <LoadingScreen label="Loading local control center" />
        <ToastViewport toasts={toasts} onDismiss={dismissToast} />
      </>
    )
  }

  return (
    <>
      <AnimatePresence mode="wait">
        <Routes>
          <Route
            path="/login"
            element={
              user ? (
                <Navigate to="/app/overview" replace />
              ) : (
                <PageTransition>
                   <AuthScreen
                     mode={authMode}
                     message={authMessage}
                     busy={loginMutation.isPending || registerMutation.isPending}
                     onSubmit={(username, password) => {
                       if (authMode === 'register') {
                         registerMutation.mutate({ username, password })
                       } else {
                         loginMutation.mutate({ username, password })
                       }
                     }}
                   />
                 </PageTransition>
              )
            }
          />
          <Route
            path="/app/*"
            element={
              user ? (
                isAdmin ? (
                  <DashboardShell
                    user={user}
                    globalStatus={globalStatus}
                    logoutBusy={logoutMutation.isPending}
                    onLogout={() => logoutMutation.mutate()}
                  >
                    <AnimatePresence mode="wait">
                      <Routes>
                        <Route
                           path="overview"
                           element={
                             <PageTransition>
                               <OverviewPage />
                             </PageTransition>
                           }
                         />
                        <Route
                           path="logs"
                           element={
                             <PageTransition>
                               <LogsPage />
                             </PageTransition>
                           }
                         />
                        <Route
                          path="config"
                          element={
                            <PageTransition>
                              <ConfigPage
                                operationStatus={operationsQuery.data}
                                onSaved={(restartRequired) => {
                                  pushToast({
                                    tone: 'success',
                                    title: 'Configuration saved',
                                    detail: restartRequired
                                      ? 'Saved successfully. Restart Node-RED to apply the changes.'
                                      : 'Saved successfully.',
                                  })
                                }}
                                onError={(message) => {
                                  pushToast({
                                    tone: 'error',
                                    title: 'Configuration failed',
                                    detail: message,
                                  })
                                }}
                                onToast={(message, type) => {
                                  pushToast({
                                    tone: type,
                                    title: message.split('\n')[0],
                                    detail: message.split('\n').slice(1).join('\n') || undefined,
                                  })
                                }}
                              />
                            </PageTransition>
                          }
                        />
                        <Route
                           path="environment"
                           element={
                             <PageTransition>
                               <EnvironmentPage />
                             </PageTransition>
                           }
                         />
                        <Route
                          path="backups"
                          element={
                            <PageTransition>
                              <BackupsPage />
                            </PageTransition>
                          }
                        />
                        <Route
                          path="flows"
                          element={
                            <PageTransition>
                              <FlowsPage />
                            </PageTransition>
                          }
                        />
                        <Route
                          path="flows/:flowId"
                          element={
                            <PageTransition>
                              <FlowsPage />
                            </PageTransition>
                          }
                        />
                        <Route
                          path="libraries"
                          element={
                            <PageTransition>
                              <LibrariesPage />
                            </PageTransition>
                          }
                        />
                        <Route
                          path="updates"
                          element={
                            <PageTransition>
                              <UpdatesPage />
                            </PageTransition>
                          }
                        />
                        <Route
                           path="diagnostics"
                           element={
                             <PageTransition>
                               <DiagnosticsPage />
                             </PageTransition>
                           }
                         />
                        <Route
                          path="users"
                          element={
                            <PageTransition>
                              <UsersPage
                                currentUser={user}
                                onToast={(title, detail, tone) => {
                                  pushToast({ title, detail, tone })
                                }}
                                onSessionRevoked={async () => {
                                  await queryClient.invalidateQueries({ queryKey: ['me'] })
                                  pushToast({
                                    tone: 'info',
                                    title: 'Session ended',
                                    detail: 'Your role or password changed, so this session was closed.',
                                  })
                                  navigate('/login', { replace: true })
                                }}
                              />
                            </PageTransition>
                          }
                        />
                        <Route path="*" element={<Navigate to="/app/overview" replace />} />
                      </Routes>
                    </AnimatePresence>
                  </DashboardShell>
                ) : (
                  <AccessDeniedPage userRole={user.role} onLogout={() => logoutMutation.mutate()} logoutBusy={logoutMutation.isPending} />
                )
              ) : (
                <Navigate to="/login" replace />
              )
            }
          />
          <Route path="*" element={<Navigate to={user ? '/app/overview' : '/login'} replace />} />
        </Routes>
      </AnimatePresence>
      <ToastViewport toasts={toasts} onDismiss={dismissToast} />
    </>
  )
}

export function App() {
  return (
    <ErrorBoundary>
      <ThemeProvider>
        <AppContent />
      </ThemeProvider>
    </ErrorBoundary>
  )
}
