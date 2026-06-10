import { describe, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { FlowDetailView } from './FlowDetailView';
import { useFlowDetailData } from '@/features/flows/hooks';

vi.mock('@/features/flows/hooks', () => ({
  useFlowDetailData: vi.fn(),
  useFlowDetailActions: vi.fn(() => ({
    analyzeFlowMutation: { mutate: vi.fn(), isPending: false, data: undefined },
    detectPatternsMutation: { mutate: vi.fn(), isPending: false },
    aiFlowMutation: { mutate: vi.fn(), isPending: false },
  })),
}));

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
});
