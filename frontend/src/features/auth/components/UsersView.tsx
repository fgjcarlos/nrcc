import { useState } from 'react';
import { type User } from '@/features/auth/services/authService';
import { useAuth } from '@/features/auth/hooks/useAuth';
import { useUsersData } from '@/features/auth/hooks/useUsersData';
import { useUsersActions } from '@/features/auth/hooks/useUsersActions';

export function UsersView() {
  const { user: currentUser } = useAuth();
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [editingUser, setEditingUser] = useState<User | null>(null);
  const [formData, setFormData] = useState({
    username: '',
    password: '',
    role: 'viewer' as 'admin' | 'viewer',
  });

  const { users, isLoading } = useUsersData({
    enabled: currentUser?.role === 'admin',
  });

  const { createMutation, deleteMutation, changePasswordMutation } = useUsersActions();

  const openModal = (user?: User) => {
    if (user) {
      setEditingUser(user);
      setFormData({ username: user.username, password: '', role: user.role });
    } else {
      setEditingUser(null);
      setFormData({ username: '', password: '', role: 'viewer' });
    }
    setIsModalOpen(true);
  };

  const closeModal = () => {
    setIsModalOpen(false);
    setEditingUser(null);
    setFormData({ username: '', password: '', role: 'viewer' });
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (editingUser) {
      if (formData.password) {
        changePasswordMutation.mutate({ id: editingUser.id, password: formData.password });
        closeModal();
      }
    } else {
      createMutation.mutate(formData);
      closeModal();
    }
  };

  const handleDelete = (id: string) => {
    if (confirm('Are you sure you want to delete this user?')) {
      deleteMutation.mutate(id);
    }
  };

  if (currentUser?.role !== 'admin') {
    return (
      <div className="p-6">
        <h1 className="text-2xl font-bold text-base-content">Access Denied</h1>
        <p className="mt-2 text-base-content/60">You don't have permission to view this page.</p>
      </div>
    );
  }

  return (
    <div className="p-6 space-y-6">
      <div className="flex justify-between items-center">
        <h1 className="text-2xl font-bold text-base-content">User Management</h1>
        <button
          onClick={() => openModal()}
          className="action-btn-primary"
        >
          Add User
        </button>
      </div>

      {isLoading ? (
        <div className="flex justify-center py-8">
          <div className="h-8 w-8 animate-spin rounded-full border-b-2 border-primary"></div>
        </div>
      ) : (
        <div className="surface-card overflow-hidden">
          <table className="w-full">
            <thead className="table-header-subtle">
              <tr>
                <th className="px-4 py-3 text-left text-sm font-medium text-base-content">Username</th>
                <th className="px-4 py-3 text-left text-sm font-medium text-base-content">Role</th>
                <th className="px-4 py-3 text-left text-sm font-medium text-base-content">Created</th>
                <th className="px-4 py-3 text-right text-sm font-medium text-base-content">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border">
              {users.map((user) => (
                <tr key={user.id} className="table-row-hover">
                  <td className="px-4 py-3 text-base-content">{user.username}</td>
                  <td className="px-4 py-3">
                    <span
                      className={`rounded-full px-2 py-1 text-xs ${
                        user.role === 'admin'
                          ? 'bg-info/15 text-info-content'
                          : 'bg-base-300/70 text-base-content/70'
                      }`}
                    >
                      {user.role}
                    </span>
                  </td>
                  <td className="px-4 py-3 text-sm text-base-content/60">
                    {new Date(user.createdAt).toLocaleDateString()}
                  </td>
                  <td className="px-4 py-3 text-right space-x-2">
                    <button
                      onClick={() => openModal(user)}
                      className="text-sm text-primary transition-colors hover:text-primary/80"
                    >
                      Change Password
                    </button>
                    {user.id !== currentUser?.id && (
                      <button
                        onClick={() => handleDelete(user.id)}
                        className="text-sm text-rose-300 transition-colors hover:text-rose-200"
                      >
                        Delete
                      </button>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {/* Modal */}
      {isModalOpen && (
        <div className="modal-overlay">
          <div className="surface-panel w-full max-w-md border border-border p-6 shadow-glow">
            <h2 className="mb-4 text-xl font-bold text-base-content">
              {editingUser ? 'Change Password' : 'Add User'}
            </h2>
            <form onSubmit={handleSubmit} className="space-y-4">
              {!editingUser && (
                <>
                  <div>
                    <label className="mb-1 block text-sm font-medium text-base-content">
                      Username
                    </label>
                    <input
                      type="text"
                      value={formData.username}
                      onChange={(e) => setFormData({ ...formData, username: e.target.value })}
                      className="glass-panel w-full rounded-xl border border-border px-3 py-2 text-base-content focus:outline-none focus:ring-2 focus:ring-primary/50"
                      required
                    />
                  </div>
                  <div>
                    <label className="mb-1 block text-sm font-medium text-base-content">
                      Role
                    </label>
                    <select
                      value={formData.role}
                      onChange={(e) => setFormData({ ...formData, role: e.target.value as 'admin' | 'viewer' })}
                      className="glass-panel w-full rounded-xl border border-border px-3 py-2 text-base-content focus:outline-none focus:ring-2 focus:ring-primary/50"
                    >
                      <option value="viewer">Viewer</option>
                      <option value="admin">Admin</option>
                    </select>
                  </div>
                </>
              )}
              <div>
                <label className="mb-1 block text-sm font-medium text-base-content">
                  {editingUser ? 'New Password' : 'Password'}
                </label>
                <input
                  type="password"
                  value={formData.password}
                  onChange={(e) => setFormData({ ...formData, password: e.target.value })}
                  className="glass-panel w-full rounded-xl border border-border px-3 py-2 text-base-content focus:outline-none focus:ring-2 focus:ring-primary/50"
                  required={!editingUser}
                  minLength={8}
                />
              </div>
              <div className="flex justify-end space-x-2">
                <button
                  type="button"
                  onClick={closeModal}
                  className="action-btn-secondary"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  className="action-btn-primary"
                >
                  {editingUser ? 'Change Password' : 'Create User'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
