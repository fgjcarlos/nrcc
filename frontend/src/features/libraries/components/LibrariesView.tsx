import { useState, useEffect, Fragment, type ReactNode } from 'react';
import { ChevronLeft, ChevronRight, ExternalLink, Library, SearchX, Trash2 } from 'lucide-react';
import { ConfirmationDialog, StateContainer } from '@/shared/components';
import { EmptyState, ErrorState, Skeleton } from '@/shared/components/feedback';
import { Button, Input, Table } from '@/shared/components/ui';
import { useLibrariesData } from '@/features/libraries/hooks/useLibrariesData';
import { useLibrariesActions } from '@/features/libraries/hooks/useLibrariesActions';
import { useT } from '@/i18n';

const INSTALLED_PAGE_SIZE = 5;

/** Package name with break opportunities only after `/` and `-` and before a non-leading `@`. */
function PackageName({ name }: { name: string }) {
  const parts = name.split(/(?<=[/-])|(?=@)/).filter(Boolean);
  return (
    <code className="break-normal font-mono text-sm font-semibold text-ds-text-primary">
      {parts.map((part, index) => (
        <Fragment key={index}>
          {index > 0 && <wbr />}
          {part}
        </Fragment>
      ))}
    </code>
  );
}

// 24px link targets on desktop, 44px rows on phones (WCAG 2.5.8).
const linkClass =
  'inline-flex min-h-11 items-center gap-1 text-xs font-medium text-ds-on-tint-primary underline-offset-4 hover:underline sm:min-h-6';

function ExternalAnchor({ href, label, children }: { href: string; label: string; children: ReactNode }) {
  return (
    <a href={href} target="_blank" rel="noopener noreferrer" aria-label={label} className={linkClass}>
      <ExternalLink className="h-3 w-3" aria-hidden="true" />
      {children}
    </a>
  );
}

export function LibrariesView() {
  const [searchQuery, setSearchQuery] = useState('');
  const [pendingUninstall, setPendingUninstall] = useState<string | null>(null);
  const [installedPage, setInstalledPage] = useState(1);
  const { t } = useT();

  const { libraries, isLoading, isError, refetch } = useLibrariesData();
  const {
    installMutation,
    uninstallMutation,
    searchResults,
    searching,
    handleSearch,
    handleInstall,
    handleUninstall,
  } = useLibrariesActions();

  // Clear searchQuery when installation succeeds
  useEffect(() => {
    if (installMutation.isSuccess && searchResults.length === 0) {
      setSearchQuery('');
    }
  }, [installMutation.isSuccess, searchResults.length]);

  const totalInstalledPages = Math.max(1, Math.ceil(libraries.length / INSTALLED_PAGE_SIZE));
  const visibleLibraries = libraries.slice(
    (installedPage - 1) * INSTALLED_PAGE_SIZE,
    installedPage * INSTALLED_PAGE_SIZE,
  );

  useEffect(() => {
    if (installedPage > totalInstalledPages) {
      setInstalledPage(totalInstalledPages);
    }
  }, [installedPage, totalInstalledPages]);

  const npmUrl = (name: string) => `https://www.npmjs.com/package/${encodeURIComponent(name)}`;
  const installingName = installMutation.isPending ? installMutation.variables?.name : undefined;
  const showNoResults = !searching && searchQuery.length >= 2 && searchResults.length === 0;

  return (
    <div className="space-y-4">
      <header className="space-y-1 border-b border-ds-border-subtle pb-3">
        <h1 className="text-xl font-bold text-ds-text-primary">{t('libraries:title')}</h1>
        <p className="max-w-[75ch] text-pretty text-sm text-ds-text-secondary">{t('libraries:description')}</p>
      </header>

      {/* One sheet: sections separated by hairline rules, no nested cards. */}
      <div className="divide-y divide-ds-border-subtle">
        <section aria-labelledby="libraries-installed-title" className="space-y-3 py-4">
          <div className="flex flex-wrap items-baseline justify-between gap-2">
            <h2 id="libraries-installed-title" className="text-sm font-semibold text-ds-text-primary">
              {t('libraries:installedLibraries')}
            </h2>
            {!isLoading && !isError && (
              <p className="text-xs tabular-nums text-ds-text-secondary">
                {t('libraries:installedCount', { count: libraries.length })}
              </p>
            )}
          </div>

          <StateContainer
            isLoading={isLoading}
            isError={isError}
            isEmpty={libraries.length === 0}
            loadingSlot={<Skeleton rows={3} className="h-10" />}
            errorSlot={
              <ErrorState title={t('libraries:loadFailed')} description={t('libraries:loadHint')} onRetry={() => void refetch()} />
            }
            emptySlot={
              <EmptyState
                icon={<Library className="h-6 w-6" />}
                title={t('libraries:empty')}
                description={t('libraries:emptyHint')}
              />
            }
          >
            <div className="space-y-3">
              <Table stacked caption={t('libraries:installedLibraries')}>
                <thead>
                  <tr>
                    <th>{t('libraries:colPackage')}</th>
                    <th>{t('libraries:colVersion')}</th>
                    <th>{t('libraries:colLinks')}</th>
                    <th className="text-right">{t('libraries:colActions')}</th>
                  </tr>
                </thead>
                <tbody>
                  {visibleLibraries.map((lib) => (
                    <tr key={lib.name}>
                      <td data-label={t('libraries:colPackage')} className="max-sm:min-h-11">
                        <div className="min-w-0 space-y-0.5">
                          <PackageName name={lib.name} />
                          {lib.description && (
                            <p className="line-clamp-2 max-w-[60ch] text-xs text-ds-text-secondary">{lib.description}</p>
                          )}
                        </div>
                      </td>
                      <td data-label={t('libraries:colVersion')} className="max-sm:min-h-11">
                        {lib.version && (
                          <span className="font-mono text-xs font-semibold tabular-nums text-ds-text-secondary">
                            {lib.version}
                          </span>
                        )}
                      </td>
                      <td data-label={t('libraries:colLinks')}>
                        <div className="flex flex-wrap gap-x-3">
                          {lib.homepage && (
                            <ExternalAnchor href={lib.homepage} label={t('libraries:homepageOf', { name: lib.name })}>
                              {t('libraries:homepage')}
                            </ExternalAnchor>
                          )}
                          <ExternalAnchor href={npmUrl(lib.name)} label={t('libraries:viewOnNpm', { name: lib.name })}>
                            {t('libraries:npm')}
                          </ExternalAnchor>
                          <ExternalAnchor
                            href={`https://flows.nodered.org/node/${encodeURIComponent(lib.name)}`}
                            label={t('libraries:viewOnFlows', { name: lib.name })}
                          >
                            {t('libraries:flows')}
                          </ExternalAnchor>
                        </div>
                      </td>
                      <td data-label={t('libraries:colActions')} className="sm:text-right">
                        <Button
                          type="button"
                          variant="ghost"
                          size="sm"
                          onClick={() => setPendingUninstall(lib.name)}
                          disabled={uninstallMutation.isPending}
                          aria-label={t('libraries:uninstallPackage', { name: lib.name })}
                          className="min-h-11 sm:min-h-8"
                        >
                          <Trash2 className="h-4 w-4" aria-hidden="true" />
                          {t('libraries:uninstall')}
                        </Button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </Table>

              {libraries.length > INSTALLED_PAGE_SIZE && (
                <div className="flex flex-wrap items-center justify-between gap-3">
                  <p className="text-xs tabular-nums text-ds-text-secondary">
                    {t('libraries:showing', {
                      from: (installedPage - 1) * INSTALLED_PAGE_SIZE + 1,
                      to: Math.min(installedPage * INSTALLED_PAGE_SIZE, libraries.length),
                      total: libraries.length,
                    })}
                  </p>
                  <div className="flex gap-2">
                    <Button
                      type="button"
                      variant="secondary"
                      size="sm"
                      className="min-h-11 sm:min-h-8"
                      disabled={installedPage === 1}
                      onClick={() => setInstalledPage((page) => Math.max(1, page - 1))}
                      aria-label={t('libraries:previousPage')}
                    >
                      <ChevronLeft className="h-4 w-4" aria-hidden="true" />
                      {t('libraries:previous')}
                    </Button>
                    <Button
                      type="button"
                      variant="secondary"
                      size="sm"
                      className="min-h-11 sm:min-h-8"
                      disabled={installedPage === totalInstalledPages}
                      onClick={() => setInstalledPage((page) => Math.min(totalInstalledPages, page + 1))}
                      aria-label={t('libraries:nextPage')}
                    >
                      {t('libraries:next')}
                      <ChevronRight className="h-4 w-4" aria-hidden="true" />
                    </Button>
                  </div>
                </div>
              )}
            </div>
          </StateContainer>
        </section>

        <section aria-labelledby="libraries-search-title" className="space-y-3 py-4">
          <div className="space-y-1">
            <h2 id="libraries-search-title" className="text-sm font-semibold text-ds-text-primary">
              {t('libraries:searchTitle')}
            </h2>
            <p className="text-xs text-ds-text-secondary">{t('libraries:searchHint')}</p>
          </div>

          <div className="max-w-xl">
            <Input
              type="search"
              label={t('libraries:searchLabel')}
              value={searchQuery}
              onChange={(e) => {
                setSearchQuery(e.target.value);
                handleSearch(e.target.value);
              }}
              placeholder={t('libraries:searchPlaceholder')}
              autoComplete="off"
              spellCheck={false}
            />
          </div>

          <p role="status" className="text-xs tabular-nums text-ds-text-secondary empty:hidden">
            {searching
              ? t('libraries:searching')
              : searchResults.length > 0
                ? t('libraries:resultCount', { count: searchResults.length })
                : ''}
          </p>

          {searchResults.length > 0 && (
            <ul className="divide-y divide-ds-border-subtle border-y border-ds-border-subtle">
              {searchResults.map((result) => (
                <li key={result.name} className="flex flex-col gap-2 py-3 sm:flex-row sm:items-start sm:justify-between sm:gap-6">
                  <div className="min-w-0 flex-1 space-y-1">
                    <div className="flex flex-wrap items-baseline gap-x-3">
                      <PackageName name={result.name} />
                      {result.version && (
                        <span className="font-mono text-xs font-semibold tabular-nums text-ds-text-secondary">
                          {result.version}
                        </span>
                      )}
                    </div>
                    {result.description && (
                      <p className="line-clamp-2 max-w-[75ch] text-xs text-ds-text-secondary">{result.description}</p>
                    )}
                    <div className="flex flex-wrap items-center gap-x-3 text-xs text-ds-text-secondary">
                      {typeof result.downloads === 'number' && (
                        <span className="tabular-nums">
                          {t('libraries:downloadsPerWeek', { count: result.downloads.toLocaleString() })}
                        </span>
                      )}
                      <ExternalAnchor
                        href={result.npm || result.homepage || npmUrl(result.name)}
                        label={t('libraries:viewOnNpm', { name: result.name })}
                      >
                        {t('libraries:npm')}
                      </ExternalAnchor>
                    </div>
                  </div>
                  <Button
                    type="button"
                    onClick={() => handleInstall(result.name)}
                    disabled={installMutation.isPending}
                    aria-label={t('libraries:installPackage', { name: result.name })}
                    className="min-h-11 shrink-0 self-start sm:min-h-9"
                  >
                    {installingName === result.name ? t('libraries:installing') : t('libraries:install')}
                  </Button>
                </li>
              ))}
            </ul>
          )}

          {showNoResults && (
            <EmptyState
              icon={<SearchX className="h-6 w-6" />}
              title={t('libraries:noResults', { query: searchQuery })}
              description={t('libraries:noResultsHint')}
            />
          )}
        </section>
      </div>

      <ConfirmationDialog
        isOpen={pendingUninstall !== null}
        title={t('libraries:uninstallTitle', { name: pendingUninstall ?? '' })}
        description={t('libraries:uninstallDescription', { name: pendingUninstall ?? '' })}
        variant="danger"
        isPending={uninstallMutation.isPending}
        onConfirm={() => {
          if (pendingUninstall) handleUninstall(pendingUninstall);
          setPendingUninstall(null);
        }}
        onCancel={() => setPendingUninstall(null)}
      />
    </div>
  );
}
