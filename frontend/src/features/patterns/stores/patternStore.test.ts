import { beforeEach, describe, expect, it } from 'vitest';
import { usePatternStore, type PatternAnalysisResult } from './patternStore';

// The store is a module-level singleton; reset it before each test.
beforeEach(() => {
  usePatternStore.getState().reset();
});

const get = () => usePatternStore.getState();

describe('patternStore', () => {
  it('starts with an empty, idle state', () => {
    const s = get();
    expect(s.selectedFlowIds.size).toBe(0);
    expect(s.analyzing).toBe(false);
    expect(s.lastAnalysis).toBeNull();
    expect(s.error).toBeNull();
  });

  it('toggleFlow adds then removes a flow id', () => {
    get().toggleFlow('flow-1');
    expect(get().selectedFlowIds.has('flow-1')).toBe(true);

    get().toggleFlow('flow-1');
    expect(get().selectedFlowIds.has('flow-1')).toBe(false);
  });

  it('toggleFlow keeps other selections intact', () => {
    get().toggleFlow('a');
    get().toggleFlow('b');
    get().toggleFlow('a'); // remove a
    const ids = get().selectedFlowIds;
    expect(ids.has('a')).toBe(false);
    expect(ids.has('b')).toBe(true);
    expect(ids.size).toBe(1);
  });

  it('selectAll replaces the selection set', () => {
    get().toggleFlow('old');
    get().selectAll(['x', 'y', 'z']);
    const ids = get().selectedFlowIds;
    expect(ids.has('old')).toBe(false);
    expect([...ids].sort()).toEqual(['x', 'y', 'z']);
  });

  it('clearSelection empties the set without touching analysis state', () => {
    get().selectAll(['a', 'b']);
    get().setAnalyzing(true);
    get().clearSelection();
    expect(get().selectedFlowIds.size).toBe(0);
    expect(get().analyzing).toBe(true);
  });

  it('setLastAnalysis stores the result and clears any prior error', () => {
    get().setError('boom');
    const result: PatternAnalysisResult = {
      patternId: 'p1',
      patterns: [],
      analyzedAt: '2026-01-01T00:00:00Z',
      flowCount: 3,
    };
    get().setLastAnalysis(result);
    expect(get().lastAnalysis).toEqual(result);
    expect(get().error).toBeNull();
  });

  it('setError records an error message', () => {
    get().setError('analysis failed');
    expect(get().error).toBe('analysis failed');
  });

  it('reset returns the store to its initial state', () => {
    get().selectAll(['a', 'b']);
    get().setAnalyzing(true);
    get().setError('x');
    get().reset();
    const s = get();
    expect(s.selectedFlowIds.size).toBe(0);
    expect(s.analyzing).toBe(false);
    expect(s.lastAnalysis).toBeNull();
    expect(s.error).toBeNull();
  });
});
