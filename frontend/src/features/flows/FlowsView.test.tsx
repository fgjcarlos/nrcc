import { beforeEach, describe, expect, it, vi } from 'vitest';
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { api } from '@/shared/lib';
import { I18nProvider, i18n } from '@/i18n';
import { aiService } from '@/features/configuration/services/aiService';
import { FlowsView } from './FlowsView';

vi.mock('@/shared/lib', () => ({ api: { get: vi.fn() } }));
vi.mock('@/features/configuration/services/aiService', () => ({ aiService: { getStatus: vi.fn(), analyzeFlow: vi.fn() } }));

beforeEach(() => {
  vi.clearAllMocks();
  vi.mocked(api.get).mockResolvedValue({ data: new Blob([JSON.stringify([{ id: 'n1', type: 'inject' }])], { type: 'application/json' }) });
  vi.mocked(aiService.getStatus).mockResolvedValue({ status: 'ready', provider: 'openai' });
});

const renderFlows = async (locale: 'en' | 'es' = 'en') => {
  await i18n.changeLanguage(locale);
  return render(<I18nProvider><MemoryRouter><FlowsView /></MemoryRouter></I18nProvider>);
};

describe('FlowsView review-first assistant', () => {
  it('unwraps actual raw export JSON, previews actual payload and sends only after explicit confirmation', async () => {
    vi.mocked(aiService.analyzeFlow).mockResolvedValueOnce({ summary: 'preview', previewToken: 'token', request: { endpoint: 'https://provider.test', body: { messages: [{ content: 'redacted' }] } } }).mockResolvedValueOnce({ summary: 'parsed summary' });
    await renderFlows();
    expect(await screen.findByRole('heading', { name: 'Flows' })).toBeInTheDocument();
    expect(api.get).toHaveBeenCalledWith('/flows/export', { responseType: 'blob' });
    fireEvent.click(screen.getByRole('button', { name: 'Review request' }));
    await screen.findByText(/redacted/);
    expect(aiService.analyzeFlow).toHaveBeenCalledWith(expect.objectContaining({ action: 'explain', previewOnly: true, flow: [{ id: 'n1', type: 'inject' }] }));
    expect(aiService.analyzeFlow).toHaveBeenCalledTimes(1);
    fireEvent.click(screen.getByRole('button', { name: 'Send reviewed request' }));
    await screen.findByText('parsed summary');
    expect(aiService.analyzeFlow).toHaveBeenLastCalledWith(expect.objectContaining({ previewToken: 'token' }));
  });

  it('uses the suggest wire action and clears stale previews without sending automatically', async () => {
    vi.mocked(aiService.analyzeFlow).mockResolvedValueOnce({ summary: 'preview', previewToken: 'token', request: { body: {} } }).mockRejectedValueOnce({ response: { data: { error: { code: 'AI_PREVIEW_CHANGED' } } } });
    await renderFlows();
    fireEvent.change(await screen.findByLabelText('Assistant action'), { target: { value: 'suggest' } });
    fireEvent.click(screen.getByRole('button', { name: 'Review request' }));
    await screen.findByRole('button', { name: 'Send reviewed request' });
    expect(aiService.analyzeFlow).toHaveBeenCalledWith(expect.objectContaining({ action: 'suggest' }));
    fireEvent.change(screen.getByLabelText('Assistant action'), { target: { value: 'audit' } });
    expect(screen.queryByRole('button', { name: 'Send reviewed request' })).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Review request' }));
    await waitFor(() => expect(aiService.analyzeFlow).toHaveBeenCalledTimes(2));
    expect(screen.queryByRole('button', { name: 'Send reviewed request' })).not.toBeInTheDocument();
  });

  it('uses real English and Spanish catalog translations', async () => {
    await renderFlows('en');
    expect(await screen.findByRole('heading', { name: 'Flows' })).toBeInTheDocument();
    cleanup();
    await renderFlows('es');
    expect(await screen.findByRole('heading', { name: 'Flujos' })).toBeInTheDocument();
  });

  it('allows candidate generation from an empty flow export and shows parsed detail fields', async () => {
    vi.mocked(api.get).mockResolvedValue({ data: new Blob(['[]'], { type: 'application/json' }) });
    vi.mocked(aiService.analyzeFlow).mockResolvedValueOnce({ summary: 'preview', previewToken: 'candidate-token', request: { body: {} } }).mockResolvedValueOnce({ summary: 'summary', suggestions: ['Reduce duplication'], auditFindings: ['Old node'], candidateFlow: { nodes: [{ id: 'new', type: 'inject' }] } });
    await renderFlows();
    fireEvent.change(await screen.findByLabelText('Assistant action'), { target: { value: 'generate' } });
    expect(screen.getByRole('button', { name: 'Review request' })).toBeEnabled();
    fireEvent.click(screen.getByRole('button', { name: 'Review request' }));
    fireEvent.click(await screen.findByRole('button', { name: 'Send reviewed request' }));
    expect(await screen.findByText('Reduce duplication')).toBeInTheDocument();
    expect(screen.getByText('Old node')).toBeInTheDocument();
  });

  it('shows a provider timeout distinctly from a generic request failure', async () => {
    vi.mocked(aiService.analyzeFlow).mockRejectedValue({ response: { status: 504, data: { error: { code: 'AI_PROVIDER_TIMEOUT' } } } });
    await renderFlows();
    fireEvent.click(await screen.findByRole('button', { name: 'Review request' }));
    expect(await screen.findByRole('alert')).toHaveTextContent('The AI provider timed out. Try again or adjust its timeout in settings.');
  });

  it('distinguishes failed flow loading from a provider that is merely not ready', async () => {
    vi.mocked(api.get).mockRejectedValue(new Error('export failed'));
    await renderFlows();
    expect(await screen.findByRole('alert')).toHaveTextContent('Could not load current flows or AI capability.');
    expect(screen.queryByText('Flow assistance is unavailable until an AI provider is ready.')).not.toBeInTheDocument();
  });

  it('links to provider settings when capability is not ready', async () => {
    vi.mocked(aiService.getStatus).mockResolvedValue({ status: 'incomplete', provider: 'offline' });
    await renderFlows();
    expect(await screen.findByText('Flow assistance is unavailable until an AI provider is ready.')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Configure AI provider settings' })).toHaveAttribute('href', '/configuration#ai');
  });

  it('disables controls while preview is pending so edited inputs cannot race a stale response', async () => {
    let finish!: (value: { summary: string; previewToken: string; request: { body: object } }) => void;
    vi.mocked(aiService.analyzeFlow).mockReturnValue(new Promise(resolve => { finish = resolve; }));
    await renderFlows();
    fireEvent.click(await screen.findByRole('button', { name: 'Review request' }));
    expect(screen.getByLabelText('Assistant action')).toBeDisabled();
    fireEvent.change(screen.getByLabelText('Additional instructions'), { target: { value: 'changed' } });
    finish({ summary: 'old', previewToken: 'old', request: { body: {} } });
    await waitFor(() => expect(screen.queryByRole('button', { name: 'Send reviewed request' })).not.toBeInTheDocument());
  });
});
