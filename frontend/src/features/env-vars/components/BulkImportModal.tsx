import { useMemo, useState } from 'react';
import { X, CheckCircle2, AlertTriangle, ClipboardPaste } from 'lucide-react';
import { envService, type BulkEnvResult } from '../services/envService';

interface BulkImportModalProps {
  open: boolean;
  onClose: () => void;
  onImported: () => void;
}

const PLACEHOLDER = `# KEY=VALUE[#type] — one per line
API_URL=https://example.test
DEBUG=true#boolean
PORT=8080#number
TOKEN=hunter2#secret
`;

export function BulkImportModal({ open, onClose, onImported }: BulkImportModalProps) {
  const [content, setContent] = useState('');
  const [report, setReport] = useState<BulkEnvResult | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  const issueByLine = useMemo(() => {
    const map = new Map<number, string>();
    if (!report) return map;
    for (const iss of report.issues) map.set(iss.line, iss.reason);
    return map;
  }, [report]);

  if (!open) return null;

  const linesForRender = content.split('\n');

  async function handleValidate() {
    setError(null);
    setLoading(true);
    try {
      const result = await envService.bulkImport(content, false);
      setReport(result);
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Validation failed';
      setError(message);
      setReport(null);
    } finally {
      setLoading(false);
    }
  }

  async function handleCommit() {
    if (!report?.valid) return;
    setError(null);
    setLoading(true);
    try {
      const result = await envService.bulkImport(content, true);
      setReport(result);
      if (result.valid) {
        onImported();
        onClose();
        setContent('');
      }
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Import failed';
      setError(message);
    } finally {
      setLoading(false);
    }
  }

  function handlePasteExample() {
    setContent(PLACEHOLDER);
    setReport(null);
    setError(null);
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
      <div className="w-full max-w-3xl rounded-lg bg-base-100 shadow-xl">
        <div className="flex items-center justify-between border-b border-base-300 px-5 py-3">
          <h2 className="text-lg font-semibold">Bulk import environment variables</h2>
          <button type="button" className="btn btn-ghost btn-sm" onClick={onClose} aria-label="Close">
            <X size={18} />
          </button>
        </div>

        <div className="space-y-4 px-5 py-4">
          <p className="text-sm text-base-content/70">
            Paste <code className="rounded bg-base-200 px-1">KEY=VALUE[#type]</code> lines. Supported types:{' '}
            <code>string</code>, <code>number</code>, <code>boolean</code>, <code>secret</code>. Secrets stay in
            NRCC and never reach Node-RED. Empty lines and lines starting with <code>#</code> are ignored.
          </p>

          <textarea
            className="textarea textarea-bordered h-48 w-full font-mono text-xs"
            placeholder={PLACEHOLDER}
            value={content}
            onChange={(e) => {
              setContent(e.target.value);
              setReport(null);
            }}
          />

          <div className="flex flex-wrap items-center gap-2">
            <button type="button" className="btn btn-primary btn-sm" onClick={handleValidate} disabled={loading}>
              Validate
            </button>
            <button
              type="button"
              className="btn btn-success btn-sm"
              onClick={handleCommit}
              disabled={loading || !report?.valid}
            >
              <CheckCircle2 size={14} />
              Import
            </button>
            <button type="button" className="btn btn-ghost btn-sm" onClick={handlePasteExample}>
              <ClipboardPaste size={14} />
              Example
            </button>
            {report && (
              <span
                className={
                  report.valid
                    ? 'text-sm text-success'
                    : 'text-sm text-error'
                }
              >
                {report.summary}
              </span>
            )}
            {error && <span className="text-sm text-error">{error}</span>}
          </div>

          {report && (
            <div className="rounded border border-base-300">
              <table className="table table-xs">
                <thead>
                  <tr>
                    <th>Line</th>
                    <th>Key</th>
                    <th>Value</th>
                    <th>Type</th>
                    <th>Status</th>
                  </tr>
                </thead>
                <tbody>
                  {linesForRender.map((line, idx) => {
                    const lineNum = idx + 1;
                    const trimmed = line.trim();
                    if (trimmed === '' || trimmed.startsWith('#')) return null;
                    const parsed = report.lines.find((l) => l.line === lineNum);
                    const issue = issueByLine.get(lineNum);
                    return (
                      <tr key={lineNum} className={issue ? 'bg-error/10' : parsed ? 'bg-success/5' : ''}>
                        <td className="font-mono">{lineNum}</td>
                        <td className="font-mono">{parsed?.key ?? trimmed.split('=')[0] ?? ''}</td>
                        <td className="font-mono break-all">
                          {parsed?.value ?? (trimmed.includes('=') ? trimmed.split('=').slice(1).join('=') : '')}
                        </td>
                        <td>{parsed?.type ?? '—'}</td>
                        <td>
                          {issue ? (
                            <span className="flex items-center gap-1 text-error">
                              <AlertTriangle size={12} />
                              {issue}
                            </span>
                          ) : parsed ? (
                            'ok'
                          ) : (
                            ''
                          )}
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}