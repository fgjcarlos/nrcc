import { api, refreshAuthToken } from '@/shared/lib/api';
import { authService } from '@/features/auth/services/authService';

export interface LogEntry {
  id: string;
  timestamp: string;
  level: string;
  source: string;
  message: string;
  runStartedAt?: string;
}

export interface LogsPage {
  entries: LogEntry[];
  generation: string;
  highWatermark: string;
  nextCursor?: string;
  startedAt?: string;
}

const ZERO_START = Date.parse('0001-01-01T00:00:00Z');

export function normalizeRunStartedAt(value?: string): string | undefined {
  if (!value) return undefined;
  const time = Date.parse(value);
  return !Number.isFinite(time) || time === ZERO_START ? undefined : new Date(time).toISOString();
}

export interface LogFilters {
  q?: string;
  levels?: string[];
  from?: string;
  to?: string;
  sinceRestart?: boolean;
}

function paramsFor(filters: LogFilters, cursor?: string, generation?: string) {
  const params = new URLSearchParams();
  if (filters.q) params.set('q', filters.q);
  filters.levels?.forEach((level) => params.append('level', level));
  if (filters.from) params.set('from', filters.from);
  if (filters.to) params.set('to', filters.to);
  if (filters.sinceRestart) params.set('sinceRestart', 'true');
  params.set('limit', '1000');
  if (cursor) params.set('cursor', cursor);
  if (generation) params.set('generation', generation);
  return params;
}

export const logsService = {
  async history(
    filters: LogFilters = {},
    signal?: AbortSignal,
    onSnapshot?: (page: LogsPage) => void,
  ): Promise<LogsPage> {
    const fetchPage = async (cursor?: string, generation?: string) => {
      const response = await api.get<{ data: LogsPage }>(`/logs?${paramsFor(filters, cursor, generation)}`, { signal });
      return response.data.data;
    };

    const fetched = await fetchPage();
    const first = { ...fetched, startedAt: normalizeRunStartedAt(fetched.startedAt) };
    onSnapshot?.(first);
    const pages: LogsPage[] = [first];
    let cursor = first.nextCursor;
    while (cursor && !signal?.aborted) {
      const page = await fetchPage(cursor, first.generation);
      pages.push(page);
      cursor = page.nextCursor;
    }
    const entries = new Map<string, LogEntry>();
    for (const page of pages) for (const entry of page.entries) entries.set(entry.id, entry);
    return { ...first, entries: [...entries.values()].sort((a, b) => Date.parse(a.timestamp) - Date.parse(b.timestamp)) };
  },

  async stream(since: string, signal: AbortSignal, onEntry: (entry: LogEntry) => void, onRestart: () => void, expectedStartedAt?: string) {
    const request = (token: string) => fetch(`/api/logs/stream?since=${encodeURIComponent(since)}`, {
      headers: { Authorization: `Bearer ${token}`, Accept: 'text/event-stream' },
      signal,
    });
    let token = authService.getToken();
    if (!token) token = await refreshAuthToken();
    if (!token) throw new Error('Authentication required');
    let response = await request(token);
    if (response.status === 401 && !signal.aborted) {
      token = await refreshAuthToken();
      if (token) response = await request(token);
    }
    if (response.status === 409) {
      onRestart();
      return;
    }
    if (!response.ok || !response.body) throw new Error(`Log stream failed (${response.status})`);
    const reader = response.body.getReader();
    const decoder = new TextDecoder();
    let buffer = '';
    try {
      while (!signal.aborted) {
        const { value, done } = await reader.read();
        if (done) break;
        buffer += decoder.decode(value, { stream: true });
        let boundary = buffer.indexOf('\n\n');
        while (boundary >= 0) {
          const block = buffer.slice(0, boundary);
          buffer = buffer.slice(boundary + 2);
          const event = block.split(/\r?\n/).find((line) => line.startsWith('event:'))?.slice(6).trim();
          const data = block.split(/\r?\n/).find((line) => line.startsWith('data:'))?.slice(5).trim();
          if (event === 'connected' && data) {
            const connection = JSON.parse(data) as { startedAt?: string };
            if (normalizeRunStartedAt(connection.startedAt) !== normalizeRunStartedAt(expectedStartedAt)) {
              onRestart();
              return;
            }
          }
          if (event === 'restart') {
            onRestart();
            return;
          }
          if (event === 'log' && data) onEntry(JSON.parse(data) as LogEntry);
          boundary = buffer.indexOf('\n\n');
        }
      }
    } finally {
      reader.releaseLock();
    }
  },
};
