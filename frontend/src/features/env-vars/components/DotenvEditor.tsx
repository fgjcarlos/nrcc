import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { envService } from '../services';
import { toast } from 'sonner';
import { Save, RefreshCw } from 'lucide-react';

import { queryKeys } from '@/shared/lib/queryKeys';
import { dotenvSchema } from '@/shared/validation/schemas';
import { useT } from '@/i18n';
import { StateContainer } from '@/shared/components';
import { ErrorState } from '@/shared/components/feedback';
import { Button, Textarea } from '@/shared/components/ui';

export function DotenvEditor() {
  const { t } = useT();
  const queryClient = useQueryClient();
  const [content, setContent] = useState('');
  const [isEditing, setIsEditing] = useState(false);

  // Parse and save failures stay next to the editor instead of in a toast.
  const [inlineError, setInlineError] = useState<string | null>(null);

  const { isLoading: isLoadingContent, isError: isLoadError, refetch } = useQuery({
    queryKey: queryKeys.dotenv.root,
    queryFn: async () => {
      const data = await envService.getDotenv();
      setContent(data.content);
      return data;
    },
  });

  const saveMutation = useMutation({
    mutationFn: () => envService.saveDotenv(content),
    onSuccess: (data) => {
      toast.success(data.message);
      setIsEditing(false);
      if (data.restarted) {
        toast.info(t('env-vars:savingToast'));
      }
    },
    onError: (error: unknown) => {
      let message = t('env-vars:dotenvSaveError');
      if (error && typeof error === 'object' && 'response' in error) {
        const err = error as { response?: { data?: { error?: { message?: string } } } };
        if (err.response?.data?.error?.message) {
          message = err.response.data.error.message;
        }
      }
      setInlineError(message);
    },
  });

  const handleSave = () => {
    const result = dotenvSchema.safeParse(content);
    if (!result.success) {
      // dotenvSchema puts the zero-based line index at path[1].
      const index = Number(result.error.issues[0]?.path[1]);
      setInlineError(t('env-vars:dotenvInvalidLine', { line: Number.isInteger(index) ? index + 1 : '?' }));
      return;
    }
    setInlineError(null);
    saveMutation.mutate();
  };

  const handleCancel = () => {
    // Reload from server
    queryClient.invalidateQueries({ queryKey: queryKeys.dotenv.root });
    setIsEditing(false);
    setInlineError(null);
  };

  return (
    <section aria-labelledby="dotenv-editor-title" className="space-y-3">
      <div className="space-y-1">
        <h2 id="dotenv-editor-title" className="text-sm font-semibold text-ds-text-primary">{t('env-vars:dotenvTitle')}</h2>
        <p className="max-w-[75ch] text-pretty text-sm text-ds-text-secondary">{t('env-vars:editorHelpText')}</p>
      </div>

      <StateContainer
        isLoading={isLoadingContent}
        isError={isLoadError}
        isEmpty={false}
        errorSlot={<ErrorState title={t('env-vars:dotenvLoadError')} onRetry={() => void refetch()} />}
      >
        <Textarea
          aria-labelledby="dotenv-editor-title"
          value={content}
          onChange={(e) => {
            setContent(e.target.value);
            setIsEditing(true);
            setInlineError(null);
          }}
          placeholder={t('env-vars:dotenvPlaceholder')}
          error={!!inlineError}
          aria-describedby={inlineError ? 'dotenv-editor-error' : undefined}
          className="h-64 font-mono"
        />

        {inlineError && (
          <p
            id="dotenv-editor-error"
            role="alert"
            className="mt-2 rounded-box border border-ds-danger/35 bg-ds-danger/10 px-3 py-2 text-sm text-ds-on-tint-error"
          >
            {inlineError}
          </p>
        )}

        <div className="mt-3 flex flex-wrap gap-2">
          <Button type="button" onClick={handleSave} disabled={!isEditing || saveMutation.isPending}>
            {saveMutation.isPending ? (
              <RefreshCw className="h-4 w-4 animate-spin motion-reduce:animate-none" aria-hidden="true" />
            ) : (
              <Save className="h-4 w-4" aria-hidden="true" />
            )}
            {saveMutation.isPending ? t('env-vars:dotenvSaving') : t('env-vars:dotenvSave')}
          </Button>
          {isEditing && (
            <Button type="button" variant="secondary" onClick={handleCancel} disabled={saveMutation.isPending}>
              {t('common:cancel')}
            </Button>
          )}
        </div>
      </StateContainer>
    </section>
  );
}

export default DotenvEditor;
