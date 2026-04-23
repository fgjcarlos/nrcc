import { useEffect } from 'react'
import { Navigate, Route, Routes, useLocation, useNavigate } from 'react-router-dom'
import { AnimatePresence } from 'framer-motion'
import { useQueryClient } from '@tanstack/react-query'

import { api } from './api'
import { ThemeProvider, PageTransition, ErrorBoundary, InlineNotice } from './common/components'
import { useAuth } from './features/auth/useAuth'
import { AuthScreen } from './features/auth/AuthScreen'
import { LoadingScreen } from './features/auth/LoadingScreen'
import { AccessDeniedPage } from './features/auth/AccessDeniedPage'
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
                              <ConfigPage />
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
