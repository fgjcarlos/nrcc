import { describe, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { FlowDetailView } from './FlowDetailView';
import { useAICapability, useFlowDetailData } from '@/features/flows/hooks';
import { useAuth } from '@/features/auth/hooks';

vi.mock('@/features/flows/hooks', () => ({
  useFlowDetailData: vi.fn(),
  useFlowDetailActions: vi.fn(() => ({
    analyzeFlowMutation: { mutate: vi.fn(), isPending: false, data: undefined },
    aiFlowMutation: { mutate: vi.fn(), isPending: false },
  })),
  useAICapability: vi.fn(() => ({
    isReady: false,
    isLoading: false,
    message: 'AI actions are unavailable until the configured provider is ready.',
  })),
}));
vi.mock('@/features/auth/hooks', () => ({ useAuth: vi.fn(() => ({ user: { role: 'viewer' } })) }));

function renderDetail(id = 'flow-1') {
  return render(
    <MemoryRouter initialEntries={[`/flows/${id}`]}>
      <Routes>
        <Route path="/flows/:id" element={<FlowDetailView />} />
      </Routes>
    </MemoryRouter>
  );
}

describe('FlowDetailView', () => {
  function mockUser(role: 'admin' | 'viewer') {
    vi.mocked(useAuth).mockReturnValue({ user: { role } } as ReturnType<typeof useAuth>);
  }

  it('shows an error state with retry when the flow query fails', () => {
    vi.mocked(useFlowDetailData).mockReturnValue({
      flow: undefined,
      metrics: undefined,
      allFlows: undefined,
      isLoading: false,
      flowError: true,
      refetchFlow: vi.fn(),
    } as unknown as ReturnType<typeof useFlowDetailData>);

    renderDetail();

    expect(screen.getByText(/failed to load flow/i)).toBeInTheDocument();
    // A fetch failure must NOT be reported as a missing flow.
    expect(screen.queryByText(/flow not found/i)).toBeNull();
  });

  it('shows not-found when the flow is genuinely missing (no error)', () => {
    vi.mocked(useFlowDetailData).mockReturnValue({
      flow: undefined,
      metrics: undefined,
      allFlows: undefined,
      isLoading: false,
      flowError: false,
      refetchFlow: vi.fn(),
    } as unknown as ReturnType<typeof useFlowDetailData>);

    renderDetail();

    expect(screen.getByText(/flow not found/i)).toBeInTheDocument();
    expect(screen.queryByText(/failed to load flow/i)).toBeNull();
  });

  it('disables detail analysis and copilot actions until the provider is ready', () => {
	  mockUser('viewer');
    vi.mocked(useFlowDetailData).mockReturnValue({
      flow: { id: 'flow-1', label: 'Flow 1', nodes: [] },
      metrics: undefined,
      allFlows: undefined,
      isLoading: false,
      flowError: false,
      refetchFlow: vi.fn(),
    } as unknown as ReturnType<typeof useFlowDetailData>);

    vi.mocked(useAICapability).mockReturnValue({
      isReady: false,
      isLoading: false,
      message: 'AI actions are unavailable until the configured provider is ready.',
    } as ReturnType<typeof useAICapability>);

    renderDetail();

    expect(screen.getByRole('button', { name: 'Analyze Flow' })).toBeDisabled();
    expect(screen.getByRole('button', { name: 'explain' })).toBeDisabled();
    expect(screen.getByText('AI actions are unavailable until the configured provider is ready.')).toBeInTheDocument();
	  expect(screen.queryByRole('link', { name: 'Configure AI provider' })).not.toBeInTheDocument();
  });

  it('links administrators to AI provider configuration when actions are unavailable', () => {
    mockUser('admin');
    vi.mocked(useFlowDetailData).mockReturnValue({
      flow: { id: 'flow-1', label: 'Flow 1', nodes: [] }, metrics: undefined, allFlows: undefined,
      isLoading: false, flowError: false, refetchFlow: vi.fn(),
    } as unknown as ReturnType<typeof useFlowDetailData>);

    renderDetail();

    expect(screen.getByRole('link', { name: 'Configure AI provider' })).toHaveAttribute('href', '/configuration');
  });
});
