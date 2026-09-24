import { type User } from '@/features/auth/services/authService';
import { useT } from '@/i18n';

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
  return (
    <>
      {/* Desktop table (md+) */}
      <div className="hidden md:block surface-card overflow-hidden">
        <table className="w-full">
          <thead className="table-header-subtle">
            <tr>
              <th className="px-4 py-3 text-left text-sm font-medium text-base-content">
                {t('auth:usernameLabel')}
              </th>
              <th className="px-4 py-3 text-left text-sm font-medium text-base-content">
                {t('auth:roleLabel')}
              </th>
              <th className="px-4 py-3 text-left text-sm font-medium text-base-content">
                {t('auth:createdLabel')}
              </th>
              <th className="px-4 py-3 text-right text-sm font-medium text-base-content">
                {t('backups:actions')}
              </th>
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
                    onClick={() => onEdit(user)}
                    className="text-sm text-primary transition-colors hover:text-primary/80"
                  >
                    {t('auth:editUser')}
                  </button>
                  <button
                    onClick={() => onChangePassword(user)}
                    className="text-sm text-primary transition-colors hover:text-primary/80"
                  >
                    {t('auth:changePassword')}
                  </button>
                  <button
                    onClick={() => onDelete(user)}
                    disabled={user.id === currentUserId || (user.role === 'admin' && adminCount === 1)}
                    className="text-sm text-error transition-colors hover:text-error/80 disabled:text-base-content/40 disabled:cursor-not-allowed"
                  >
                    {t('common:delete')}
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {/* Mobile cards (< md) */}
      <div className="md:hidden space-y-4">
        {users.map((user) => (
          <div key={user.id} className="surface-card rounded-lg p-4 border border-border">
            <div className="space-y-3">
              <div>
                <p className="text-xs text-base-content/60 uppercase tracking-wider">
                  {t('auth:usernameLabel')}
                </p>
                <p className="text-base font-medium text-base-content">{user.username}</p>
              </div>
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-xs text-base-content/60 uppercase tracking-wider">
                    {t('auth:roleLabel')}
                  </p>
                  <span
                    className={`inline-block rounded-full px-2 py-1 text-xs ${
                      user.role === 'admin'
                        ? 'bg-info/15 text-info-content'
                        : 'bg-base-300/70 text-base-content/70'
                    }`}
                  >
                    {user.role}
                  </span>
                </div>
                <div>
                  <p className="text-xs text-base-content/60 uppercase tracking-wider">
                    {t('auth:createdLabel')}
                  </p>
                  <p className="text-sm text-base-content/60">
                    {new Date(user.createdAt).toLocaleDateString()}
                  </p>
                </div>
              </div>
              <div className="flex gap-2 pt-2">
                <button
                  onClick={() => onEdit(user)}
                  className="flex-1 text-sm text-primary transition-colors hover:text-primary/80"
                >
                  {t('auth:editUser')}
                </button>
                <button
                  onClick={() => onChangePassword(user)}
                  className="flex-1 text-sm text-primary transition-colors hover:text-primary/80"
                >
                  {t('auth:changePassword')}
                </button>
                <button
                  onClick={() => onDelete(user)}
                  disabled={user.id === currentUserId || (user.role === 'admin' && adminCount === 1)}
                  className="flex-1 text-sm text-error transition-colors hover:text-error/80 disabled:text-base-content/40 disabled:cursor-not-allowed"
                >
                  {t('common:delete')}
                </button>
              </div>
            </div>
          </div>
        ))}
      </div>
    </>
  );
}
