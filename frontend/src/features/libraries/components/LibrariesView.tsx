import { useState, useEffect } from 'react';
import { useLibrariesData } from '@/features/libraries/hooks/useLibrariesData';
import { useLibrariesActions } from '@/features/libraries/hooks/useLibrariesActions';
import { ChevronLeft, ChevronRight, ExternalLink, Trash2 } from 'lucide-react';

export function LibrariesView() {
  const [searchQuery, setSearchQuery] = useState('');
  const [confirmingUninstall, setConfirmingUninstall] = useState<string | null>(null);
  const [installedPage, setInstalledPage] = useState(1);
  const installedPageSize = 5;

  // Data hooks
  const { libraries, isLoading, isError } = useLibrariesData();

  // Actions hooks
  const {
    installMutation,
    uninstallMutation,
    searchResults,
    searching,
    handleSearch,
    handleInstall: handleInstallAction,
    handleUninstall,
  } = useLibrariesActions();

  const handleInstall = (name: string, alias?: string) => {
    handleInstallAction(name, alias);
  };

  // Clear searchQuery when installation succeeds
  useEffect(() => {
    if (installMutation.isSuccess && searchResults.length === 0) {
      setSearchQuery('');
    }
  }, [installMutation.isSuccess, searchResults.length]);

  const totalInstalledPages = Math.max(1, Math.ceil(libraries.length / installedPageSize));
  const visibleLibraries = libraries.slice(
    (installedPage - 1) * installedPageSize,
    installedPage * installedPageSize
  );

  useEffect(() => {
    if (installedPage > totalInstalledPages) {
      setInstalledPage(totalInstalledPages);
    }
  }, [installedPage, totalInstalledPages]);

  const handleConfirmUninstall = (name: string) => {
    if (confirmingUninstall === name) {
      handleUninstall(name);
      setConfirmingUninstall(null);
      return;
    }

    setConfirmingUninstall(name);
  };

  // Truncate long strings
  const truncate = (text: string | undefined, maxLength: number = 60): string => {
    if (!text) return '';
    return text.length > maxLength ? text.substring(0, maxLength) + '...' : text;
  };

  // Get keywords badges (first 2-3)
  const getKeywordBadges = (keywords: string[] | undefined) => {
    if (!keywords || keywords.length === 0) return null;
    const maxBadges = 2;
    return keywords.slice(0, maxBadges);
  };

  return (
    <div className="space-y-8 p-4 sm:p-6">
      <div className="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
        <div>
          <p className="text-xs uppercase tracking-[0.28em] text-base-content/50">Extensions</p>
          <h1 className="text-3xl font-bold tracking-tight text-base-content">npm Libraries</h1>
          <p className="mt-2 max-w-2xl text-sm text-base-content/65">
            Search, install and remove Node-RED libraries from a single operational console.
          </p>
        </div>

        <div className="flex flex-wrap gap-2 text-sm text-base-content/70">
          <span className="rounded-full border border-border bg-base-300/60 px-3 py-1">Installed: {libraries.length}</span>
          <span className="rounded-full border border-border bg-base-300/60 px-3 py-1">
            Search: {searchResults.length} results
          </span>
        </div>
      </div>

      <div className="grid grid-cols-1 gap-6 xl:grid-cols-[minmax(0,0.95fr)_minmax(0,1.15fr)]">
        {/* Search Column */}
        <div className="space-y-4">
          <div className="surface-card overflow-hidden p-0">
            <div className="border-b border-border p-5">
              <h2 className="text-lg font-semibold text-base-content">Buscar e instalar</h2>
              <p className="mt-1 text-sm text-base-content/65">
                Busca en npm y añade paquetes al runtime de Node-RED.
              </p>
            </div>
            <div className="p-5">
            <div className="mt-4">
              <input
                type="text"
                value={searchQuery}
                onChange={(e) => {
                  setSearchQuery(e.target.value);
                  handleSearch(e.target.value);
                }}
                placeholder="Buscar paquetes npm..."
                className="glass-panel w-full rounded-xl border border-border px-3 py-2 text-base-content transition-colors focus:outline-none focus:ring-2 focus:ring-primary/50"
              />
            </div>

            {searching && (
              <div className="mt-4 text-sm text-base-content/60">Buscando...</div>
            )}

            <div className="mt-4 space-y-2">
              {searchResults.map((result) => (
                <div
                  key={result.name}
                  className="surface-panel flex flex-col gap-3 p-4 sm:flex-row sm:items-center sm:justify-between"
                >
                  <div className="min-w-0 flex-1">
                    <div className="truncate font-mono text-sm font-semibold text-base-content">{result.name}</div>
                    <div className="text-sm text-base-content/65">
                      v{result.version} • {truncate(result.description, 50)}
                    </div>
                    {typeof result.downloads === 'number' && (
                      <div className="text-xs text-base-content/55">
                        {result.downloads.toLocaleString()} descargas/semana
                      </div>
                    )}
                  </div>
                  <button
                    onClick={() => handleInstall(result.name)}
                    disabled={installMutation.isPending}
                    className="action-btn-primary justify-center shrink-0 text-sm"
                  >
                    Instalar
                  </button>
                </div>
              ))}

              {!searching && searchQuery.length >= 2 && searchResults.length === 0 && (
                <div className="glass-panel rounded-2xl border border-dashed border-border px-4 py-8 text-center text-sm text-base-content/60">
                  No results yet. Try a different package name.
                </div>
              )}
            </div>
            </div>
          </div>
        </div>

        {/* Installed Column */}
        <div className="space-y-4">
          <div className="surface-card overflow-hidden p-0">
            <div className="border-b border-border p-5">
              <h2 className="text-lg font-semibold text-base-content">Librerías instaladas</h2>
              <p className="mt-1 text-sm text-base-content/65">
                Paquetes activos detectados en el runtime de Node-RED.
              </p>
            </div>

            <div className="p-5">
            {isError ? (
              <div className="rounded-2xl border border-error/40 bg-error/10 px-4 py-8 text-center">
                <p className="font-medium text-error">No se pudieron cargar las librerías.</p>
                <p className="mt-1 text-sm text-base-content/60">Revisá que el backend esté levantado y autenticado.</p>
              </div>
            ) : isLoading ? (
              <div className="animate-pulse space-y-2">
                <div className="h-14 rounded-2xl skeleton-dark"></div>
                <div className="h-14 rounded-2xl skeleton-dark"></div>
              </div>
            ) : libraries.length === 0 ? (
              <div className="mt-4 rounded-2xl border border-dashed border-border px-4 py-8 text-center">
                <p className="font-medium text-base-content">No hay librerías instaladas</p>
                <p className="mt-1 text-sm text-base-content/60">Instalá una desde la columna de búsqueda.</p>
              </div>
            ) : (
              <div className="space-y-3">
                {visibleLibraries.map((lib) => (
                  <div
                    key={lib.name}
                    className="surface-panel rounded-xl p-4"
                  >
                    {/* Header row: name, version badge, uninstall button */}
                    <div className="flex items-start justify-between gap-3 mb-2">
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2 mb-1">
                          <code className="font-mono text-sm font-semibold text-base-content break-all">
                            {lib.name}
                          </code>
                          {lib.version && (
                            <span className="badge badge-sm badge-primary shrink-0 font-mono">
                              v{lib.version}
                            </span>
                          )}
                        </div>
                      </div>
                      <button
                        onClick={() => handleConfirmUninstall(lib.name)}
                        disabled={uninstallMutation.isPending}
                        title="Uninstall"
                        className={
                          confirmingUninstall === lib.name
                            ? 'btn btn-error btn-xs shrink-0 px-3'
                            : 'btn btn-ghost btn-xs shrink-0'
                        }
                      >
                        {confirmingUninstall === lib.name ? 'Confirmar' : <Trash2 size={16} />}
                      </button>
                    </div>

                    {/* Description */}
                    {lib.description && (
                      <p className="text-sm text-base-content/70 mb-2">
                        {truncate(lib.description, 100)}
                      </p>
                    )}

                    {/* Keywords badges */}
                    {lib.keywords && lib.keywords.length > 0 && (
                      <div className="flex flex-wrap gap-1 mb-3">
                        {getKeywordBadges(lib.keywords)?.map((keyword) => (
                          <span
                            key={keyword}
                            className="badge badge-outline badge-sm text-xs"
                          >
                            {keyword}
                          </span>
                        ))}
                      </div>
                    )}

                    {/* Links row */}
                    <div className="flex flex-wrap gap-2 border-t border-border/70 pt-3">
                      {lib.homepage && (
                        <a
                          href={lib.homepage}
                          target="_blank"
                          rel="noopener noreferrer"
                          className="btn btn-ghost btn-xs text-xs gap-1"
                          title="Visit homepage"
                        >
                          <ExternalLink size={14} />
                          Homepage
                        </a>
                      )}
                      <a
                        href={`https://www.npmjs.com/package/${encodeURIComponent(lib.name)}`}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="btn btn-ghost btn-xs text-xs gap-1"
                        title="View on npm"
                      >
                        <ExternalLink size={14} />
                        npm
                      </a>
                      <a
                        href={`https://flows.nodered.org/node/${encodeURIComponent(lib.name)}`}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="btn btn-ghost btn-xs text-xs gap-1"
                        title="View on Node-RED Flows"
                      >
                        <ExternalLink size={14} />
                        Flows
                      </a>
                    </div>
                  </div>
                ))}
                {libraries.length > installedPageSize && (
                  <div className="flex flex-col gap-3 border-t border-border/70 pt-4 sm:flex-row sm:items-center sm:justify-between">
                    <p className="text-xs font-medium text-base-content/55">
                      Mostrando {(installedPage - 1) * installedPageSize + 1}-
                      {Math.min(installedPage * installedPageSize, libraries.length)} de {libraries.length}
                    </p>
                    <div className="glass-panel inline-flex items-center gap-1 rounded-full border border-border p-1 self-start sm:self-auto">
                      <button
                        type="button"
                        className="inline-flex h-8 items-center gap-1 rounded-full px-3 text-xs font-semibold text-base-content/70 transition-colors hover:bg-primary/10 hover:text-primary disabled:pointer-events-none disabled:opacity-35"
                        disabled={installedPage === 1}
                        onClick={() => setInstalledPage((page) => Math.max(1, page - 1))}
                        aria-label="Página anterior"
                      >
                        <ChevronLeft size={14} />
                        Anterior
                      </button>
                      <span className="rounded-full bg-primary/15 px-3 py-1.5 font-mono text-xs font-bold text-primary">
                        {installedPage} / {totalInstalledPages}
                      </span>
                      <button
                        type="button"
                        className="inline-flex h-8 items-center gap-1 rounded-full px-3 text-xs font-semibold text-base-content/70 transition-colors hover:bg-primary/10 hover:text-primary disabled:pointer-events-none disabled:opacity-35"
                        disabled={installedPage === totalInstalledPages}
                        onClick={() => setInstalledPage((page) => Math.min(totalInstalledPages, page + 1))}
                        aria-label="Página siguiente"
                      >
                        Siguiente
                        <ChevronRight size={14} />
                      </button>
                    </div>
                  </div>
                )}
              </div>
            )}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
