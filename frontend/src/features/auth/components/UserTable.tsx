import { type User } from '@/features/auth/services/authService';
import { useT } from '@/i18n';
import { Table } from '@/shared/components/ui';

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
  const linkButton = 'min-h-6 text-sm text-ds-on-tint-primary transition-colors hover:underline';
  return (
    // One table; below sm each user becomes a labelled card.
    <Table stacked>
      <thead>
        <tr>
          <th>{t('auth:usernameLabel')}</th>
          <th>{t('auth:roleLabel')}</th>
          <th>{t('auth:createdLabel')}</th>
          <th className="text-right">{t('backups:table.actions')}</th>
        </tr>
      </thead>
      <tbody>
        {users.map((user) => (
          <tr key={user.id}>
            <td data-label={t('auth:usernameLabel')} className="font-medium">{user.username}</td>
            <td data-label={t('auth:roleLabel')}>
              <span
                className={`rounded-full px-2 py-1 text-xs ${
                  user.role === 'admin'
                    ? 'bg-info/15 text-ds-on-tint-info'
                    : 'bg-base-300/70 text-base-content/70'
                }`}
              >
                {user.role}
              </span>
            </td>
            <td data-label={t('auth:createdLabel')} className="text-ds-text-secondary">
              {new Date(user.createdAt).toLocaleDateString()}
            </td>
            <td data-label={t('backups:table.actions')} className="text-right">
              <div className="flex flex-wrap justify-end gap-x-3 gap-y-1">
                <button onClick={() => onEdit(user)} className={linkButton}>
                  {t('auth:editUser')}
                </button>
                <button onClick={() => onChangePassword(user)} className={linkButton}>
                  {t('auth:changePassword')}
                </button>
                <button
                  onClick={() => onDelete(user)}
                  disabled={user.id === currentUserId || (user.role === 'admin' && adminCount === 1)}
                  className="min-h-6 text-sm text-ds-on-tint-error transition-colors hover:underline disabled:cursor-not-allowed disabled:text-base-content/40"
                >
                  {t('common:delete')}
                </button>
              </div>
            </td>
          </tr>
        ))}
      </tbody>
    </Table>
  );
}
