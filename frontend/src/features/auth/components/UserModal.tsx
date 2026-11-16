import { useState, useEffect } from 'react';
import { type User } from '@/features/auth/services/authService';
import { useT } from '@/i18n';
import { passwordSchema } from '@/shared/validation/schemas';
import { Button, Dialog, DialogBody, DialogFooter, Input, Select } from '@/shared/components/ui';

type ModalMode = 'create' | 'edit_full' | 'edit_password';

interface UserModalProps {
  mode: ModalMode;
  editingUser: User | null;
  isPending: boolean;
  adminCount: number;
  currentUserId: string;
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
  currentUserId,
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
  const isCurrentUser = editingUser?.id === currentUserId;
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
    const result = passwordSchema.safeParse(pw);
    if (!result.success) {
      setPasswordError(result.error.issues[0]?.message ?? 'Invalid password');
      return false;
    }
    setPasswordError('');
    return true;
  };

  const { t } = useT();


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
    if (mode === 'create') return t('auth:createUser');
    if (mode === 'edit_full') return t('auth:editUser');
    return t('auth:changePassword');
  };

  return (
    <Dialog open onClose={onClose} title={getTitle()} dismissible={!isPending}>
      <form onSubmit={handleSubmit} className="flex min-h-0 flex-1 flex-col">
        <DialogBody className="space-y-4">
          {/* Username and role: create and edit_full modes */}
          {(mode === 'create' || mode === 'edit_full') && (
            <>
              <Input
                label={t('auth:usernameLabel')}
                type="text"
                value={formData.username}
                onChange={(e) => setFormData({ ...formData, username: e.target.value })}
                disabled={mode === 'edit_full'}
                required={mode === 'create'}
              />

              <div className="space-y-1">
                <Select
                  label={t('auth:roleLabel')}
                  value={formData.role}
                  onChange={(e) => setFormData({ ...formData, role: e.target.value as 'admin' | 'viewer' })}
                  disabled={isLastAdmin || isCurrentUser}
                  required={mode === 'create'}
                >
                  <option value="viewer">{t('auth:roles.viewer')}</option>
                  <option value="admin">{t('auth:roles.admin')}</option>
                </Select>
                {isLastAdmin && (
                  <p className="text-xs text-ds-on-tint-error">{t('auth:cannotDemoteLastAdmin')}</p>
                )}
                {isCurrentUser && (
                  <p className="text-xs text-ds-text-secondary">{t('auth:cannotChangeOwnRole')}</p>
                )}
              </div>
            </>
          )}

          {/* Password: every mode */}
          <Input
            label={mode === 'create' ? t('auth:passwordLabel') : t('auth:newPasswordLabel')}
            type="password"
            value={formData.password}
            onChange={(e) => {
              setFormData({ ...formData, password: e.target.value });
              if (passwordError) setPasswordError('');
            }}
            error={passwordError || undefined}
            required={mode === 'create' || mode === 'edit_password'}
            minLength={8}
            placeholder={mode === 'edit_full' ? t('auth:keepCurrentPasswordPlaceholder') : undefined}
          />
        </DialogBody>

        <DialogFooter>
          <Button type="button" variant="secondary" onClick={onClose} disabled={isPending}>
            {t('common:cancel')}
          </Button>
          <Button type="submit" variant="solid" disabled={!canSubmit}>
            {isPending && (
              <span
                className="h-4 w-4 animate-spin rounded-full border-b-2 border-current motion-reduce:animate-none"
                aria-hidden="true"
              />
            )}
            {mode === 'create' ? t('auth:createUser') : t('common:confirm')}
          </Button>
        </DialogFooter>
      </form>
    </Dialog>
  );
}
