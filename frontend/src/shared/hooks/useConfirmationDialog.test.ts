import { describe, expect, it } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useConfirmationDialog } from './useConfirmationDialog';

interface TestItem {
  id: string;
  name: string;
}

describe('useConfirmationDialog', () => {
  it('initializes with closed dialog and null pending item', () => {
    const { result } = renderHook(() => useConfirmationDialog<TestItem>());

    expect(result.current.isOpen).toBe(false);
    expect(result.current.pendingItem).toBe(null);
  });

  it('opens dialog and sets pending item when open is called', () => {
    const { result } = renderHook(() => useConfirmationDialog<TestItem>());
    const testItem: TestItem = { id: '1', name: 'Test Item' };

    act(() => {
      result.current.open(testItem);
    });

    expect(result.current.isOpen).toBe(true);
    expect(result.current.pendingItem).toEqual(testItem);
  });

  it('closes dialog and clears pending item when close is called', () => {
    const { result } = renderHook(() => useConfirmationDialog<TestItem>());
    const testItem: TestItem = { id: '1', name: 'Test Item' };

    act(() => {
      result.current.open(testItem);
    });

    expect(result.current.isOpen).toBe(true);

    act(() => {
      result.current.close();
    });

    expect(result.current.isOpen).toBe(false);
    expect(result.current.pendingItem).toBe(null);
  });

  it('supports multiple open/close cycles', () => {
    const { result } = renderHook(() => useConfirmationDialog<TestItem>());
    const item1: TestItem = { id: '1', name: 'Item 1' };
    const item2: TestItem = { id: '2', name: 'Item 2' };

    act(() => {
      result.current.open(item1);
    });
    expect(result.current.pendingItem?.id).toBe('1');

    act(() => {
      result.current.close();
    });

    act(() => {
      result.current.open(item2);
    });
    expect(result.current.pendingItem?.id).toBe('2');
  });

  it('replaces pending item when open is called while already open', () => {
    const { result } = renderHook(() => useConfirmationDialog<TestItem>());
    const item1: TestItem = { id: '1', name: 'Item 1' };
    const item2: TestItem = { id: '2', name: 'Item 2' };

    act(() => {
      result.current.open(item1);
    });
    expect(result.current.pendingItem?.id).toBe('1');

    act(() => {
      result.current.open(item2);
    });
    expect(result.current.pendingItem?.id).toBe('2');
  });
});
