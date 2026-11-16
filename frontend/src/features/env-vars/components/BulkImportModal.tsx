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

const PLACEHOLDER = `# KEY=VALUE[#type] — one per line
API_URL=https://example.test
DEBUG=true#boolean
PORT=8080#number
TOKEN=hunter2#secret
`;

export function BulkImportModal({ open, onClose, onImported }: BulkImportModalProps) {
  const { t } = useT();
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

  async function handleValidate() {
    if (inFlightRef.current) return;
    inFlightRef.current = true;
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
      const message = err instanceof Error ? err.message : 'Import failed';
      setError(message);
    } finally {
      inFlightRef.current = false;
      setLoading(false);
    }
  }

  function handlePasteExample() {
    setContent(PLACEHOLDER);
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
          placeholder={PLACEHOLDER}
          value={content}
          onChange={(e) => setContent(e.target.value)}
        />

        {(report || error) && (
          <p role="status" className="flex flex-wrap gap-x-3 text-sm">
            {report && (
              <span className={report.valid ? 'text-ds-on-tint-success' : 'text-ds-on-tint-error'}>
                {report.summary}
              </span>
            )}
            {error && <span className="text-ds-on-tint-error">{error}</span>}
          </p>
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
