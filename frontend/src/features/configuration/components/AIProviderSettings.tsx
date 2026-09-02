import { useEffect, useState } from 'react';
import { Bot, Save, Wifi } from 'lucide-react';
import { toast } from 'sonner';
import { aiService, type AIConfigInput, type AIProviderStatus } from '../services';
import { InputField, SelectField, ToggleField } from './FormFields';

const initialConfig: AIConfigInput = { enabled: false, provider: 'offline', endpoint: '', model: '' };

export function AIProviderSettings() {
  const [config, setConfig] = useState<AIConfigInput>(initialConfig);
  const [status, setStatus] = useState<AIProviderStatus>();
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [testing, setTesting] = useState(false);

  useEffect(() => {
    Promise.all([aiService.getConfig(), aiService.getStatus()])
      .then(([savedConfig, providerStatus]) => {
        setConfig({ ...savedConfig, apiKey: '' });
        setStatus(providerStatus);
      })
      .catch(() => toast.error('Unable to load AI provider settings'))
      .finally(() => setLoading(false));
  }, []);

  const update = <K extends keyof AIConfigInput>(key: K, value: AIConfigInput[K]) => {
    setConfig((current) => ({ ...current, [key]: value }));
  };

  const save = async () => {
    setSaving(true);
    try {
      const savedConfig = await aiService.saveConfig(config);
      setConfig({ ...savedConfig, apiKey: '' });
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

  if (loading) return <div className="py-8 text-sm text-base-content/60">Loading AI provider settings…</div>;

  const remoteProvider = config.provider === 'openai';
  return (
    <div className="space-y-6">
      <div className="flex items-start gap-3">
        <Bot className="mt-0.5 h-5 w-5 text-primary" aria-hidden="true" />
        <div>
          <h3 className="text-lg font-medium text-base-content">AI Provider</h3>
          <p className="text-sm text-base-content/60">Configure optional, review-first AI flow assistance.</p>
        </div>
      </div>

      <div className="rounded-xl border border-border bg-base-200/35 p-4 text-sm" aria-live="polite">
        <p className="font-medium text-base-content">Capability: {status?.status ?? 'unknown'}</p>
        {status?.reason && <p className="mt-1 text-base-content/65">{status.reason}</p>}
      </div>

      <ToggleField label="Enable AI assistance" value={config.enabled ?? false} onChange={(value) => update('enabled', value)} disabled={saving || testing} />
      <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
        <SelectField label="Provider" value={config.provider ?? 'offline'} onChange={(value) => update('provider', value as AIConfigInput['provider'])} options={[{ value: 'offline', label: 'Offline' }, { value: 'openai', label: 'OpenAI-compatible' }]} disabled={saving || testing} />
        <InputField label="Model" value={config.model ?? ''} onChange={(value) => update('model', String(value))} placeholder={remoteProvider ? 'gpt-4o-mini' : 'local'} disabled={saving || testing} />
        {remoteProvider && <>
          <InputField label="Endpoint" value={config.endpoint ?? ''} onChange={(value) => update('endpoint', String(value))} placeholder="https://api.example.com/v1/chat/completions" disabled={saving || testing} />
          <InputField label="API key" type="password" value={config.apiKey ?? ''} onChange={(value) => update('apiKey', String(value))} help="Write-only. Leave blank to retain the configured key." disabled={saving || testing} />
        </>}
      </div>
      <div className="flex flex-wrap gap-3">
        <button type="button" onClick={save} disabled={saving || testing} className="action-btn-primary"><Save className="h-4 w-4" />{saving ? 'Saving…' : 'Save AI settings'}</button>
        <button type="button" onClick={testConnection} disabled={saving || testing} className="action-btn-secondary"><Wifi className="h-4 w-4" />{testing ? 'Testing…' : 'Test connection'}</button>
      </div>
    </div>
  );
}
