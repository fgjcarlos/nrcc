import { useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { api, APIRequestError, type User } from '../../api'
import { ConfirmDialog, EmptyState, InlineNotice, LoadingState } from '../../common/components'
import { formatErrorMessage } from '../../common/utils/format'

const roleOptions = ['admin', 'operator', 'viewer']

export function UsersPage({
  currentUser,
  onToast,
  onSessionRevoked,
}: {
  currentUser: User
  onToast: (title: string, detail: string, tone: 'success' | 'error' | 'info') => void
  onSessionRevoked: () => Promise<void>
}) {
  const queryClient = useQueryClient()
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [role, setRole] = useState('operator')
  const [roleDrafts, setRoleDrafts] = useState<Record<string, string>>({})
  const [passwordDrafts, setPasswordDrafts] = useState<Record<string, string>>({})
  const [deleteTarget, setDeleteTarget] = useState<User | null>(null)

  const usersQuery = useQuery({
    queryKey: ['users'],
    queryFn: api.usersList,
    retry: false,
  })

  const refreshUsers = async () => {
    await queryClient.invalidateQueries({ queryKey: ['users'] })
  }

  const handleSessionRevoked = async () => {
    await refreshUsers()
    await onSessionRevoked()
  }

  const createUserMutation = useMutation({
    mutationFn: api.createUser,
    onSuccess: async (result) => {
      setUsername('')
      setPassword('')
      setRole('operator')
      await refreshUsers()
      onToast('User created', `${result.user.username} is ready to sign in as ${result.user.role}.`, 'success')
    },
    onError: (error) => {
      onToast('User creation failed', formatErrorMessage(error, 'The user could not be created.'), 'error')
    },
  })

  const updateRoleMutation = useMutation({
    mutationFn: ({ id, role }: { id: string; role: string }) => api.updateUserRole(id, role),
    onSuccess: async (result) => {
      await refreshUsers()
      onToast('Role updated', `${result.user.username} is now ${result.user.role}.`, 'success')
      if (result.sessionRevoked) {
        await handleSessionRevoked()
      }
    },
    onError: (error) => {
      onToast('Role update failed', formatErrorMessage(error, 'The role could not be changed.'), 'error')
    },
  })

  const resetPasswordMutation = useMutation({
    mutationFn: ({ id, password }: { id: string; password: string }) => api.resetUserPassword(id, password),
    onSuccess: async (result, variables) => {
      setPasswordDrafts((current) => ({ ...current, [variables.id]: '' }))
      await refreshUsers()
      onToast('Password reset', `A new password is now active for ${result.user.username}.`, 'success')
      if (result.sessionRevoked) {
        await handleSessionRevoked()
      }
    },
    onError: (error) => {
      onToast('Password reset failed', formatErrorMessage(error, 'The password could not be reset.'), 'error')
    },
  })

  const deleteUserMutation = useMutation({
    mutationFn: api.deleteUser,
    onSuccess: async (result) => {
      const target = deleteTarget
      setDeleteTarget(null)
      await refreshUsers()
      onToast('User deleted', target ? `${target.username} was removed.` : 'The user was removed.', 'success')
      if (result.sessionRevoked) {
        await handleSessionRevoked()
      }
    },
    onError: (error) => {
      setDeleteTarget(null)
      onToast('Delete failed', formatErrorMessage(error, 'The user could not be deleted.'), 'error')
    },
  })

  const isForbidden = usersQuery.error instanceof APIRequestError && usersQuery.error.status === 403
  const users = usersQuery.data?.items ?? []
  const counts = useMemo(
    () => ({
      total: users.length,
      admins: users.filter((user) => user.role === 'admin').length,
    }),
    [users],
  )

  return (
    <>
      <header className="mb-8 flex flex-col gap-6 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <p className="text-xs uppercase tracking-[0.28em] text-base-content/50">Administration</p>
          <h2 className="page-title mt-1 text-3xl">Users</h2>
          <p className="mt-2 max-w-2xl text-sm text-base-content/65">
            Manage local accounts for this first admin-led rollout. Non-admin accounts can sign in, but broader read-only access is intentionally deferred.
          </p>
        </div>
        <div className="flex flex-wrap gap-2 text-sm text-base-content/70">
          <span className="rounded-full bg-base-300/60 px-3 py-1">Users: {counts.total}</span>
          <span className="rounded-full bg-base-300/60 px-3 py-1">Admins: {counts.admins}</span>
        </div>
      </header>

      {isForbidden ? (
        <InlineNotice tone="error" title="Access denied" detail="Administrator access is required to manage users." />
      ) : null}

      {!isForbidden && usersQuery.error ? (
        <InlineNotice
          tone="error"
          title="Users unavailable"
          detail={formatErrorMessage(usersQuery.error, 'User accounts could not be loaded.')}
        />
      ) : null}

      <section className="surface-card mb-6 border border-base-300/60 p-6 md:p-7">
        <div className="mb-5">
          <h3 className="section-title">Create user</h3>
          <p className="mt-1 text-sm text-base-content/60">Create a local account with a starting role and password.</p>
        </div>
        <form
          className="grid gap-4 md:grid-cols-[1.2fr_1.2fr_0.8fr_auto]"
          onSubmit={(event) => {
            event.preventDefault()
            createUserMutation.mutate({ username, password, role })
          }}
        >
          <label className="form-control">
            <span className="label-text mb-2 block text-sm font-medium">Username</span>
            <input className="input input-bordered w-full" value={username} onChange={(event) => setUsername(event.target.value)} />
          </label>
          <label className="form-control">
            <span className="label-text mb-2 block text-sm font-medium">Password</span>
            <input type="password" className="input input-bordered w-full" value={password} onChange={(event) => setPassword(event.target.value)} />
          </label>
          <label className="form-control">
            <span className="label-text mb-2 block text-sm font-medium">Role</span>
            <select className="select select-bordered w-full" value={role} onChange={(event) => setRole(event.target.value)}>
              {roleOptions.map((option) => (
                <option key={option} value={option}>
                  {option}
                </option>
              ))}
            </select>
          </label>
          <div className="flex items-end">
            <button className="btn btn-primary w-full md:w-auto" type="submit" disabled={createUserMutation.isPending}>
              {createUserMutation.isPending ? 'Creating...' : 'Create user'}
            </button>
          </div>
        </form>
      </section>

      <section className="surface-card border border-base-300/60 p-6 md:p-7">
        <div className="mb-5">
          <h3 className="section-title">Current users</h3>
          <p className="mt-1 text-sm text-base-content/60">Role changes and password resets revoke the target user&apos;s active sessions.</p>
        </div>

        {usersQuery.isLoading ? <LoadingState message="Loading users..." /> : null}
        {!usersQuery.isLoading && !users.length ? (
          <EmptyState title="No users found" description="Create another administrator, operator, or viewer to start testing role-aware auth." />
        ) : null}

        {users.length ? (
          <div className="space-y-4">
            {users.map((user) => {
              const roleDraft = roleDrafts[user.id] ?? user.role
              const passwordDraft = passwordDrafts[user.id] ?? ''
              const isSelf = user.id === currentUser.id

              return (
                <article key={user.id} className="rounded-2xl border border-base-300/60 bg-base-100/70 p-5">
                  <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
                    <div>
                      <h4 className="text-lg font-semibold text-base-content">{user.username}</h4>
                      <p className="mt-1 text-sm text-base-content/65">
                        Role: <span className="font-medium">{user.role}</span>
                        {isSelf ? ' · Current session' : ''}
                      </p>
                      <p className="mt-1 text-xs text-base-content/50">Created {new Date(user.createdAt).toLocaleString()}</p>
                    </div>

                    <div className="grid gap-3 md:grid-cols-[minmax(0,12rem)_auto] lg:min-w-[28rem]">
                      <label className="form-control">
                        <span className="label-text mb-2 block text-sm font-medium">Role</span>
                        <div className="flex gap-2">
                          <select
                            className="select select-bordered w-full"
                            value={roleDraft}
                            onChange={(event) => setRoleDrafts((current) => ({ ...current, [user.id]: event.target.value }))}
                          >
                            {roleOptions.map((option) => (
                              <option key={option} value={option}>
                                {option}
                              </option>
                            ))}
                          </select>
                          <button
                            className="btn btn-outline"
                            type="button"
                            disabled={updateRoleMutation.isPending || roleDraft === user.role}
                            onClick={() => updateRoleMutation.mutate({ id: user.id, role: roleDraft })}
                          >
                            Save role
                          </button>
                        </div>
                      </label>

                      <label className="form-control md:col-span-2">
                        <span className="label-text mb-2 block text-sm font-medium">Reset password</span>
                        <div className="flex gap-2">
                          <input
                            type="password"
                            className="input input-bordered w-full"
                            placeholder="Set a new password"
                            value={passwordDraft}
                            onChange={(event) => setPasswordDrafts((current) => ({ ...current, [user.id]: event.target.value }))}
                          />
                          <button
                            className="btn btn-outline"
                            type="button"
                            disabled={resetPasswordMutation.isPending || passwordDraft.trim() === ''}
                            onClick={() => resetPasswordMutation.mutate({ id: user.id, password: passwordDraft })}
                          >
                            Reset password
                          </button>
                        </div>
                      </label>

                      <div className="md:col-span-2 flex justify-end">
                        <button className="btn btn-error btn-outline" type="button" onClick={() => setDeleteTarget(user)}>
                          Delete user
                        </button>
                      </div>
                    </div>
                  </div>
                </article>
              )
            })}
          </div>
        ) : null}
      </section>

      <ConfirmDialog
        open={deleteTarget !== null}
        title="Delete user"
        description={deleteTarget ? `Delete ${deleteTarget.username}? Active sessions for that account will be revoked.` : 'Delete this user?'}
        confirmLabel="Delete"
        tone="danger"
        busy={deleteUserMutation.isPending}
        onConfirm={() => {
          if (deleteTarget) {
            deleteUserMutation.mutate(deleteTarget.id)
          }
        }}
        onCancel={() => setDeleteTarget(null)}
      />
    </>
  )
}
