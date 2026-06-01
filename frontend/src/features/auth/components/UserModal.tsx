import { useState, useEffect } from 'react';
import { type User } from '@/features/auth/services/authService';
import { UI_COPY } from '@/shared/constants/uiCopy';

type ModalMode = 'create' | 'edit_full' | 'edit_password';

interface UserModalProps {
  mode: ModalMode;
  editingUser: User | null;
  isPending: boolean;
  adminCount: number;
  onSubmit: (data: {
    username?: string;
    password?: string;
    role?: 'admin' | 'viewer';
  }) => void;
  onClose: () => void;
}

export function UserModal({
  mode,
  editingUser,
  isPending,
  adminCount,
  onSubmit,
  onClose,
}: UserModalProps) {
  const [formData, setFormData] = useState({
    username: '',
    password: '',
    role: 'viewer' as 'admin' | 'viewer',
  });
  const [passwordError, setPasswordError] = useState('');

  // Pre-fill form data when modal opens
  useEffect(() => {
    if (mode === 'create') {
      setFormData({ username: '', password: '', role: 'viewer' });
    } else if (editingUser) {
      setFormData({
        username: editingUser.username,
        password: '',
        role: editingUser.role,
      });
    }
  }, [mode, editingUser]);

  const isLastAdmin = editingUser?.role === 'admin' && adminCount === 1;
  const canSubmit = !isPending && (
    mode === 'create'
      ? formData.username.trim() && formData.password.trim()
      : mode === 'edit_full'
      ? true // can change role or leave as-is
      : formData.password.trim() // edit_password requires password
    // In edit_full the last admin cannot submit (their role select is disabled
    // and password changes go through the dedicated change-password modal).
  ) && (mode !== 'edit_full' || !isLastAdmin);

  const validatePassword = (pw: string): boolean => {
    if (pw.length < 8) {
      setPasswordError('Password must be at least 8 characters');
      return false;
    }
    setPasswordError('');
    return true;
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();

    const pw = formData.password.trim();
    if ((mode === 'create' || mode === 'edit_password' || (mode === 'edit_full' && pw)) && !validatePassword(pw)) {
      return;
    }

    const submitData: {
      username?: string;
      password?: string;
      role?: 'admin' | 'viewer';
    } = {};

    if (mode === 'create') {
      submitData.username = formData.username.trim();
      submitData.password = formData.password.trim();
      submitData.role = formData.role;
    } else if (mode === 'edit_full') {
      if (formData.role !== editingUser?.role) {
        submitData.role = formData.role;
      }
      if (formData.password.trim()) {
        submitData.password = formData.password.trim();
      }
    } else if (mode === 'edit_password') {
      submitData.password = formData.password.trim();
    }

    onSubmit(submitData);
  };

  const getTitle = () => {
    if (mode === 'create') return UI_COPY.createUser;
    if (mode === 'edit_full') return UI_COPY.editUser;
    return UI_COPY.changePassword;
  };

  return (
    <div className="modal-overlay" onClick={(e) => {
      // Only close if clicking the overlay directly (not the modal)
      if (e.target === e.currentTarget) {
        onClose();
      }
    }}>
      <div className="surface-panel w-full max-w-md border border-border p-6 shadow-glow">
        <h2 className="mb-4 text-xl font-bold text-base-content">{getTitle()}</h2>
        
        <form onSubmit={handleSubmit} className="space-y-4">
          {/* Username field - visible in create and edit_full modes */}
          {(mode === 'create' || mode === 'edit_full') && (
            <div>
              <label className="mb-1 block text-sm font-medium text-base-content">
                {UI_COPY.usernameLabel}
              </label>
              <input
                type="text"
                value={formData.username}
                onChange={(e) => setFormData({ ...formData, username: e.target.value })}
                disabled={mode === 'edit_full'}
                className={`glass-panel w-full rounded-xl border border-border px-3 py-2 text-base-content focus:outline-none focus:ring-2 focus:ring-primary/50 ${
                  mode === 'edit_full' ? 'opacity-60 cursor-not-allowed' : ''
                }`}
                required={mode === 'create'}
              />
            </div>
          )}

          {/* Role selector - visible in create and edit_full modes */}
          {(mode === 'create' || mode === 'edit_full') && (
            <div>
              <label className="mb-1 block text-sm font-medium text-base-content">
                {UI_COPY.roleLabel}
              </label>
              <select
                value={formData.role}
                onChange={(e) => setFormData({ ...formData, role: e.target.value as 'admin' | 'viewer' })}
                disabled={isLastAdmin}
                className={`glass-panel w-full rounded-xl border border-border px-3 py-2 text-base-content focus:outline-none focus:ring-2 focus:ring-primary/50 ${
                  isLastAdmin ? 'opacity-60 cursor-not-allowed' : ''
                }`}
                required={mode === 'create'}
              >
                <option value="viewer">Viewer</option>
                <option value="admin">Admin</option>
              </select>
              {isLastAdmin && (
                <p className="mt-2 text-sm text-error">{UI_COPY.cannotDemoteLastAdmin}</p>
              )}
            </div>
          )}

          {/* Password field - visible in all modes */}
          <div>
            <label className="mb-1 block text-sm font-medium text-base-content">
              {mode === 'create' ? UI_COPY.passwordLabel : UI_COPY.newPasswordLabel}
            </label>
            <input
              type="password"
              value={formData.password}
              onChange={(e) => {
                setFormData({ ...formData, password: e.target.value });
                if (passwordError) setPasswordError('');
              }}
              className="glass-panel w-full rounded-xl border border-border px-3 py-2 text-base-content focus:outline-none focus:ring-2 focus:ring-primary/50"
              required={mode === 'create' || mode === 'edit_password'}
              minLength={8}
              placeholder={mode === 'edit_full' ? 'Leave empty to keep current password' : undefined}
            />
            {passwordError && (
              <p className="mt-1 text-sm text-error">{passwordError}</p>
            )}
          </div>

          <div className="flex justify-end space-x-2 pt-4">
            <button
              type="button"
              onClick={onClose}
              disabled={isPending}
              className="action-btn-secondary disabled:opacity-60 disabled:cursor-not-allowed"
            >
              {UI_COPY.cancel}
            </button>
            <button
              type="submit"
              disabled={!canSubmit}
              className="action-btn-primary disabled:opacity-60 disabled:cursor-not-allowed flex items-center gap-2"
            >
              {isPending && (
                <div className="h-4 w-4 animate-spin rounded-full border-b-2 border-current"></div>
              )}
              {mode === 'create' ? UI_COPY.createUser : UI_COPY.confirm}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
