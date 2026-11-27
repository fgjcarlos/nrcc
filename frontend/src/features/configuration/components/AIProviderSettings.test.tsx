import { beforeEach, describe, expect, it, vi } from 'vitest';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { aiService } from '../services';
import { AIProviderSettings } from './AIProviderSettings';

vi.mock('@/i18n', () => ({ useT: () => ({ t: (key: string) => key }) }));
vi.mock('../services', () => ({ aiService: { getConfig: vi.fn(), getStatus: vi.fn(), saveConfig: vi.fn(), testConfig: vi.fn() } }));

beforeEach(() => {
  vi.clearAllMocks();
  vi.mocked(aiService.getConfig).mockResolvedValue({ enabled: true, provider: 'ollama', apiKeyConfigured: false, endpoint: 'http://localhost:11434', allowPrivateHTTP: false });
  vi.mocked(aiService.getStatus).mockResolvedValue({ status: 'ready', provider: 'ollama' });
});

describe('AI provider settings', () => {
  it('offers supported providers and explicit private HTTP opt-in without an Ollama key field', async () => {
    render(<AIProviderSettings />);
    const provider = await screen.findByLabelText('configuration:aiProviderLabel');
    expect(provider.querySelectorAll('option')).toHaveLength(4);
    expect(screen.getByLabelText('configuration:aiAllowPrivateHTTP')).toBeInTheDocument();
    expect(screen.queryByLabelText('configuration:aiApiKey')).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'configuration:aiAllowPrivateHTTP' }));
    fireEvent.click(screen.getByRole('button', { name: 'configuration:aiSave' }));
    await waitFor(() => expect(aiService.saveConfig).toHaveBeenCalledWith(expect.objectContaining({ provider: 'ollama', allowPrivateHTTP: true, timeoutSeconds: 30 })));
    expect(aiService.saveConfig).toHaveBeenCalledWith(expect.not.objectContaining({ apiKey: '' }));
  });

  it('defaults a stored zero timeout to 30 seconds and validates the supported range', async () => {
    vi.mocked(aiService.getConfig).mockResolvedValue({ enabled: false, provider: 'offline', apiKeyConfigured: false, timeoutSeconds: 0 });
    render(<AIProviderSettings />);
    const input = await screen.findByLabelText('configuration:aiTimeoutSeconds');
    expect(input).toHaveValue(30);
    fireEvent.change(input, { target: { value: '301' } });
    expect(screen.getByRole('button', { name: 'configuration:aiSave' })).toBeDisabled();
    fireEvent.change(input, { target: { value: '300' } });
    fireEvent.click(screen.getByRole('button', { name: 'configuration:aiSave' }));
    await waitFor(() => expect(aiService.saveConfig).toHaveBeenCalledWith(expect.objectContaining({ timeoutSeconds: 300 })));
  });

  it('offers an API key for remote providers', async () => {
    vi.mocked(aiService.getConfig).mockResolvedValue({ enabled: true, provider: 'openai', apiKeyConfigured: false });
    render(<AIProviderSettings />);
    await screen.findByLabelText('configuration:aiProviderLabel');
    expect(screen.getByLabelText('configuration:aiApiKey')).toBeInTheDocument();
  });
});
