import { useState, useEffect } from 'react';
import { type InstalledLibrary } from '@/features/libraries/services';
import { ConfirmationDialog } from '@/shared/components';
import { useLibrariesData } from '@/features/libraries/hooks/useLibrariesData';
import { useLibrariesActions } from '@/features/libraries/hooks/useLibrariesActions';
import { ExternalLink, Trash2 } from 'lucide-react';

export function LibrariesView() {
  const [searchQuery, setSearchQuery] = useState('');
  const [confirmingUninstall, setConfirmingUninstall] = useState<string | null>(null);

  // Confirmation dialog state
  const [confirmConfig, setConfirmConfig] = useState<{
    isOpen: boolean;
    title: string;
    description: string;
    confirmText?: string;
    variant: 'danger' | 'warning' | 'default';
    onConfirm: () => void;
  } | null>(null);

  // Data hooks
  const { libraries, isLoading } = useLibrariesData();

  // Actions hooks
  const {
    installMutation,
    uninstallMutation,
    searchResults,
    searching,
    handleSearch,
    handleInstall: handleInstallAction,
    handleUninstall,
    handleClearSearch,
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

  const handleConfirmUninstall = (name: string) => {
    setConfirmConfig({
      isOpen: true,
      title: 'Desinstalar librería',
      description: `¿Está seguro que desea desinstalar ${name}? Node-RED se reiniciará.`,
      confirmText: name,
      variant: 'danger',
      onConfirm: () => {
        setConfirmConfig(null);
        handleUninstall(name);
      },
    });
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
    <div className="space-y-8 p-6">
      <div className="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
        <div>
          <p className="text-xs uppercase tracking-[0.28em] text-base-content/50">Extensions</p>
          <h1 className="text-3xl font-bold tracking-tight text-base-content">npm Libraries</h1>
          <p className="mt-2 max-w-2xl text-sm text-base-content/65">
            Search, install and remove Node-RED libraries from a single operational console.
          </p>
        </div>

        <div className="flex flex-wrap gap-2 text-sm text-base-content/70">
          <span className="rounded-full bg-base-300/60 px-3 py-1">Installed: {libraries.length}</span>
          <span className="rounded-full bg-base-300/60 px-3 py-1">
            Search: {searchResults.length} results
          </span>
        </div>
      </div>

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
        {/* Search Column */}
        <div className="space-y-4">
          <div className="surface-card p-6">
            <h2 className="text-lg font-semibold text-base-content">Buscar e instalar</h2>
            <p className="mt-1 text-sm text-base-content/65">
              Query npm packages and install them directly into the runtime.
            </p>
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
                  className="surface-panel flex items-center justify-between gap-4 p-4"
                >
                  <div className="flex-1">
                    <div className="font-medium text-base-content">{result.name}</div>
                    <div className="text-sm text-base-content/65">
                      v{result.version} • {truncate(result.description, 50)}
                    </div>
                    <div className="text-xs text-base-content/55">
                      {result.downloads.toLocaleString()} descargas/semana
                    </div>
                  </div>
                  <button
                    onClick={() => handleInstall(result.name)}
                    disabled={installMutation.isPending}
                    className="action-btn-primary shrink-0 text-sm"
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

        {/* Installed Column */}
        <div className="space-y-4">
          <div className="surface-card p-6">
            <h2 className="text-lg font-semibold text-base-content">Librerías instaladas</h2>
            <p className="mt-1 text-sm text-base-content/65">
              Active libraries currently installed in the Node-RED runtime.
            </p>

            {isLoading ? (
              <div className="animate-pulse space-y-2 pt-4">
                <div className="h-14 rounded-2xl skeleton-dark"></div>
                <div className="h-14 rounded-2xl skeleton-dark"></div>
              </div>
            ) : libraries.length === 0 ? (
              <div className="mt-4 rounded-2xl border border-dashed border-border px-4 py-8 text-center">
                <p className="font-medium text-base-content">No hay librerías instaladas</p>
                <p className="mt-1 text-sm text-base-content/60">Instalá una desde la columna de búsqueda.</p>
              </div>
            ) : (
              <div className="mt-4 space-y-3">
                {libraries.map((lib) => (
                  <div
                    key={lib.name}
                    className="surface-panel rounded-lg p-4"
                  >
                    {/* Header row: name, version badge, uninstall button */}
                    <div className="flex items-start justify-between gap-3 mb-2">
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2 mb-1">
                          <code className="font-mono text-sm font-semibold text-base-content break-all">
                            {lib.name}
                          </code>
                          {lib.version && (
                            <span className="badge badge-sm badge-primary shrink-0">
                              v{lib.version}
                            </span>
                          )}
                        </div>
                      </div>
                      <button
                        onClick={() => handleConfirmUninstall(lib.name)}
                        disabled={uninstallMutation.isPending}
                        title="Uninstall"
                        className="btn btn-ghost btn-xs shrink-0"
                      >
                        <Trash2 size={16} />
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
                    <div className="flex gap-2 flex-wrap">
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
              </div>
            )}
          </div>
        </div>
      </div>

      {/* Confirmation Dialog */}
      {confirmConfig && (
        <ConfirmationDialog
          isOpen={confirmConfig.isOpen}
          title={confirmConfig.title}
          description={confirmConfig.description}
          confirmText={confirmConfig.confirmText}
          variant={confirmConfig.variant}
          isPending={uninstallMutation.isPending}
          onConfirm={confirmConfig.onConfirm}
          onCancel={() => setConfirmConfig(null)}
        />
      )}
    </div>
  );
}
