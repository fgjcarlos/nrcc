import { useEffect, useMemo, useState } from 'react';
import { AxiosError } from 'axios';
import api from '@/shared/lib';

type State = 'available' | 'absent' | 'malformed' | 'unknown';
type Discovery = { packages: { state: State }; flows: { state: State }; legacy?: { path: string }; flowFuse: unknown[]; uiBases: { nodeId: string; path?: string; state: State }[] };
type Target = 'legacy' | 'flowfuse';

const ok = <T,>(response: { data: { data: T } }) => response.data.data;

export function DashboardAccess({ editable, expectedRevision, onApplied }: { editable: boolean; expectedRevision?: string; onApplied: () => void }) {
  const [discovery, setDiscovery] = useState<Discovery>();
  const [target, setTarget] = useState<Target>('legacy');
  const [recipe, setRecipe] = useState('basic-auth');
  const [username, setUsername] = useState('');
  const [secret, setSecret] = useState('');
  const [status, setStatus] = useState('Loading dashboard discovery…');
  const [saving, setSaving] = useState(false);
  useEffect(() => { void api.get('/dashboards/discovery').then(ok<Discovery>).then((value) => { setDiscovery(value); setStatus(''); }).catch(() => setStatus('Dashboard discovery could not be loaded. Refresh before applying access.')); }, []);
  const eligible = useMemo(() => {
    if (!discovery || discovery.packages.state !== 'available') return false;
    return target === 'legacy'
      ? Boolean(discovery.legacy)
      : discovery.flows.state === 'available' && discovery.flowFuse.length > 0 && discovery.uiBases.length > 0 && discovery.uiBases.every((base) => base.state === 'available' && base.path);
  }, [discovery, target]);
  const formDisabled = !editable || !expectedRevision || saving;
  const paths = discovery?.uiBases.map((base) => base.path || `${base.nodeId} (path unavailable)`).join(', ') || 'none discovered';
  const apply = async () => {
    if (formDisabled || !eligible || !secret || (recipe === 'basic-auth' && !username)) return;
    setSaving(true); setStatus('');
    try {
      await api.post('/dashboards/access', { target, recipe, secret, ...(recipe === 'basic-auth' ? { username } : {}), expectedRevision });
      setSecret(''); setStatus('Dashboard access was applied transactionally.'); onApplied();
    } catch (error) {
      const code = error instanceof AxiosError ? error.response?.data?.error?.code : '';
      setStatus(code === 'SETTINGS_REVISION_CONFLICT' ? 'Settings changed elsewhere. Refresh, review the redacted preview, then retry.' : code === 'APPLY_IN_FLIGHT' ? 'Another transaction is in progress. Wait for it to finish, then retry.' : 'The transaction did not complete. Any failed readiness check is rolled back; review runtime readiness, then retry.');
    } finally { setSaving(false); }
  };
  return <section aria-labelledby="dashboard-access-title" className="space-y-6">
    <div><h2 id="dashboard-access-title" className="text-lg font-medium">Dashboard Access</h2><p className="text-sm text-base-content/65">Configure a reviewed access policy without exposing credentials.</p></div>
    <div role="status" aria-live="polite" className="rounded-xl border border-border bg-base-200/35 p-4 text-sm"><strong>Discovery:</strong> packages {discovery?.packages.state ?? 'unknown'}; flows {discovery?.flows.state ?? 'unknown'}; legacy {discovery?.legacy ? `${discovery.legacy.path} (deprecated)` : 'absent'}; FlowFuse paths {paths}.</div>
    <aside aria-label="Redacted dashboard policy preview" className="rounded-xl border border-border bg-base-200/35 p-4 text-sm"><h3 className="font-medium">Redacted policy preview</h3><p>Target: {target}. Recipe: {recipe}. Credentials, hashes, and authorization values are never shown.</p></aside>
    {status && <p role="status" aria-live="polite" className="rounded-xl border border-warning/40 bg-warning/10 p-4 text-sm text-warning">{status}</p>}
    {!editable && <p role="status" className="rounded-xl border border-warning/40 bg-warning/10 p-4 text-sm text-warning">Dashboard Access is read-only because this runtime configuration is not editable.</p>}
    <fieldset disabled={formDisabled} className="space-y-4">
      <label className="block text-sm">Dashboard target<select aria-label="Dashboard target" className="select select-bordered mt-1 w-full" value={target} onChange={(event) => setTarget(event.target.value as Target)}><option value="legacy">Legacy dashboard (deprecated){discovery?.legacy ? ` (${discovery.legacy.path})` : ' (unavailable)'}</option><option value="flowfuse">FlowFuse dashboard ({discovery?.uiBases.length ?? 0} ui-base paths)</option></select></label>
      <label className="block text-sm">Access recipe<select aria-label="Access recipe" className="select select-bordered mt-1 w-full" value={recipe} onChange={(event) => setRecipe(event.target.value)}><option value="basic-auth">Basic authentication</option><option value="reverse-proxy-sso">Reverse proxy SSO</option></select></label>
      {recipe === 'basic-auth' && <label className="block text-sm">Username<input className="input input-bordered mt-1 w-full" value={username} onChange={(event) => setUsername(event.target.value)} /></label>}
      <label className="block text-sm">Access secret<input className="input input-bordered mt-1 w-full" type="password" value={secret} onChange={(event) => setSecret(event.target.value)} /></label>
      <button type="button" className="action-btn-primary" disabled={formDisabled || !eligible || !secret || (recipe === 'basic-auth' && !username)} onClick={apply}>{saving ? 'Applying…' : 'Apply dashboard access'}</button>
    </fieldset>
  </section>;
}
