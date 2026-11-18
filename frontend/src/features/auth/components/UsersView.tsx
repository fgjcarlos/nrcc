import { useState } from 'react';
import { Plus } from 'lucide-react';
import { type User } from '@/features/auth/services/authService';
import { useAuth } from '@/features/auth/hooks/useAuth';
import { useUsersData } from '@/features/auth/hooks/useUsersData';
import { useUsersActions } from '@/features/auth/hooks/useUsersActions';
import { UserTable } from '@/features/auth/components/UserTable';
import { UserModal } from '@/features/auth/components/UserModal';
import { StateContainer } from '@/shared/components/StateContainer';
import { ConfirmationDialog } from '@/shared/components/ConfirmationDialog';
import { useConfirmationDialog } from '@/shared/hooks/useConfirmationDialog';
import { Button } from '@/shared/components/ui';
import { useT } from '@/i18n';

type ModalMode = 'create' | 'edit_full' | 'edit_password';

interface ModalState {
  mode: ModalMode;
  editingUser: User | null;
}

export function UsersView() {
  const { user: currentUser } = useAuth();
  const [modalState, setModalState] = useState<ModalState | null>(null);

  const { users, isLoading, isError } = useUsersData({
    enabled: currentUser?.role === 'admin',
  });

  const { createMutation, deleteMutation, changePasswordMutation, updateRoleMutation } = useUsersActions();
  const { t } = useT();

  const deleteDialog = useConfirmationDialog<User>();

  // Count admins in current user list
  const adminCount = users.filter((u) => u.role === 'admin').length;

  // Modal handlers
  const openCreateModal = () => {
    setModalState({ mode: 'create', editingUser: null });
  };

  const openEditModal = (user: User) => {
    setModalState({ mode: 'edit_full', editingUser: user });
  };

  const openPasswordModal = (user: User) => {
    setModalState({ mode: 'edit_password', editingUser: user });
  };

  const closeModal = () => {
    setModalState(null);
  };

  // Form submission handlers
  const handleCreateSubmit = (data: {
    username?: string;
    password?: string;
    role?: 'admin' | 'viewer';
  }) => {
    createMutation.mutate(
      {
        username: data.username!,
        password: data.password!,
        role: data.role!,
      },
      {
        onSuccess: closeModal,
      }
    );
  };

  const handleEditSubmit = (data: {
    username?: string;
    password?: string;
    role?: 'admin' | 'viewer';
  }) => {
    if (!modalState?.editingUser) return;

    // If role changed, mutate it
    if (data.role && data.role !== modalState.editingUser.role) {
      updateRoleMutation.mutate(
        { id: modalState.editingUser.id, role: data.role },
        {
          onSuccess: closeModal,
        }
      );
    } else if (data.password) {
      // If only password changed (no role change)
      changePasswordMutation.mutate(
        { id: modalState.editingUser.id, password: data.password },
        {
          onSuccess: closeModal,
        }
      );
    } else {
      // No changes
      closeModal();
    }
  };

  const handlePasswordSubmit = (data: {
    username?: string;
    password?: string;
    role?: 'admin' | 'viewer';
  }) => {
    if (!modalState?.editingUser) return;
    changePasswordMutation.mutate(
      { id: modalState.editingUser.id, password: data.password! },
      {
        onSuccess: closeModal,
      }
    );
  };

  const handleModalSubmit = (data: {
    username?: string;
    password?: string;
    role?: 'admin' | 'viewer';
  }) => {
    if (!modalState) return;
    if (modalState.mode === 'create') {
      handleCreateSubmit(data);
    } else if (modalState.mode === 'edit_full') {
      handleEditSubmit(data);
    } else if (modalState.mode === 'edit_password') {
      handlePasswordSubmit(data);
    }
  };

  // Delete handler
  const handleDelete = (user: User) => {
    deleteDialog.open(user);
  };

  const handleDeleteConfirm = () => {
    if (deleteDialog.pendingItem) {
      deleteMutation.mutate(deleteDialog.pendingItem.id, {
        onSuccess: deleteDialog.close,
      });
    }
  };

  // Access control
  if (currentUser?.role !== 'admin') {
    return (
      <div className="space-y-1">
        <h1 className="text-xl font-bold text-ds-text-primary">{t('auth:accessDenied')}</h1>
        <p className="text-sm text-ds-text-secondary">{t('auth:accessDeniedDescription')}</p>
      </div>
    );
  }

  // Check if any mutation is pending (for disabling buttons)
  const isAnyMutationPending =
    createMutation.isPending ||
    deleteMutation.isPending ||
    changePasswordMutation.isPending ||
    updateRoleMutation.isPending;

  return (
    <div className="space-y-4">
      <header className="flex flex-col gap-3 border-b border-ds-border-subtle pb-3 sm:flex-row sm:items-end sm:justify-between">
        <div className="min-w-0 space-y-1">
          <h1 className="text-xl font-bold text-ds-text-primary">{t('auth:userManagement')}</h1>
          <p className="max-w-[75ch] text-pretty text-sm text-ds-text-secondary">{t('auth:userManagementSafetyNotice')}</p>
        </div>
        <Button
          type="button"
          onClick={openCreateModal}
          disabled={isAnyMutationPending}
          className="min-h-11 shrink-0 self-start md:min-h-9 sm:self-auto"
        >
          <Plus className="h-4 w-4" aria-hidden="true" />
          {t('auth:addUser')}
        </Button>
      </header>

      {/* State container: loading, error, empty, or content */}
      <StateContainer
        isLoading={isLoading}
        isError={isError}
        isEmpty={users.length === 0}
        emptySlot={
          <div className="flex flex-col items-center justify-center gap-1 py-12 text-sm">
            <p className="font-semibold text-ds-text-primary">{t('auth:noUsersYet')}</p>
            <p className="text-ds-text-secondary">{t('auth:addFirstUser')}</p>
          </div>
        }
      >
        <UserTable
          users={users}
          adminCount={adminCount}
          currentUserId={currentUser.id}
          onEdit={openEditModal}
          onDelete={handleDelete}
          onChangePassword={openPasswordModal}
        />
      </StateContainer>

      {/* Delete Confirmation Dialog */}
      <ConfirmationDialog
        isOpen={deleteDialog.isOpen}
        title={deleteDialog.pendingItem ? t('auth:deleteUserTitle', { username: deleteDialog.pendingItem.username }) : ''}
        description={
          deleteDialog.pendingItem
            ? t('auth:deleteUserDescription', { username: deleteDialog.pendingItem.username })
            : ''
        }
        variant="danger"
        isPending={deleteMutation.isPending}
        onConfirm={handleDeleteConfirm}
        onCancel={deleteDialog.close}
      />

      {/* Modal */}
      {modalState && (
        <UserModal
          mode={modalState.mode}
          editingUser={modalState.editingUser}
          isPending={
            modalState.mode === 'create'
              ? createMutation.isPending
              : modalState.mode === 'edit_full'
              ? updateRoleMutation.isPending || changePasswordMutation.isPending
              : changePasswordMutation.isPending
          }
          adminCount={adminCount}
          currentUserId={currentUser.id}
          onSubmit={handleModalSubmit}
          onClose={closeModal}
        />
      )}
    </div>
  );
}
