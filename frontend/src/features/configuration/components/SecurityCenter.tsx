import { useEffect, useState } from 'react';
import { AxiosError } from 'axios';
import { toast } from 'sonner';
import { ConfirmationDialog } from '@/shared/components/ConfirmationDialog';
import { configService } from '../services';
import type { NodeRedConfigResponse } from '../lib/configTransformers';

type AdminUser = { username: string; permissions: '*' | 'read'; password: string };
type BasicAuth = { user: string; pass: string; storedPass?: string };

interface Props {
  config: NodeRedConfigResponse | null;
  rawSettingsContent: string;
  expectedRevision?: string;
  editable: boolean;
  onApplied: () => void;
}

const emptyUser = (): AdminUser => ({ username: '', permissions: '*', password: '' });
const aliases = (source: string) => ['nodeHttpAuth', 'staticAuth'].filter((name) => new RegExp(`\\b${name}\\s*:`).test(source));

export function SecurityCenter({ config, rawSettingsContent, expectedRevision, editable, onApplied }: Props) {
  const [users, setUsers] = useState<AdminUser[]>([]);
  const [expiry, setExpiry] = useState(0);
  const [node, setNode] = useState<BasicAuth>({ user: '', pass: '' });
  const [staticAuth, setStaticAuth] = useState<BasicAuth>({ user: '', pass: '' });
  const [saving, setSaving] = useState(false);
  const [confirmMigration, setConfirmMigration] = useState(false);
  const [applyStatus, setApplyStatus] = useState<string>();
  const [changed, setChanged] = useState({ admin: false, node: false, static: false });
  const legacyAliases = aliases(rawSettingsContent);

  useEffect(() => {
    const admin = config?.adminAuth as { users?: Array<{ username?: string; permissions?: '*' | 'read' }>; sessionExpiryTime?: number } | undefined;
    const nodeAuth = config?.httpNodeAuth as Partial<BasicAuth> | undefined;
    const staticSurface = config?.httpStaticAuth as Partial<BasicAuth> | undefined;
    setUsers(admin?.users?.map((user) => ({ username: user.username ?? '', permissions: user.permissions ?? '*', password: '' })) ?? []);
    setExpiry(admin?.sessionExpiryTime ?? 0);
    setNode({ user: nodeAuth?.user ?? '', pass: '', storedPass: nodeAuth?.pass });
    setStaticAuth({ user: staticSurface?.user ?? '', pass: '', storedPass: staticSurface?.pass });
    setChanged({ admin: false, node: false, static: false });
  }, [config]);

  const payload = () => {
    const { adminAuth: _admin, httpNodeAuth: _node, httpStaticAuth: _static, ...preserved } = config ?? {};
    return {
    ...preserved,
    ...((changed.admin || legacyAliases.length) ? { adminAuth: users.length ? { type: 'credentials', users: users.map(({ username, permissions, password }) => ({ username, permissions, password })) , ...(expiry > 0 ? { sessionExpiryTime: expiry } : {}) } : null } : {}),
    ...((changed.node || legacyAliases.length) ? { httpNodeAuth: node.user ? { user: node.user, pass: node.pass || node.storedPass || '' } : null } : {}),
    ...((changed.static || legacyAliases.length) ? { httpStaticAuth: staticAuth.user ? { user: staticAuth.user, pass: staticAuth.pass || staticAuth.storedPass || '' } : null } : {}),
    };
  };
  const save = async () => {
    if (legacyAliases.length) return setConfirmMigration(true);
    await apply();
  };
  const apply = async () => {
    if ([node.pass, staticAuth.pass].some((pass) => pass && !/^\$2[aby]\$/.test(pass))) {
      toast.error('HTTP authentication replacements must be bcrypt hashes');
      return;
    }
    if (!expectedRevision) {
      setApplyStatus('The settings revision is unavailable. Refresh before retrying the transaction.');
      return;
    }
    setConfirmMigration(false);
    setApplyStatus(undefined);
    setSaving(true);
    try {
      await configService.applyConfig(payload(), expectedRevision);
      setChanged({ admin: false, node: false, static: false });
      onApplied();
      toast.success('Security Center settings applied');
    } catch (error) {
      const code = error instanceof AxiosError ? error.response?.data?.error?.code : undefined;
      const message = code === 'SETTINGS_REVISION_CONFLICT'
        ? 'Settings changed elsewhere. Refresh, review the redacted preview, then retry.'
        : code === 'APPLY_IN_FLIGHT'
          ? 'Another transaction is in progress. Wait for it to finish, then retry.'
          : 'The transaction did not complete. Any failed readiness check is rolled back; review runtime readiness, then retry.';
      setApplyStatus(message);
      toast.error('Security Center transaction needs attention');
    } finally {
      setSaving(false);
    }
  };
  const updateUser = (index: number, update: Partial<AdminUser>) => { setChanged((current) => ({ ...current, admin: true })); setUsers((current) => current.map((user, i) => i === index ? { ...user, ...update } : user)); };
  const disabled = !editable || saving;

  return <section aria-labelledby="security-center-title" className="space-y-6">
    <div><h2 id="security-center-title" className="text-lg font-medium">Security Center</h2><p className="text-sm text-base-content/65">Passwords and bcrypt hashes are never shown. Leave a password blank to preserve its existing value.</p></div>
    <aside aria-label="Redacted transaction preview" className="rounded-xl border border-border bg-base-200/35 p-4 text-sm"><h3 className="font-medium">Redacted transaction preview</h3><p>Canonical surfaces: adminAuth, httpNodeAuth, and httpStaticAuth. Credential values, hashes, and passwords are redacted.</p></aside>
    {applyStatus && <p role="status" aria-live="polite" className="rounded-xl border border-warning/40 bg-warning/10 p-4 text-sm text-warning">{applyStatus}</p>}
    {!editable && <p role="status" className="rounded-xl border border-warning/40 bg-warning/10 p-4 text-sm text-warning">Security Center is read-only because this runtime configuration is not editable.</p>}
    {editable && legacyAliases.length > 0 && <div className="rounded-xl border border-warning/40 bg-warning/10 p-4 text-sm"><h3 className="font-medium">Legacy authentication detected</h3><p>Saving will migrate {legacyAliases.join(' and ')} to the canonical {legacyAliases.map((name) => name === 'nodeHttpAuth' ? 'httpNodeAuth' : 'httpStaticAuth').join(' and ')} surfaces. Secrets remain redacted.</p></div>}
    {editable && <fieldset disabled={disabled} className="space-y-6">
      <div className="space-y-3"><div className="flex items-center justify-between"><h3 className="font-medium">Admin authentication</h3><button type="button" className="action-btn-secondary" onClick={() => { setChanged((current) => ({ ...current, admin: true })); setUsers((current) => [...current, emptyUser()]); }}>Add user</button></div>
        <label className="block text-sm">Session expiry (seconds)<input aria-label="Session expiry seconds" className="input input-bordered mt-1 w-full" min="0" type="number" value={expiry} onChange={(event) => { setChanged((current) => ({ ...current, admin: true })); setExpiry(Number(event.target.value)); }} /></label>
        {users.map((user, index) => <div key={index} className="grid gap-3 rounded-xl border border-border p-3 md:grid-cols-4"><label className="text-sm">Username<input className="input input-bordered mt-1 w-full" value={user.username} onChange={(event) => updateUser(index, { username: event.target.value })} /></label><label className="text-sm">Permission<select aria-label={`Permission for ${user.username || `user ${index + 1}`}`} className="select select-bordered mt-1 w-full" value={user.permissions} onChange={(event) => updateUser(index, { permissions: event.target.value as '*' | 'read' })}><option value="*">Full access</option><option value="read">Read only</option></select></label><label className="text-sm">New password<input className="input input-bordered mt-1 w-full" type="password" value={user.password} onChange={(event) => updateUser(index, { password: event.target.value })} /></label><button type="button" className="btn btn-ghost self-end" aria-label={`Remove ${user.username || `user ${index + 1}`}`} onClick={() => { setChanged((current) => ({ ...current, admin: true })); setUsers((current) => current.filter((_, i) => i !== index)); }}>Remove</button></div>)}
      </div>
      <Surface title="HTTP node authentication" value={node} onChange={(value) => { setChanged((current) => ({ ...current, node: true })); setNode(value); }} />
      <Surface title="Static HTTP authentication" value={staticAuth} onChange={(value) => { setChanged((current) => ({ ...current, static: true })); setStaticAuth(value); }} />
      <button type="button" className="action-btn-primary" onClick={save} disabled={saving}>Save Security Center</button>
    </fieldset>}
    <ConfirmationDialog isOpen={confirmMigration} title="Apply authentication migration" description={`Canonical targets: ${legacyAliases.map((name) => name === 'nodeHttpAuth' ? 'httpNodeAuth' : 'httpStaticAuth').join(', ')}. Before: legacy aliases with redacted credentials. After: canonical surfaces with redacted credentials.`} acknowledgement="I understand that this explicitly migrates the legacy authentication aliases." variant="warning" onCancel={() => setConfirmMigration(false)} onConfirm={apply} />
  </section>;
}

function Surface({ title, value, onChange }: { title: string; value: BasicAuth; onChange: (value: BasicAuth) => void }) {
  return <div className="space-y-3"><h3 className="font-medium">{title}</h3><div className="grid gap-3 md:grid-cols-2"><label className="text-sm">Username<input className="input input-bordered mt-1 w-full" value={value.user} onChange={(event) => onChange({ ...value, user: event.target.value })} /></label><label className="text-sm">Replacement bcrypt hash<input className="input input-bordered mt-1 w-full" type="password" value={value.pass} onChange={(event) => onChange({ ...value, pass: event.target.value })} /></label></div><p className="text-sm text-base-content/65">Leave blank to preserve the existing hash. New values must be bcrypt hashes and are never displayed.</p></div>;
}
