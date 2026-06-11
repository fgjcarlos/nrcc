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

  // Pattern detection is gated off (FEATURES.patternDetection === false) because
  // its backend returns 501. The UI must advertise it as upcoming, never expose
  // an interactive trigger that would dump users into a generic error toast.
  it('gates pattern detection behind a "coming soon" state, no interactive trigger', () => {
    vi.mocked(useFlowDetailData).mockReturnValue({
      flow: { id: 'flow-1', label: 'My Flow', nodes: [] },
      metrics: undefined,
      allFlows: { flows: [] },
      isLoading: false,
      flowError: false,
      refetchFlow: vi.fn(),
    } as unknown as ReturnType<typeof useFlowDetailData>);

    renderDetail();

    // The section is still discoverable...
    expect(screen.getByText(/detect reusable patterns/i)).toBeInTheDocument();
    expect(screen.getByText(/coming soon/i)).toBeInTheDocument();
    // ...but there is no clickable "Detect Patterns" control.
    expect(
      screen.queryByRole('button', { name: /detect patterns/i })
    ).toBeNull();
  });
});
