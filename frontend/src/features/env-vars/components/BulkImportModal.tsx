import { useMemo, useRef, useState } from 'react';
import { CheckCircle2, AlertTriangle, ClipboardPaste } from 'lucide-react';
import { envService, type BulkEnvResult } from '../services/envService';
import { useT } from '@/i18n';
import { Button, Dialog, DialogBody, DialogFooter, Table, Textarea } from '@/shared/components/ui';

interface BulkImportModalProps {
  open: boolean;
  onClose: () => void;
  onImported: () => void;
}

// Example lines are code; the hint line above them comes from the catalog.
const EXAMPLE_LINES = `API_URL=https://example.test
DEBUG=true#boolean
PORT=8080#number
TOKEN=hunter2#secret
`;

export function BulkImportModal({ open, onClose, onImported }: BulkImportModalProps) {
  const { t } = useT();
  const placeholder = `${t('env-vars:bulkPlaceholderHint')}\n${EXAMPLE_LINES}`;
  const [content, setContent] = useState('');
  const [report, setReport] = useState<BulkEnvResult | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const inFlightRef = useRef(false);

  const issueByLine = useMemo(() => {
    const map = new Map<number, string>();
    if (!report) return map;
    for (const iss of report.issues) map.set(iss.line, iss.reason);
    return map;
  }, [report]);

  const linesForRender = content.split('\n');
  const lineIssueCount = report ? report.issues.filter((iss) => iss.line > 0).length : 0;
  const generalIssues = report ? report.issues.filter((iss) => !(iss.line > 0)) : [];

  async function handleValidate() {
    if (inFlightRef.current) return;
    inFlightRef.current = true;
    setError(null);
    setLoading(true);
    try {
      const result = await envService.bulkImport(content, false);
      setReport(result);
    } catch (err) {
      const message = err instanceof Error ? err.message : t('env-vars:bulkValidateFailed');
      setError(message);
      setReport(null);
    } finally {
      inFlightRef.current = false;
      setLoading(false);
    }
  }

  async function handleCommit() {
    if (!report?.valid) return;
    if (inFlightRef.current) return;
    inFlightRef.current = true;
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
      const message = err instanceof Error ? err.message : t('env-vars:bulkImportFailed');
      setError(message);
    } finally {
      inFlightRef.current = false;
      setLoading(false);
    }
  }

  function handlePasteExample() {
    setContent(placeholder);
    setReport(null);
    setError(null);
  }

  return (
    // A stray backdrop click must not throw away pasted lines.
    <Dialog
      open={open}
      onClose={onClose}
      title={t('env-vars:bulkTitle')}
      description={t('env-vars:bulkHelp')}
      size="lg"
      closeOnBackdrop={false}
    >
      <DialogBody className="space-y-4">
        <Textarea
          aria-label={t('env-vars:bulkTitle')}
          className="h-48 font-mono text-xs"
          placeholder={placeholder}
          value={content}
          onChange={(e) => setContent(e.target.value)}
        />

        {report?.valid && !error && (
          <p role="status" className="text-sm text-ds-on-tint-success">
            {t('env-vars:bulkReady', { count: report.lines.length })}
          </p>
        )}

        {(error || (report && !report.valid)) && (
          <div
            role="alert"
            className="space-y-1 rounded-box border border-ds-danger/35 bg-ds-danger/10 px-3 py-2 text-sm text-ds-on-tint-error"
          >
            {error && <p>{error}</p>}
            {report && !report.valid && lineIssueCount > 0 && (
              <p className="font-semibold">{t('env-vars:bulkLinesInvalid', { count: lineIssueCount })}</p>
            )}
            {/* Issues without a line (limits, server errors) have no table row to sit on. */}
            {report && !report.valid && generalIssues.map((issue, idx) => <p key={idx}>{issue.reason}</p>)}
            {report && !report.valid && lineIssueCount === 0 && generalIssues.length === 0 && (
              <p>{report.summary || t('env-vars:bulkImportFailed')}</p>
            )}
          </div>
        )}

        {report && (
          <Table>
            <thead>
              <tr>
                <th>{t('env-vars:bulkLine')}</th>
                <th>{t('env-vars:bulkKey')}</th>
                <th>{t('env-vars:bulkValue')}</th>
                <th>{t('env-vars:bulkType')}</th>
                <th>{t('env-vars:bulkStatus')}</th>
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
                  <tr key={lineNum} className={issue ? 'bg-error/10' : undefined}>
                    <td className="font-mono">{lineNum}</td>
                    <td className="font-mono">{parsed?.key ?? trimmed.split('=')[0] ?? ''}</td>
                    <td className="font-mono break-all">
                      {parsed?.value ?? (trimmed.includes('=') ? trimmed.split('=').slice(1).join('=') : '')}
                    </td>
                    <td>{parsed?.type ?? '—'}</td>
                    <td>
                      {issue ? (
                        <span className="flex items-center gap-1 text-ds-on-tint-error">
                          <AlertTriangle size={12} aria-hidden="true" />
                          {issue}
                        </span>
                      ) : parsed ? (
                        t('env-vars:bulkOk')
                      ) : (
                        ''
                      )}
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </Table>
        )}
      </DialogBody>

      <DialogFooter>
        <Button type="button" variant="ghost" onClick={handlePasteExample} className="mr-auto">
          <ClipboardPaste size={14} aria-hidden="true" />
          {t('env-vars:bulkExample')}
        </Button>
        <Button type="button" variant="secondary" onClick={handleValidate} disabled={loading}>
          {t('env-vars:bulkValidate')}
        </Button>
        <Button type="button" variant="solid" onClick={handleCommit} disabled={loading || !report?.valid}>
          <CheckCircle2 size={14} aria-hidden="true" />
          {t('env-vars:bulkImportAction')}
        </Button>
      </DialogFooter>
    </Dialog>
  );
}
