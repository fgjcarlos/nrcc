import { useState, useEffect } from 'react'
import { useNavigate, useLocation } from 'react-router-dom'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api, type User, APIRequestError } from '../../api'
import { formatErrorMessage } from '../../common/utils/format'
import type { AuthMode, Toast } from '../../common/types'

type OnToast = (toast: Omit<Toast, 'id'>) => void

export function useAuth(onToast?: OnToast) {
  const queryClient = useQueryClient()
  const location = useLocation()
  const navigate = useNavigate()
  const [authMode, setAuthMode] = useState<AuthMode>('login')
  const [authMessage, setAuthMessage] = useState('')

  const authStatusQuery = useQuery({
    queryKey: ['auth-status'],
    queryFn: api.authStatus,
    retry: false,
  })

  const meQuery = useQuery({
    queryKey: ['me'],
    queryFn: api.me,
    retry: false,
  })

  useEffect(() => {
    if (authStatusQuery.data?.hasUsers === false) {
      setAuthMode('register')
    } else {
      setAuthMode('login')
    }
  }, [authStatusQuery.data?.hasUsers])

  useEffect(() => {
    if (meQuery.isSuccess && location.pathname === '/login') {
      navigate('/app/overview', { replace: true })
    }
    if (meQuery.isError && location.pathname !== '/login') {
      navigate('/login', { replace: true })
    }
  }, [location.pathname, meQuery.isError, meQuery.isSuccess, navigate])

  const loginMutation = useMutation({
    mutationFn: ({ username, password }: { username: string; password: string }) =>
      api.login(username, password),
    onSuccess: async () => {
      setAuthMessage('')
      onToast?.({
        tone: 'success',
        title: 'Signed in',
        detail: 'The local administrator session is active.',
      })
      await queryClient.invalidateQueries({ queryKey: ['me'] })
      await queryClient.invalidateQueries({ queryKey: ['auth-status'] })
    },
    onError: (error) => {
      const message = formatErrorMessage(error, 'Login failed')
      setAuthMessage(message)
      onToast?.({
        tone: 'error',
        title: 'Login failed',
        detail: message,
      })
    },
  })

  const registerMutation = useMutation({
    mutationFn: ({ username, password }: { username: string; password: string }) =>
      api.register(username, password),
    onSuccess: async () => {
      setAuthMessage('')
      onToast?.({
        tone: 'success',
        title: 'Administrator created',
        detail: 'Bootstrap completed and the local session is ready.',
      })
      await queryClient.invalidateQueries({ queryKey: ['me'] })
      await queryClient.invalidateQueries({ queryKey: ['auth-status'] })
    },
    onError: (error) => {
      const message = formatErrorMessage(error, 'Registration failed')
      setAuthMessage(message)
      onToast?.({
        tone: 'error',
        title: 'Bootstrap failed',
        detail: message,
      })
    },
  })

  const logoutMutation = useMutation({
    mutationFn: api.logout,
    onSuccess: async () => {
      onToast?.({
        tone: 'info',
        title: 'Signed out',
        detail: 'The local session has been closed.',
      })
      await queryClient.invalidateQueries({ queryKey: ['me'] })
    },
    onError: (error) => {
      onToast?.({
        tone: 'error',
        title: 'Sign out failed',
        detail: formatErrorMessage(error, 'Could not sign out'),
      })
    },
  })

  return {
    user: meQuery.data?.user,
    isLoading: authStatusQuery.isLoading || meQuery.isLoading,
    authMode,
    setAuthMode,
    authMessage,
    loginMutation,
    registerMutation,
    logoutMutation,
  }
}
