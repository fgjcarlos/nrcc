import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { api } from '@/shared/lib';
import { useT } from '@/i18n';
import { aiService, type AIProviderStatus, type FlowAIAction, type FlowAIResponse } from '@/features/configuration/services/aiService';

const actions: FlowAIAction[] = ['explain', 'suggest', 'audit', 'generate'];
type FlowNode = Record<string, unknown>;

export function FlowsView() {
  const { t } = useT();
  const [flow, setFlow] = useState<FlowNode[] | null>(null);
  const [status, setStatus] = useState<AIProviderStatus>();
  const [loading, setLoading] = useState(true);
  const [action, setAction] = useState<FlowAIAction>('explain');
  const [prompt, setPrompt] = useState('');
  const [preview, setPreview] = useState<FlowAIResponse>();
  const [result, setResult] = useState<FlowAIResponse>();
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    let cancelled = false;
    Promise.all([api.get<Blob>('/flows/export', { responseType: 'blob' }), aiService.getStatus()])
      .then(async ([exported, capability]) => {
        const exportedFlow: unknown = JSON.parse(await exported.data.text());
        if (!Array.isArray(exportedFlow)) throw new Error('Flow export was not an array');
        if (!cancelled) { setFlow(exportedFlow as FlowNode[]); setStatus(capability); }
      })
      .catch(() => { if (!cancelled) setError(t('configuration:flows.loadError')); })
      .finally(() => { if (!cancelled) setLoading(false); });
    return () => { cancelled = true; };
  }, [t]);

  const clearPreview = () => { setPreview(undefined); setResult(undefined); };
  const requestPreview = async () => {
    if (!flow || busy) return;
    setBusy(true); setError(''); clearPreview();
    try { setPreview(await aiService.analyzeFlow({ action, flow, prompt, previewOnly: true })); }
    catch (e) {
      const failure = e as { response?: { status?: number; data?: { error?: { code?: string } } } };
      setError(failure.response?.status === 504 || failure.response?.data?.error?.code === 'AI_PROVIDER_TIMEOUT'
        ? t('configuration:flows.timeoutError') : t('configuration:flows.requestError'));
    } finally { setBusy(false); }
  };
  const send = async () => {
    if (!preview?.previewToken || busy) return;
    setBusy(true); setError(''); setResult(undefined);
    try { setResult(await aiService.analyzeFlow({ action, flow, prompt, previewToken: preview.previewToken })); }
    catch (e) {
      const failure = e as { response?: { status?: number; data?: { error?: { code?: string } } } };
      const code = failure.response?.data?.error?.code;
      setError(code === 'AI_PREVIEW_CHANGED' ? t('configuration:flows.previewChanged')
        : failure.response?.status === 504 || code === 'AI_PROVIDER_TIMEOUT' ? t('configuration:flows.timeoutError')
          : t('configuration:flows.requestError'));
      setPreview(undefined);
    } finally { setBusy(false); }
  };
  const ready = status?.status === 'ready';
  const download = (content: unknown, filename: string) => {
    const url = URL.createObjectURL(new Blob([JSON.stringify(content, null, 2)], { type: 'application/json' }));
    const anchor = document.createElement('a'); anchor.href = url; anchor.download = filename; anchor.click(); URL.revokeObjectURL(url);
  };

  return <section className="space-y-5" aria-labelledby="flows-title">
    <header><h1 id="flows-title" className="text-2xl font-bold">{t('configuration:flows.title')}</h1><p className="mt-2 text-sm text-ds-text-secondary">{t('configuration:flows.description')}</p></header>
    {loading ? <p role="status">{t('configuration:flows.loading')}</p> : error && flow === null ? <p role="alert">{error}</p> : !ready ? <div className="rounded-box border border-ds-border-subtle p-4" role="status"><p>{t('configuration:flows.disabled')}</p><Link className="link link-primary" to="/configuration#ai">{t('configuration:flows.settingsLink')}</Link></div> : <>
      <label className="block text-sm font-medium">{t('configuration:flows.action')}<select className="input input-bordered mt-1 block w-full" value={action} disabled={busy} onChange={e => { setAction(e.target.value as FlowAIAction); clearPreview(); }}>
        {actions.map(value => <option key={value} value={value}>{t(`configuration:flows.actions.${value}`)}</option>)}
      </select></label>
      <label className="block text-sm font-medium">{t('configuration:flows.prompt')}<textarea className="textarea textarea-bordered mt-1 block w-full" value={prompt} disabled={busy} onChange={e => { setPrompt(e.target.value); clearPreview(); }} /></label>
      <button className="action-btn-primary" type="button" disabled={busy} onClick={() => void requestPreview()}>{busy ? t('configuration:flows.working') : t('configuration:flows.preview')}</button>
      {error && <p role="alert">{error}</p>}
      {preview && <div className="space-y-3 rounded-box border border-ds-border-subtle p-4">
        <h2 className="font-semibold">{t('configuration:flows.review')}</h2><p>{t('configuration:flows.endpoint')}: {preview.request?.endpoint ?? '—'}</p>
        <h3 className="font-medium">{t('configuration:flows.payload')}</h3><pre className="max-h-96 overflow-auto whitespace-pre-wrap break-all rounded-box bg-base-200 p-3 text-xs">{preview.request?.body === undefined ? t('configuration:flows.noPayload') : JSON.stringify(preview.request.body, null, 2)}</pre>
        <button className="action-btn-primary" type="button" disabled={busy} onClick={() => void send()}>{t('configuration:flows.send')}</button>
      </div>}
      {result && <div className="space-y-3" aria-live="polite"><h2 className="font-semibold">{t('configuration:flows.result')}</h2><p>{result.summary}</p>
        {result.suggestions?.length ? <section><h3 className="font-medium">{t('configuration:flows.suggestions')}</h3><ul className="list-disc pl-5">{result.suggestions.map((item, i) => <li key={i}>{item}</li>)}</ul></section> : null}
        {result.auditFindings?.length ? <section><h3 className="font-medium">{t('configuration:flows.auditFindings')}</h3><ul className="list-disc pl-5">{result.auditFindings.map((item, i) => <li key={i}>{item}</li>)}</ul></section> : null}
        {result.candidateFlow && <section><h3 className="font-medium">{t('configuration:flows.candidate')}</h3><pre className="max-h-96 overflow-auto whitespace-pre-wrap rounded-box bg-base-200 p-3 text-xs">{JSON.stringify(result.candidateFlow.nodes, null, 2)}</pre>
          <div className="flex gap-3"><button type="button" className="action-btn-secondary" onClick={() => void navigator.clipboard.writeText(JSON.stringify(result.candidateFlow!.nodes, null, 2))}>{t('configuration:flows.copy')}</button><button type="button" className="action-btn-secondary" onClick={() => download(result.candidateFlow!.nodes, 'candidate-flow.json')}>{t('configuration:flows.download')}</button></div>
          <p className="text-sm">{t('configuration:flows.importDirections')}</p>
        </section>}
      </div>}
    </>}
  </section>;
}
