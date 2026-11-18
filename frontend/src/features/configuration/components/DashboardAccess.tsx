import { useEffect, useMemo, useState } from 'react';
import { AxiosError } from 'axios';
import api from '@/shared/lib';
import { useT } from '@/i18n';
import { Button, Input, Select } from '@/shared/components/ui';

type State = 'available' | 'absent' | 'malformed' | 'unknown';
type Discovery = { packages: { state: State }; flows: { state: State }; legacy?: { path: string }; flowFuse: unknown[]; uiBases: { nodeId: string; path?: string; state: State }[] };
type Target = 'legacy' | 'flowfuse';
// One inline outcome line: `status` for progress and success, `alert` for failures (no toasts on this flow).
type Outcome = { role: 'status' | 'alert'; key: string; tone?: 'success' };

const ok = <T,>(response: { data: { data: T } }) => response.data.data;
const i18n = (key: string) => `security:dashboardAccess.${key}`;
const box = 'rounded-box border px-3 py-2 text-sm';
const tones = {
  alert: 'border-ds-danger/40 bg-ds-danger/10 text-ds-on-tint-error',
  success: 'border-ds-success/40 bg-ds-success/10 text-ds-on-tint-success',
  warning: 'border-ds-warning/40 bg-ds-warning/10 text-ds-on-tint-warning',
};

/** Dashboard HTTP/Socket.IO policy form. Rendered inside the /security row, which carries its title. */
export function DashboardAccess({ editable, expectedRevision, onApplied }: { editable: boolean; expectedRevision?: string; onApplied: () => void }) {
  const { t } = useT();
  const [discovery, setDiscovery] = useState<Discovery>();
  const [target, setTarget] = useState<Target>('legacy');
  const [recipe, setRecipe] = useState('basic-auth');
  const [username, setUsername] = useState('');
  const [secret, setSecret] = useState('');
  const [status, setStatus] = useState<Outcome | null>({ role: 'status', key: i18n('loading') });
  const [saving, setSaving] = useState(false);
  useEffect(() => { void api.get('/dashboards/discovery').then(ok<Discovery>).then((value) => { setDiscovery(value); setStatus(null); }).catch(() => setStatus({ role: 'alert', key: i18n('discoveryFailed') })); }, []);
  const eligible = useMemo(() => {
    if (!discovery || discovery.packages.state !== 'available') return false;
    return target === 'legacy'
      ? Boolean(discovery.legacy)
      : discovery.flows.state === 'available' && discovery.flowFuse.length > 0 && discovery.uiBases.length > 0 && discovery.uiBases.every((base) => base.state === 'available' && base.path);
  }, [discovery, target]);
  const formDisabled = !editable || !expectedRevision || saving;
  const stateText = (state?: State) => t(i18n(`states.${state ?? 'unknown'}`), { defaultValue: state });
  const paths = discovery?.uiBases.map((base) => base.path || t(i18n('pathUnavailable'), { nodeId: base.nodeId })).join(', ') || t(i18n('noneDiscovered'));
  const legacyMissing = Boolean(discovery) && !discovery?.legacy;
  const apply = async () => {
    if (formDisabled || !eligible || !secret || (recipe === 'basic-auth' && !username)) return;
    setSaving(true); setStatus(null);
    try {
      await api.post('/dashboards/access', { target, recipe, secret, ...(recipe === 'basic-auth' ? { username } : {}), expectedRevision });
      setSecret(''); setStatus({ role: 'status', key: i18n('applied'), tone: 'success' }); onApplied();
    } catch (error) {
      const code = error instanceof AxiosError ? error.response?.data?.error?.code : '';
      setStatus({ role: 'alert', key: code === 'SETTINGS_REVISION_CONFLICT' ? 'security:applyErrors.revisionConflict' : code === 'APPLY_IN_FLIGHT' ? 'security:applyErrors.inFlight' : 'security:applyErrors.generic' });
    } finally { setSaving(false); }
  };
  return <div className="max-w-2xl space-y-4">
    <p className="text-sm text-ds-text-secondary"><strong className="font-semibold text-ds-text-primary">{t(i18n('discovery'))}</strong> {t(i18n('discoverySummary'), { packages: stateText(discovery?.packages.state), flows: stateText(discovery?.flows.state), legacy: discovery?.legacy ? t(i18n('legacyPath'), { path: discovery.legacy.path }) : stateText('absent'), paths })}</p>
    <aside aria-label={t(i18n('previewLabel'))} className={`${box} border-ds-border-subtle bg-ds-bg-surface`}><h3 className="font-semibold text-ds-text-primary">{t(i18n('previewTitle'))}</h3><p className="text-ds-text-secondary">{t(i18n('previewBody'), { target, recipe })}</p></aside>
    {status && <p key={status.role} role={status.role} className={status.role === 'alert' ? `${box} ${tones.alert}` : status.tone ? `${box} ${tones.success}` : 'text-sm text-ds-text-secondary'}>{t(status.key)}</p>}
    {!editable && <p role="status" className={`${box} ${tones.warning}`}>{t(i18n('readOnly'))}</p>}
    <fieldset disabled={formDisabled} className="space-y-4">
      <div className="space-y-1">
        <Select id="dashboard-access-target" label={t(i18n('target'))} aria-describedby={legacyMissing ? 'dashboard-access-target-help' : undefined} value={target} onChange={(event) => setTarget(event.target.value as Target)}>
          <option value="legacy" disabled={legacyMissing}>{t(i18n('legacyOption'))}</option>
          <option value="flowfuse">{t(i18n('flowFuseOption'), { count: discovery?.uiBases.length ?? 0 })}</option>
        </Select>
        {legacyMissing && <p id="dashboard-access-target-help" className="text-xs text-ds-text-secondary">{t(i18n('legacyUnavailable'))}</p>}
      </div>
      <Select id="dashboard-access-recipe" label={t(i18n('recipe'))} value={recipe} onChange={(event) => setRecipe(event.target.value)}>
        <option value="basic-auth">{t(i18n('basicAuth'))}</option>
        <option value="reverse-proxy-sso">{t(i18n('reverseProxySso'))}</option>
      </Select>
      {recipe === 'basic-auth' && <Input id="dashboard-access-username" label={t(i18n('username'))} value={username} onChange={(event) => setUsername(event.target.value)} />}
      <Input id="dashboard-access-secret" label={t(i18n('secret'))} type="password" value={secret} onChange={(event) => setSecret(event.target.value)} />
      <Button type="button" disabled={formDisabled || !eligible || !secret || (recipe === 'basic-auth' && !username)} onClick={apply}>{saving ? t(i18n('applying')) : t(i18n('apply'))}</Button>
    </fieldset>
  </div>;
}
