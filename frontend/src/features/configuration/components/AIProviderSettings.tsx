import { useEffect, useState } from 'react';
import { Save, Wifi } from 'lucide-react';
import { toast } from 'sonner';
import { aiService, type AIConfigInput, type AIProviderStatus } from '../services';
import { InputField, SelectField, ToggleField } from './FormFields';
import { useT } from '@/i18n';

const initialConfig: AIConfigInput = { enabled: false, provider: 'offline', endpoint: '', model: '', timeoutSeconds: 30 };

export function AIProviderSettings() {
  const { t } = useT();
  const [config, setConfig] = useState<AIConfigInput>(initialConfig);
  const [status, setStatus] = useState<AIProviderStatus>();
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [testing, setTesting] = useState(false);

  useEffect(() => {
    Promise.all([aiService.getConfig(), aiService.getStatus()])
      .then(([savedConfig, providerStatus]) => {
        setConfig({ ...savedConfig, apiKey: '', timeoutSeconds: savedConfig.timeoutSeconds || 30 });
        setStatus(providerStatus);
      })
      .catch(() => toast.error('Unable to load AI provider settings'))
      .finally(() => setLoading(false));
  }, []);

  const update = <K extends keyof AIConfigInput>(key: K, value: AIConfigInput[K]) => {
    setConfig((current) => ({ ...current, [key]: value }));
  };
  const timeout = config.timeoutSeconds ?? 30;
  const timeoutValid = Number.isInteger(timeout) && timeout >= 1 && timeout <= 300;

  const save = async () => {
    if (!timeoutValid) return;
    setSaving(true);
    try {
      const input = { ...config };
      if (config.provider === 'ollama') delete input.apiKey;
      const savedConfig = await aiService.saveConfig(input);
      setConfig({ ...savedConfig, apiKey: '', timeoutSeconds: savedConfig.timeoutSeconds || 30 });
      setStatus(await aiService.getStatus());
      toast.success('AI provider settings saved');
    } catch {
      toast.error('Unable to save AI provider settings');
    } finally {
      setSaving(false);
    }
  };

  const testConnection = async () => {
    setTesting(true);
    try {
      setStatus(await aiService.testConfig());
      toast.success('AI provider connection is ready');
    } catch {
      setStatus(await aiService.getStatus().catch(() => undefined));
      toast.error('AI provider connection test failed');
    } finally {
      setTesting(false);
    }
  };

  if (loading) return <div className="py-8 text-sm text-ds-text-muted">{t('configuration:aiLoading')}</div>;

  const remoteProvider = config.provider === 'openai' || config.provider === 'anthropic';
  const ollamaProvider = config.provider === 'ollama';
  return (
    <div className="space-y-6">
      <p className="text-sm text-ds-text-secondary">{t('configuration:aiProviderDesc')}</p>

      <div className="rounded-box border border-ds-border-subtle p-4 text-sm" aria-live="polite">
        <p className="font-medium text-base-content">{t('configuration:aiCapability', { status: status?.status ?? 'unknown' })}</p>
        {status?.reason && <p className="mt-1 text-ds-text-secondary">{status.reason}</p>}
      </div>

      <ToggleField label={t('configuration:aiEnable')} value={config.enabled ?? false} onChange={(value) => update('enabled', value)} disabled={saving || testing} />
      <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
        <SelectField label={t('configuration:aiProviderLabel')} value={config.provider ?? 'offline'} onChange={(value) => update('provider', value as AIConfigInput['provider'])} options={[{ value: 'offline', label: t('configuration:aiProviders.offline') }, { value: 'openai', label: t('configuration:aiProviders.openai') }, { value: 'anthropic', label: t('configuration:aiProviders.anthropic') }, { value: 'ollama', label: t('configuration:aiProviders.ollama') }]} disabled={saving || testing} />
        <InputField label="Model" value={config.model ?? ''} onChange={(value) => update('model', String(value))} placeholder={remoteProvider ? 'gpt-4o-mini' : 'local'} disabled={saving || testing} />
        <label className="space-y-1 text-xs text-ds-text-secondary">{t('configuration:aiTimeoutSeconds')}<input aria-label={t('configuration:aiTimeoutSeconds')} className="block w-full rounded-md border border-solid px-3 py-2 text-sm" type="number" min={1} max={300} step={1} value={timeout} onChange={event => update('timeoutSeconds', Number(event.target.value))} disabled={saving || testing} /><span className="block">{t('configuration:aiTimeoutHelp')}</span></label>
        {(remoteProvider || ollamaProvider) && <>
          <InputField label={t('configuration:aiEndpoint')} value={config.endpoint ?? ''} onChange={(value) => update('endpoint', String(value))} placeholder={ollamaProvider ? 'http://localhost:11434' : 'https://api.example.com/v1/chat/completions'} disabled={saving || testing} />
          {remoteProvider && <InputField label={t('configuration:aiApiKey')} type="password" value={config.apiKey ?? ''} onChange={(value) => update('apiKey', String(value))} help={t('configuration:aiApiKeyHelp')} disabled={saving || testing} />}
          {ollamaProvider && <ToggleField label={t('configuration:aiAllowPrivateHTTP')} value={config.allowPrivateHTTP ?? false} onChange={(value) => update('allowPrivateHTTP', value)} disabled={saving || testing} />}
        </>}
      </div>
      <div className="flex flex-wrap gap-3">
        <button type="button" onClick={save} disabled={saving || testing || !timeoutValid} className="action-btn-primary"><Save className="h-4 w-4" />{saving ? t('configuration:aiSaving') : t('configuration:aiSave')}</button>
        <button type="button" onClick={testConnection} disabled={saving || testing} className="action-btn-secondary"><Wifi className="h-4 w-4" />{testing ? t('configuration:aiTesting') : t('configuration:aiTest')}</button>
      </div>
    </div>
  );
}
