import { ChangeEvent, useMemo, useRef } from 'react';
import { Download, File as FileIcon, RefreshCw, Trash2, Upload } from 'lucide-react';
import { StateContainer } from '@/shared/components/StateContainer';
import { ConfirmationDialog } from '@/shared/components/ConfirmationDialog';
import { Button } from '@/shared/components/ui/Button';
import { useConfirmationDialog } from '@/shared/hooks/useConfirmationDialog';
import { formatBytes } from '@/shared/lib';
import { filesService } from '../services';
import { useFiles } from '../hooks';
import type { ManagedFile } from '../types';

function formatModTime(modTime: number) {
  return new Intl.DateTimeFormat(undefined, {
    dateStyle: 'medium',
    timeStyle: 'short',
  }).format(new Date(modTime * 1000));
}

export function FilesView() {
  const fileInputRef = useRef<HTMLInputElement>(null);
  const deleteDialog = useConfirmationDialog<ManagedFile>();
  const { files, isLoading, isError, refetch, uploadMutation, deleteMutation } = useFiles();

  const sortedFiles = useMemo(
    () => [...files].sort((a, b) => b.modTime - a.modTime || a.name.localeCompare(b.name)),
    [files],
  );

  const handleFileChange = (event: ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (!file) return;

    uploadMutation.mutate(file, {
      onSettled: () => {
        event.target.value = '';
      },
    });
  };

  const handleConfirmDelete = () => {
    if (!deleteDialog.pendingItem) return;

    deleteMutation.mutate(deleteDialog.pendingItem.name, {
      onSuccess: () => deleteDialog.close(),
    });
  };

  const loadingSlot = (
    <div className="flex flex-col items-center justify-center gap-3 py-12" role="status">
      <RefreshCw className="h-8 w-8 animate-spin text-primary" aria-hidden="true" />
      <p className="text-base-content/60">Loading files...</p>
    </div>
  );

  const errorSlot = (
    <div className="rounded-2xl border border-error/30 bg-error/10 px-4 py-8 text-center">
      <p className="font-medium text-error">Could not load files.</p>
      <p className="mt-1 text-sm text-base-content/60">Check that the backend is running and try again.</p>
      <Button type="button" onClick={() => refetch()} variant="secondary" size="sm" className="mt-4">
        Retry
      </Button>
    </div>
  );

  const emptySlot = (
    <div className="rounded-2xl border border-dashed border-border px-4 py-10 text-center">
      <FileIcon className="mx-auto h-10 w-10 text-base-content/35" aria-hidden="true" />
      <p className="mt-3 font-medium text-base-content">No files uploaded yet</p>
      <p className="mt-1 text-sm text-base-content/60">Upload a file to make it available here.</p>
    </div>
  );

  return (
    <div className="space-y-6 p-4 sm:p-6">
      <div className="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
        <div>
          <p className="text-xs uppercase tracking-[0.28em] text-base-content/50">Storage</p>
          <h1 className="text-3xl font-bold tracking-tight text-base-content">Files</h1>
          <p className="mt-2 max-w-2xl text-sm text-base-content/65">
            Upload, download, and remove files stored by the Node-RED Control Center.
          </p>
        </div>

        <div className="flex flex-wrap items-center gap-2">
          <span className="rounded-full border border-border bg-base-300/60 px-3 py-1 text-sm text-base-content/70">
            Files: {files.length}
          </span>
          <input
            ref={fileInputRef}
            aria-label="Choose file to upload"
            type="file"
            className="hidden"
            onChange={handleFileChange}
          />
          <Button
            type="button"
            onClick={() => fileInputRef.current?.click()}
            disabled={uploadMutation.isPending}
            aria-label="Upload file"
            className="gap-2"
          >
            <Upload className="h-4 w-4" aria-hidden="true" />
            {uploadMutation.isPending ? 'Uploading...' : 'Upload file'}
          </Button>
        </div>
      </div>

      {uploadMutation.isError && (
        <div className="rounded-xl border border-error/30 bg-error/10 px-4 py-3 text-sm text-error" role="alert">
          Upload failed. Please select a valid file and try again.
        </div>
      )}

      <section className="surface-card overflow-hidden p-0" aria-labelledby="files-list-heading">
        <div className="border-b border-border p-5">
          <h2 id="files-list-heading" className="text-lg font-semibold text-base-content">Uploaded files</h2>
          <p className="mt-1 text-sm text-base-content/65">Files are listed immediately after a successful upload.</p>
        </div>

        <div className="p-5">
          <StateContainer
            isLoading={isLoading}
            isError={isError}
            isEmpty={sortedFiles.length === 0}
            loadingSlot={loadingSlot}
            errorSlot={errorSlot}
            emptySlot={emptySlot}
          >
            <div className="overflow-x-auto">
              <table className="w-full text-left text-sm">
                <thead className="border-b border-border text-xs uppercase tracking-wide text-base-content/55">
                  <tr>
                    <th className="px-3 py-3 font-semibold">Name</th>
                    <th className="px-3 py-3 font-semibold">Size</th>
                    <th className="px-3 py-3 font-semibold">Modified</th>
                    <th className="px-3 py-3 text-right font-semibold">Actions</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-border">
                  {sortedFiles.map((file) => (
                    <tr key={file.name}>
                      <td className="max-w-[18rem] px-3 py-4">
                        <div className="flex items-center gap-2">
                          <FileIcon className="h-4 w-4 flex-shrink-0 text-base-content/45" aria-hidden="true" />
                          <span className="truncate font-mono text-sm text-base-content" title={file.name}>{file.name}</span>
                        </div>
                      </td>
                      <td className="px-3 py-4 text-base-content/70">{formatBytes(file.size)}</td>
                      <td className="px-3 py-4 text-base-content/70">{formatModTime(file.modTime)}</td>
                      <td className="px-3 py-4">
                        <div className="flex justify-end gap-2">
                          <a
                            href={filesService.getDownloadUrl(file.name)}
                            className="action-btn-secondary inline-flex items-center gap-2 text-sm"
                            aria-label={`Download ${file.name}`}
                          >
                            <Download className="h-4 w-4" aria-hidden="true" />
                            Download
                          </a>
                          <Button
                            type="button"
                            variant="secondary"
                            size="sm"
                            onClick={() => deleteDialog.open(file)}
                            aria-label={`Delete ${file.name}`}
                            className="gap-2"
                          >
                            <Trash2 className="h-4 w-4" aria-hidden="true" />
                            Delete
                          </Button>
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </StateContainer>
        </div>
      </section>

      <ConfirmationDialog
        isOpen={deleteDialog.isOpen}
        title="Delete file"
        description={`Delete ${deleteDialog.pendingItem?.name ?? 'this file'}? This action cannot be undone.`}
        confirmText={deleteDialog.pendingItem?.name ?? ''}
        variant="danger"
        isPending={deleteMutation.isPending}
        onConfirm={handleConfirmDelete}
        onCancel={deleteDialog.close}
      />
    </div>
  );
}
