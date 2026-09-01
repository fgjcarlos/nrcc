import { describe, expect, it, vi } from 'vitest';
import { fireEvent, render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { FlowsView } from './FlowsView';
import { useFlowsActions } from '@/features/flows/hooks/useFlowsActions';
import { useFlowsData } from '@/features/flows/hooks/useFlowsData';
import { useAICapability } from '@/features/flows/hooks/useAICapability';
import { useAuth } from '@/features/auth/hooks';

vi.mock('@/features/flows/hooks/useFlowsData', () => ({ useFlowsData: vi.fn() }));
vi.mock('@/features/flows/hooks/useFlowsActions', () => ({ useFlowsActions: vi.fn() }));
vi.mock('@/features/flows/hooks/useAICapability', () => ({ useAICapability: vi.fn() }));
vi.mock('@/features/auth/hooks', () => ({ useAuth: vi.fn() }));

vi.mock('sonner', () => ({ toast: { error: vi.fn() } }));

describe('FlowsView', () => {
  function mockUser(role: 'admin' | 'viewer') {
    vi.mocked(useAuth).mockReturnValue({ user: { role } } as ReturnType<typeof useAuth>);
  }

  it('disables batch analysis until the provider is ready', () => {
	  mockUser('viewer');
    vi.mocked(useFlowsData).mockReturnValue({
      flows: [{ id: 'flow-1', label: 'Flow 1', nodes: 1, connections: 0, disabled: false }],
      available: true,
      isLoading: false,
      error: null,
    } as ReturnType<typeof useFlowsData>);
    vi.mocked(useFlowsActions).mockReturnValue({ analyzeFlows: vi.fn() });
    vi.mocked(useAICapability).mockReturnValue({
      isReady: false,
      isLoading: false,
      message: 'AI actions are unavailable until the configured provider is ready.',
    } as ReturnType<typeof useAICapability>);

    render(<MemoryRouter><FlowsView /></MemoryRouter>);
    fireEvent.click(screen.getByRole('button', { name: 'Select flow' }));

    expect(screen.getByRole('button', { name: /analyze with ai/i })).toBeDisabled();
    expect(screen.getByText('AI actions are unavailable until the configured provider is ready.')).toBeInTheDocument();
	  expect(screen.queryByRole('link', { name: 'Configure AI provider' })).not.toBeInTheDocument();
  });

  it('links administrators to AI provider configuration when batch analysis is unavailable', () => {
    mockUser('admin');
    vi.mocked(useFlowsData).mockReturnValue({
      flows: [{ id: 'flow-1', label: 'Flow 1', nodes: 1, connections: 0, disabled: false }],
      available: true, isLoading: false, error: null,
    } as ReturnType<typeof useFlowsData>);
    vi.mocked(useFlowsActions).mockReturnValue({ analyzeFlows: vi.fn() });
    vi.mocked(useAICapability).mockReturnValue({
      isReady: false, isLoading: false, message: 'AI actions are unavailable until the configured provider is ready.',
    } as ReturnType<typeof useAICapability>);

    render(<MemoryRouter><FlowsView /></MemoryRouter>);
    fireEvent.click(screen.getByRole('button', { name: 'Select flow' }));

    expect(screen.getByRole('link', { name: 'Configure AI provider' })).toHaveAttribute('href', '/configuration');
  });
});
