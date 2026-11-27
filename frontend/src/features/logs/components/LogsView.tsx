import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { logsService, type LogEntry, type LogFilters, type LogsPage } from '../services/logsService';

const LEVELS = ['error', 'fatal', 'warn', 'info', 'debug', 'trace', 'unknown'];
const ROW_HEIGHT = 48;
const VISIBLE_ROWS = 12;
const PAGE_SIZE = 1000;

type Criteria = LogFilters;
type BucketSelection = { from: number; to: number } | null;

const timestamp = (value: string) => Date.parse(value);
const localDateToRFC3339 = (value: string) => value ? new Date(value).toISOString() : undefined;
const dateToLocalInput = (value: string) => {
  const date = new Date(value);
  date.setMinutes(date.getMinutes() - date.getTimezoneOffset());
  return date.toISOString().slice(0, 16);
};
const matches = (entry: LogEntry, filters: Criteria, startedAt?: string) => {
  const time = timestamp(entry.timestamp);
  const lowerQuery = filters.q?.toLowerCase();
  return Number.isFinite(time) &&
    (!lowerQuery || entry.message.toLowerCase().includes(lowerQuery)) &&
    (!filters.levels?.length || filters.levels.some((level) => level.toLowerCase() === entry.level.toLowerCase())) &&
    (!filters.from || time >= timestamp(filters.from)) &&
    (!filters.to || time <= timestamp(filters.to)) &&
    (!filters.sinceRestart || (startedAt !== undefined && timestamp(entry.runStartedAt ?? '') === timestamp(startedAt)));
};

function levelColor(level: string) {
  switch (level.toLowerCase()) {
    case 'error': return 'bg-error';
    case 'fatal': return 'bg-secondary';
    case 'warn': return 'bg-warning';
    case 'info': return 'bg-primary';
    case 'debug': return 'bg-accent';
    case 'trace': return 'bg-neutral';
    default: return 'bg-base-300';
  }
}

function mergeEntries(...groups: LogEntry[][]) {
  const unique = new Map<string, LogEntry>();
  for (const group of groups) for (const entry of group) unique.set(entry.id, entry);
  return [...unique.values()].sort((a, b) => timestamp(a.timestamp) - timestamp(b.timestamp));
}

export function LogsView() {
  const { t } = useTranslation('logs');
  const [entries, setEntries] = useState<LogEntry[]>([]);
  const [queryInput, setQueryInput] = useState('');
  const [query, setQuery] = useState('');
  const [levels, setLevels] = useState<string[]>([]);
  const [dateFrom, setDateFrom] = useState('');
  const [dateTo, setDateTo] = useState('');
  const [sinceRestart, setSinceRestart] = useState(false);
  const [selection, setSelection] = useState<BucketSelection>(null);
  const [paused, setPaused] = useState(true);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(false);
  const [historyFailed, setHistoryFailed] = useState(false);
  const [expanded, setExpanded] = useState<string | null>(null);
  const [scrollTop, setScrollTop] = useState(0);
  const [restartVersion, setRestartVersion] = useState(0);
  const scroller = useRef<HTMLDivElement>(null);
  const loaded = useRef<LogEntry[]>([]);
  const highWatermark = useRef('');
  const [startedAt, setStartedAt] = useState<string>();

  useEffect(() => {
    const timer = setTimeout(() => setQuery(queryInput.trim()), 250);
    return () => clearTimeout(timer);
  }, [queryInput]);

  const filters = useMemo<Criteria>(() => ({
    q: query || undefined,
    levels,
    from: localDateToRFC3339(dateFrom),
    to: localDateToRFC3339(dateTo),
    sinceRestart,
  }), [query, levels, dateFrom, dateTo, sinceRestart]);

  const restartSnapshot = useCallback(() => setRestartVersion((version) => version + 1), []);

  useEffect(() => {
    const controller = new AbortController();
    let streamStarted = false;
    let launch: string | undefined;
    setLoading(true);
    setError(false);
    setHistoryFailed(false);
    setEntries([]);
    loaded.current = [];
    highWatermark.current = '';
    setStartedAt(undefined);
    setScrollTop(0);
    if (scroller.current) scroller.current.scrollTop = 0;

    const onEntry = (entry: LogEntry) => {
      if (controller.signal.aborted) return;
      highWatermark.current = entry.id;
      if (!matches(entry, filters, launch)) return;
      loaded.current = mergeEntries(loaded.current, [entry]);
      setEntries(loaded.current);
    };
    const onSnapshot = (page: LogsPage) => {
      if (controller.signal.aborted) return;
      highWatermark.current = page.highWatermark;
      launch = page.startedAt;
      setStartedAt(launch);
      loaded.current = mergeEntries(page.entries);
      setEntries(loaded.current);
      if (!paused && !streamStarted) {
        streamStarted = true;
        void runStream();
      }
    };
    const runStream = async () => {
      while (!controller.signal.aborted) {
        try {
          await logsService.stream(highWatermark.current, controller.signal, onEntry, () => {
            if (!controller.signal.aborted) {
              controller.abort();
              restartSnapshot();
            }
          }, launch);
        } catch {
          if (!controller.signal.aborted) setError(true);
        }
        if (!controller.signal.aborted) await new Promise((resolve) => setTimeout(resolve, 1000));
      }
    };

    void logsService.history(filters, controller.signal, onSnapshot).then((page) => {
      if (controller.signal.aborted) return;
      // Keep stream arrivals received while cursor pages were loading.
      if (!highWatermark.current) highWatermark.current = page.highWatermark;
      launch ??= page.startedAt;
      setStartedAt(launch);
      loaded.current = mergeEntries(page.entries, loaded.current);
      setEntries(loaded.current);
      setLoading(false);
      setError(false);
      if (!paused && !streamStarted) {
        streamStarted = true;
        void runStream();
      }
    }).catch(() => {
      if (!controller.signal.aborted) {
        setLoading(false);
        setError(true);
        setHistoryFailed(true);
      }
    });

    return () => controller.abort();
  }, [filters, paused, restartVersion, restartSnapshot]);

  const filtered = useMemo(() => entries.filter((entry) => {
    if (!matches(entry, filters, startedAt)) return false;
    if (!selection) return true;
    const time = timestamp(entry.timestamp);
    return time >= selection.from && time <= selection.to;
  }), [entries, filters, selection, startedAt]);

  useEffect(() => {
    setScrollTop(0);
    if (scroller.current) scroller.current.scrollTop = 0;
  }, [filters, selection]);
  useEffect(() => {
    const maxScroll = Math.max(0, filtered.length * ROW_HEIGHT - VISIBLE_ROWS * ROW_HEIGHT);
    if (scrollTop > maxScroll) {
      const corrected = Math.floor(maxScroll / ROW_HEIGHT) * ROW_HEIGHT;
      setScrollTop(corrected);
      if (scroller.current) scroller.current.scrollTop = corrected;
    }
  }, [filtered.length, scrollTop]);

  const buckets = useMemo(() => {
    const grouped = new Map<number, Record<string, number>>();
    for (const entry of entries) {
      if (!matches(entry, filters, startedAt)) continue;
      const time = timestamp(entry.timestamp);
      const bucket = Math.floor(time / 60000) * 60000;
      const counts = grouped.get(bucket) ?? {};
      const level = entry.level.toLowerCase();
      counts[level] = (counts[level] ?? 0) + 1;
      grouped.set(bucket, counts);
    }
    return [...grouped.entries()].sort(([a], [b]) => a - b).slice(-24);
  }, [entries, filters, startedAt]);

  const start = Math.min(Math.floor(scrollTop / ROW_HEIGHT), Math.max(0, filtered.length - VISIBLE_ROWS));
  const visible = filtered.slice(start, start + VISIBLE_ROWS);
  const selectedEntry = entries.find((entry) => entry.id === expanded);
  const selectBucket = (time: number) => setSelection({ from: time, to: time + 59999 });
  const clearFilters = () => {
    setQueryInput(''); setQuery(''); setLevels([]); setDateFrom(''); setDateTo(''); setSinceRestart(false); setSelection(null);
  };
  const quickFilter = (next: Partial<Criteria>) => {
    setQueryInput(''); setQuery(''); setLevels(next.levels ?? []);
    setDateFrom(next.from ? dateToLocalInput(next.from) : '');
    setDateTo(next.to ? dateToLocalInput(next.to) : '');
    setSinceRestart(next.sinceRestart ?? false);
    setSelection(null);
  };
  const quickErrors = () => quickFilter({ levels: ['error', 'fatal'], from: new Date(Date.now() - 15 * 60000).toISOString() });
  const quickWarnings = () => quickFilter({ levels: ['warn'] });
  const quickSinceRestart = () => quickFilter({ sinceRestart: true });

  return <main className="mx-auto max-w-7xl space-y-5 p-5">
    <header className="flex flex-wrap items-center justify-between gap-3">
      <div><h1 className="text-2xl font-semibold">{t('title')}</h1><p className="text-sm text-base-content/70">{t('retention', { limit: PAGE_SIZE })}</p></div>
      <button type="button" className="btn btn-secondary" onClick={() => setPaused((value) => !value)} aria-pressed={!paused}>{paused ? t('resume') : t('pause')}</button>
    </header>
    {error && <p role="alert" className="alert alert-error">{t('error')}</p>}
    <section aria-label={t('filters')} className="flex flex-wrap items-end gap-3 rounded-box border border-base-300 bg-base-100 p-4">
      <label className="form-control"><span>{t('search')}</span><input className="input input-bordered" value={queryInput} onChange={(event) => setQueryInput(event.target.value)} /></label>
      <fieldset><legend>{t('levels')}</legend><div className="flex flex-wrap gap-2">{LEVELS.map((level) => <label key={level} className="flex items-center gap-1"><input type="checkbox" className="checkbox" checked={levels.includes(level)} onChange={() => { setLevels((current) => current.includes(level) ? current.filter((item) => item !== level) : [...current, level]); setSelection(null); }} />{t(`level.${level}`)}</label>)}</div></fieldset>
      <label className="form-control"><span>{t('from')}</span><input aria-label={t('from')} className="input input-bordered" type="datetime-local" value={dateFrom} onChange={(event) => { setDateFrom(event.target.value); setSelection(null); }} /></label>
      <label className="form-control"><span>{t('to')}</span><input aria-label={t('to')} className="input input-bordered" type="datetime-local" value={dateTo} onChange={(event) => { setDateTo(event.target.value); setSelection(null); }} /></label>
      <button type="button" className="btn btn-ghost" onClick={clearFilters}>{t('reset')}</button>
    </section>
    <div className="flex flex-wrap gap-2">
      <button type="button" className="btn btn-outline" onClick={quickErrors}>{t('errors15')}</button>
      <button type="button" className="btn btn-outline" onClick={quickWarnings}>{t('warnings')}</button>
      <button type="button" className="btn btn-outline" onClick={quickSinceRestart}>{t('sinceRestart')}</button>
      {selection && <button type="button" className="btn btn-ghost" onClick={() => setSelection(null)}>{t('clearChartSelection')}</button>}
    </div>
    <section className="rounded-box border border-base-300 bg-base-100 p-4" aria-label={t('chartTitle')}>
      <h2 className="mb-3 font-semibold">{t('chartTitle')}</h2><div className="flex h-32 items-end gap-2 overflow-x-auto" role="group" aria-label={t('chartTitle')}>
        {buckets.map(([time, counts]) => <button key={time} type="button" className="flex h-full min-w-10 flex-col justify-end" aria-label={t('bucket', { time: new Date(time).toLocaleTimeString(), count: Object.values(counts).reduce((a, b) => a + b, 0) })} aria-pressed={selection?.from === time} onClick={() => selectBucket(time)}>
          <span className="flex flex-col-reverse" style={{ height: `${Math.max(8, Math.min(100, Object.values(counts).reduce((a, b) => a + b, 0) * 8))}%` }} aria-hidden="true">{Object.entries(counts).map(([level, count]) => <span key={level} className={`block ${levelColor(level)}`} style={{ height: `${count * 8}px` }} />)}</span><small>{new Date(time).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}</small>
        </button>)}
      </div><ul className="mt-2 flex flex-wrap gap-3" aria-label={t('legend')}>{LEVELS.map((level) => <li key={level} className="flex items-center gap-1"><span aria-hidden="true" className={`inline-block h-3 w-3 ${levelColor(level)}`} />{t(`level.${level}`)}</li>)}</ul><h3 className="mt-3 text-sm font-medium">{t('topErrors')}</h3>{[...entries.reduce((counts, entry) => { if (matches(entry, filters, startedAt) && (entry.level === 'error' || entry.level === 'fatal')) counts.set(entry.message, (counts.get(entry.message) ?? 0) + 1); return counts; }, new Map<string, number>())].sort((a, b) => b[1] - a[1]).slice(0, 5).map(([message, count]) => <button key={message} type="button" className="mr-2 underline" onClick={() => { setQueryInput(message); setSelection(null); }}>{message} ({count})</button>)}
    </section>
    {loading ? <p role="status">{t('loading')}</p> : filtered.length === 0 ? (historyFailed ? null : <p role="status">{t('empty')}</p>) : <section aria-label={t('tableTitle')}>
      <div ref={scroller} className="overflow-auto" style={{ maxHeight: ROW_HEIGHT * VISIBLE_ROWS }} onScroll={(event) => setScrollTop(event.currentTarget.scrollTop)}>
        <table className="ds-table w-full"><caption className="sr-only">{t('tableTitle')}</caption><thead className="sticky top-0 bg-base-200"><tr><th>{t('time')}</th><th>{t('levelLabel')}</th><th>{t('source')}</th><th>{t('message')}</th><th>{t('details')}</th></tr></thead><tbody>
          {start > 0 && <tr aria-hidden="true" style={{ height: start * ROW_HEIGHT }}><td colSpan={5} /> </tr>}
          {visible.map((entry) => <tr key={entry.id} style={{ height: ROW_HEIGHT, maxHeight: ROW_HEIGHT }}><td>{new Date(entry.timestamp).toLocaleString()}</td><td><span className={`badge ${entry.level === 'error' ? 'badge-error' : entry.level === 'fatal' ? 'badge-secondary' : entry.level === 'warn' ? 'badge-warning' : entry.level === 'debug' ? 'badge-accent' : 'badge-neutral'}`}>{t(`level.${entry.level.toLowerCase()}`, { defaultValue: entry.level })}</span></td><td>{entry.source}</td><td className="max-w-xl truncate">{entry.message}</td><td><button type="button" className="btn btn-ghost btn-sm" aria-expanded={expanded === entry.id} onClick={() => setExpanded((current) => current === entry.id ? null : entry.id)}>{expanded === entry.id ? t('collapse') : t('expand')}</button></td></tr>)}
          {start + VISIBLE_ROWS < filtered.length && <tr aria-hidden="true" style={{ height: (filtered.length - start - VISIBLE_ROWS) * ROW_HEIGHT }}><td colSpan={5} /></tr>}
        </tbody></table>
      </div>
      {selectedEntry && <section className="mt-3 rounded-box border border-base-300 p-3" aria-label={t('expandedDetails')}><h3 className="font-semibold">{selectedEntry.message}</h3><pre className="max-h-64 overflow-auto whitespace-pre-wrap">{JSON.stringify(selectedEntry, null, 2)}</pre></section>}
    </section>}
  </main>;
}
