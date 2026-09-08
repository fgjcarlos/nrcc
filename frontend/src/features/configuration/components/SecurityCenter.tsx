import { useEffect, useState } from 'react';
import { toast } from 'sonner';
import { ConfirmationDialog } from '@/shared/components/ConfirmationDialog';
import { configService } from '../services';
import type { NodeRedConfigResponse } from '../lib/configTransformers';

type AdminUser = { username: string; permissions: '*' | 'read'; password: string };
type BasicAuth = { user: string; pass: string; storedPass?: string };

interface Props {
  config: NodeRedConfigResponse | null;
  rawSettingsContent: string;
  editable: boolean;
}

const emptyUser = (): AdminUser => ({ username: '', permissions: '*', password: '' });
const aliases = (source: string) => ['nodeHttpAuth', 'staticAuth'].filter((name) => new RegExp(`\\b${name}\\s*:`).test(source));

export function SecurityCenter({ config, rawSettingsContent, editable }: Props) {
  const [users, setUsers] = useState<AdminUser[]>([]);
  const [expiry, setExpiry] = useState(0);
  const [node, setNode] = useState<BasicAuth>({ user: '', pass: '' });
  const [staticAuth, setStaticAuth] = useState<BasicAuth>({ user: '', pass: '' });
  const [saving, setSaving] = useState(false);
  const [confirmMigration, setConfirmMigration] = useState(false);
  const legacyAliases = aliases(rawSettingsContent);

  useEffect(() => {
    const admin = config?.adminAuth as { users?: Array<{ username?: string; permissions?: '*' | 'read' }>; sessionExpiryTime?: number } | undefined;
    const nodeAuth = config?.httpNodeAuth as Partial<BasicAuth> | undefined;
    const staticSurface = config?.httpStaticAuth as Partial<BasicAuth> | undefined;
    setUsers(admin?.users?.map((user) => ({ username: user.username ?? '', permissions: user.permissions ?? '*', password: '' })) ?? []);
    setExpiry(admin?.sessionExpiryTime ?? 0);
    setNode({ user: nodeAuth?.user ?? '', pass: '', storedPass: nodeAuth?.pass });
    setStaticAuth({ user: staticSurface?.user ?? '', pass: '', storedPass: staticSurface?.pass });
  }, [config]);

  const payload = () => {
    const { adminAuth: _admin, httpNodeAuth: _node, httpStaticAuth: _static, ...preserved } = config ?? {};
    return {
    ...preserved,
    adminAuth: users.length ? { type: 'credentials', users: users.map(({ username, permissions, password }) => ({ username, permissions, password })) , ...(expiry > 0 ? { sessionExpiryTime: expiry } : {}) } : null,
    httpNodeAuth: node.user ? { user: node.user, pass: node.pass || node.storedPass || '' } : null,
    httpStaticAuth: staticAuth.user ? { user: staticAuth.user, pass: staticAuth.pass || staticAuth.storedPass || '' } : null,
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
    setConfirmMigration(false);
    setSaving(true);
    try {
      await configService.updateConfig(payload());
      toast.success('Security Center settings saved');
    } catch {
      toast.error('Unable to save Security Center settings');
    } finally {
      setSaving(false);
    }
  };
  const updateUser = (index: number, update: Partial<AdminUser>) => setUsers((current) => current.map((user, i) => i === index ? { ...user, ...update } : user));
  const disabled = !editable || saving;

  return <section aria-labelledby="security-center-title" className="space-y-6">
    <div><h2 id="security-center-title" className="text-lg font-medium">Security Center</h2><p className="text-sm text-base-content/65">Passwords and bcrypt hashes are never shown. Leave a password blank to preserve its existing value.</p></div>
    {!editable && <p role="status" className="rounded-xl border border-warning/40 bg-warning/10 p-4 text-sm text-warning">Security Center is read-only because this runtime configuration is not editable.</p>}
    {editable && legacyAliases.length > 0 && <div className="rounded-xl border border-warning/40 bg-warning/10 p-4 text-sm"><h3 className="font-medium">Legacy authentication detected</h3><p>Saving will migrate {legacyAliases.join(' and ')} to the canonical {legacyAliases.map((name) => name === 'nodeHttpAuth' ? 'httpNodeAuth' : 'httpStaticAuth').join(' and ')} surfaces. Secrets remain redacted.</p></div>}
    {editable && <fieldset disabled={disabled} className="space-y-6">
      <div className="space-y-3"><div className="flex items-center justify-between"><h3 className="font-medium">Admin authentication</h3><button type="button" className="action-btn-secondary" onClick={() => setUsers((current) => [...current, emptyUser()])}>Add user</button></div>
        <label className="block text-sm">Session expiry (seconds)<input aria-label="Session expiry seconds" className="input input-bordered mt-1 w-full" min="0" type="number" value={expiry} onChange={(event) => setExpiry(Number(event.target.value))} /></label>
        {users.map((user, index) => <div key={index} className="grid gap-3 rounded-xl border border-border p-3 md:grid-cols-4"><label className="text-sm">Username<input className="input input-bordered mt-1 w-full" value={user.username} onChange={(event) => updateUser(index, { username: event.target.value })} /></label><label className="text-sm">Permission<select aria-label={`Permission for ${user.username || `user ${index + 1}`}`} className="select select-bordered mt-1 w-full" value={user.permissions} onChange={(event) => updateUser(index, { permissions: event.target.value as '*' | 'read' })}><option value="*">Full access</option><option value="read">Read only</option></select></label><label className="text-sm">New password<input className="input input-bordered mt-1 w-full" type="password" value={user.password} onChange={(event) => updateUser(index, { password: event.target.value })} /></label><button type="button" className="btn btn-ghost self-end" aria-label={`Remove ${user.username || `user ${index + 1}`}`} onClick={() => setUsers((current) => current.filter((_, i) => i !== index))}>Remove</button></div>)}
      </div>
      <Surface title="HTTP node authentication" value={node} onChange={setNode} />
      <Surface title="Static HTTP authentication" value={staticAuth} onChange={setStaticAuth} />
      <button type="button" className="action-btn-primary" onClick={save} disabled={saving}>Save Security Center</button>
    </fieldset>}
    <ConfirmationDialog isOpen={confirmMigration} title="Apply authentication migration" description={`Canonical targets: ${legacyAliases.map((name) => name === 'nodeHttpAuth' ? 'httpNodeAuth' : 'httpStaticAuth').join(', ')}. Before: legacy aliases with redacted credentials. After: canonical surfaces with redacted credentials.`} acknowledgement="I understand that this explicitly migrates the legacy authentication aliases." variant="warning" onCancel={() => setConfirmMigration(false)} onConfirm={apply} />
  </section>;
}

function Surface({ title, value, onChange }: { title: string; value: BasicAuth; onChange: (value: BasicAuth) => void }) {
  return <div className="space-y-3"><h3 className="font-medium">{title}</h3><div className="grid gap-3 md:grid-cols-2"><label className="text-sm">Username<input className="input input-bordered mt-1 w-full" value={value.user} onChange={(event) => onChange({ ...value, user: event.target.value })} /></label><label className="text-sm">Replacement bcrypt hash<input className="input input-bordered mt-1 w-full" type="password" value={value.pass} onChange={(event) => onChange({ ...value, pass: event.target.value })} /></label></div><p className="text-sm text-base-content/65">Leave blank to preserve the existing hash. New values must be bcrypt hashes and are never displayed.</p></div>;
}
