import { type User } from '@/features/auth/services/authService';
import { useT } from '@/i18n';
import { StatusChip, Table } from '@/shared/components/ui';

interface UserTableProps {
  users: User[];
  adminCount: number;
  currentUserId: string;
  onEdit: (user: User) => void;
  onDelete: (user: User) => void;
  onChangePassword: (user: User) => void;
}

export function UserTable({
  users,
  adminCount,
  currentUserId,
  onEdit,
  onDelete,
  onChangePassword,
}: UserTableProps) {
  const { t } = useT();
  // 44px touch targets on phones, dense 24px links from sm up.
  const linkButton = 'min-h-11 text-sm transition-colors hover:underline md:min-h-6';
  return (
    // One table; below sm each user becomes a labelled card.
    <Table stacked>
      <thead>
        <tr>
          <th>{t('auth:usernameLabel')}</th>
          <th>{t('auth:roleLabel')}</th>
          <th>{t('auth:createdLabel')}</th>
          <th className="text-right">{t('auth:actionsLabel')}</th>
        </tr>
      </thead>
      <tbody>
        {users.map((user) => {
          // Explains the disabled Delete; the guard itself stays on `disabled` below.
          const deleteBlockedReason =
            user.id === currentUserId
              ? t('auth:cannotDeleteSelf')
              : user.role === 'admin' && adminCount === 1
                ? t('auth:cannotDeleteLastAdmin')
                : null;
          const reasonId = `user-delete-reason-${user.id}`;
          return (
            <tr key={user.id}>
              <td data-label={t('auth:usernameLabel')} className="font-medium">{user.username}</td>
              <td data-label={t('auth:roleLabel')}>
                <StatusChip variant="neutral" size="sm">
                  {user.role === 'admin' ? t('auth:roles.admin') : t('auth:roles.viewer')}
                </StatusChip>
              </td>
              <td data-label={t('auth:createdLabel')} className="text-ds-text-secondary">
                {new Date(user.createdAt).toLocaleDateString()}
              </td>
              <td data-label={t('auth:actionsLabel')} className="text-right">
                <div className="space-y-0.5">
                  <div className="flex flex-wrap justify-end gap-x-3">
                    <button onClick={() => onEdit(user)} className={`${linkButton} text-ds-on-tint-primary`}>
                      {t('auth:editUser')}
                    </button>
                    <button onClick={() => onChangePassword(user)} className={`${linkButton} text-ds-on-tint-primary`}>
                      {t('auth:changePassword')}
                    </button>
                    <button
                      onClick={() => onDelete(user)}
                      disabled={user.id === currentUserId || (user.role === 'admin' && adminCount === 1)}
                      aria-describedby={deleteBlockedReason ? reasonId : undefined}
                      className={`${linkButton} text-ds-on-tint-error disabled:cursor-not-allowed disabled:text-base-content/40`} // contrast-ok: disabled control, WCAG 1.4.3 exempts inactive UI
                    >
                      {t('auth:deleteAction')}
                    </button>
                  </div>
                  {deleteBlockedReason && (
                    <p id={reasonId} className="text-start text-xs text-ds-text-secondary sm:text-end">
                      {deleteBlockedReason}
                    </p>
                  )}
                </div>
              </td>
            </tr>
          );
        })}
      </tbody>
    </Table>
  );
}
