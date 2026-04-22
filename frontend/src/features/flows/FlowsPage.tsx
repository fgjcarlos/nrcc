import { useEffect, useRef, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link, useParams } from 'react-router-dom'

import { APIRequestError, api, type FlowAnalysis, type FlowList, type OperationStatus } from '../../api'
import { EmptyState, InlineNotice, LoadingState, StatCard } from '../../common/components'
import { formatErrorMessage } from '../../common/utils/format'
import { type ImportResponse } from '../../common/types'
import { useToasts } from '../shell/useToasts'

function formatCount(value: number) {
  return new Intl.NumberFormat().format(value)
}

function formatTimestamp(value?: string) {
  if (!value) {
    return 'Unavailable'
  }

  const parsed = new Date(value)
  if (Number.isNaN(parsed.getTime())) {
    return value
  }

  return parsed.toLocaleString()
}

function getAnalysisAction(error: unknown) {
  if (!(error instanceof APIRequestError) || !error.details || typeof error.details !== 'object') {
    return ''
  }

  const details = error.details as Record<string, unknown>
  return typeof details.action === 'string' ? details.action : ''
}

function AnalysisList({ title, items }: { title: string; items: string[] }) {
  return (
    <section>
      <h4 className="text-sm font-semibold uppercase tracking-[0.18em] text-base-content/55">{title}</h4>
      {items.length ? (
        <ul className="mt-3 space-y-2 text-sm text-base-content/80">
          {items.map((item) => (
            <li key={item} className="rounded-2xl border border-base-300/60 bg-base-100/70 px-3 py-2">
              {item}
            </li>
          ))}
        </ul>
      ) : (
        <p className="mt-3 text-sm text-base-content/60">No items reported.</p>
      )}
    </section>
  )
}

export function FlowsPage({
  flows,
  loading,
  error,
  operationStatus,
}: {
  flows?: FlowList
  loading: boolean
  error: unknown
  operationStatus?: OperationStatus
}) {
  const { flowId } = useParams()
  const selectedFlowId = flowId ?? ''
  const { pushToast } = useToasts()
  const queryClient = useQueryClient()
  const fileInputRef = useRef<HTMLInputElement>(null)

  // Multi-select state
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set())

  const detailQuery = useQuery({
    queryKey: ['flow-detail', selectedFlowId],
    queryFn: () => api.flow(selectedFlowId),
    enabled: selectedFlowId.length > 0,
  })

  const analysisMutation = useMutation({
    mutationFn: () => api.analyzeFlow(selectedFlowId),
  })

  // Export mutation
  const exportMutation = useMutation({
    mutationFn: () => api.exportFlows([...selectedIds]),
    onSuccess: (blob) => {
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = 'flows.json'
      document.body.appendChild(a)
      a.click()
      document.body.removeChild(a)
      URL.revokeObjectURL(url)
      pushToast({
        title: 'Export successful',
        detail: 'Flows exported successfully',
        tone: 'success',
      })
    },
    onError: (error) => {
      pushToast({
        title: 'Export failed',
        detail: formatErrorMessage(error, 'Failed to export flows'),
        tone: 'error',
      })
    },
  })

  // Import mutation
  const importMutation = useMutation({
    mutationFn: (file: File) => api.importFlows(file),
    onSuccess: (response: ImportResponse) => {
      // Invalidate flows query to refresh the table
      queryClient.invalidateQueries({ queryKey: ['flows'] })
      // Clear selection
      setSelectedIds(new Set())
      // Show success toast
      const detail = `${response.importedCount} flow(s) imported. ${response.restartAdvisory ? 'Node-RED may need to be restarted.' : ''}`
      pushToast({
        title: 'Import successful',
        detail,
        tone: 'success',
      })
    },
    onError: (error) => {
      pushToast({
        title: 'Import failed',
        detail: formatErrorMessage(error, 'Failed to import flows'),
        tone: 'error',
      })
    },
  })

  useEffect(() => {
    analysisMutation.reset()
  }, [selectedFlowId])

  const selectedFlow = !flows || !selectedFlowId ? undefined : flows.items.find((item) => item.id === selectedFlowId)
  const analysis = analysisMutation.data?.flow.id === selectedFlowId ? analysisMutation.data : undefined
  const analysisAction = getAnalysisAction(analysisMutation.error)

  const isLoadingExportOrImport = exportMutation.isPending || importMutation.isPending

  // Helper functions for multi-select
  const toggleSelect = (id: string) => {
    const newSet = new Set(selectedIds)
    if (newSet.has(id)) {
      newSet.delete(id)
    } else {
      newSet.add(id)
    }
    setSelectedIds(newSet)
  }

  const selectAll = () => {
    if (flows) {
      setSelectedIds(new Set(flows.items.map((f) => f.id)))
    }
  }

  const clearAll = () => {
    setSelectedIds(new Set())
  }

  const allSelected = flows && flows.items.length > 0 && selectedIds.size === flows.items.length
  const partialSelected = selectedIds.size > 0 && !allSelected

  const handleImportFile = (file: File) => {
    if (file.type !== 'application/json' && !file.name.endsWith('.json')) {
      pushToast({
        title: 'Invalid file',
        detail: 'Please select a valid JSON file',
        tone: 'error',
      })
      return
    }
    importMutation.mutate(file)
    if (fileInputRef.current) {
      fileInputRef.current.value = ''
    }
  }

  return (
    <div className="space-y-6 sm:space-y-8">
      <header className="flex flex-col gap-6 lg:flex-row lg:items-end lg:justify-between">
        <div>
          <p className="text-xs uppercase tracking-[0.28em] text-base-content/50">Runtime</p>
          <h2 className="page-title text-3xl mt-1">Flows</h2>
          <p className="mt-2 max-w-3xl text-sm text-base-content/65">
            Inspect and manage the configured Node-RED flow file from the control center.
          </p>
        </div>
        <div className="flex flex-wrap gap-2 text-sm text-base-content/70">
          <span className="rounded-full bg-base-300/60 px-3 py-1">Busy: {operationStatus?.busy ? 'Yes' : 'No'}</span>
        </div>
      </header>

      {operationStatus?.busy ? (
        <InlineNotice
          tone="warn"
          title="System busy"
          detail={
            (operationStatus.type ? `${operationStatus.type} in progress` : 'Another operation is in progress') +
            (operationStatus.detail ? `: ${operationStatus.detail}` : '.')
          }
        />
      ) : null}

      {error ? (
        <InlineNotice
          tone="error"
          title="Flows unavailable"
          detail={formatErrorMessage(error, 'The configured flow file could not be inspected.')}
        />
      ) : null}

      <section className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <StatCard
          label="Flows"
          value={loading ? 'Loading' : formatCount(flows?.summary.flowCount ?? 0)}
          detail="Tabs discovered in the configured flow file."
          accent="info"
        />
        <StatCard
          label="Nodes"
          value={loading ? 'Loading' : formatCount(flows?.summary.nodeCount ?? 0)}
          detail="Operational nodes assigned to tabs."
          accent="ok"
        />
        <StatCard
          label="Disabled nodes"
          value={loading ? 'Loading' : formatCount(flows?.summary.disabledNodeCount ?? 0)}
          detail="Nodes marked disabled in the flow definition."
          accent={(flows?.summary.disabledNodeCount ?? 0) > 0 ? 'warn' : 'neutral'}
        />
        <StatCard
          label="Custom nodes"
          value={loading ? 'Loading' : formatCount(flows?.summary.customNodeCount ?? 0)}
          detail="Node instances outside the built-in/core heuristic set."
          accent={(flows?.summary.customNodeCount ?? 0) > 0 ? 'info' : 'neutral'}
        />
      </section>

      <section className="grid grid-cols-1 gap-6 xl:grid-cols-[minmax(0,1.6fr)_minmax(20rem,1fr)]">
        <article className="surface-card border border-base-300/60 p-6 md:p-7">
          <div className="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
            <div>
              <h3 className="section-title">Configured flows</h3>
              <p className="mt-1 text-sm text-base-content/60">
                Source: <span className="font-medium text-base-content">{flows?.source.path ?? 'Loading...'}</span>
              </p>
            </div>
            <div className="flex flex-wrap gap-2">
              <button
                className="action-btn-primary"
                type="button"
                disabled={selectedIds.size === 0 || isLoadingExportOrImport || operationStatus?.busy}
                onClick={() => exportMutation.mutate()}
              >
                {exportMutation.isPending ? 'Exporting…' : 'Export Selected'}
              </button>
              <button
                className="action-btn-secondary"
                type="button"
                disabled={isLoadingExportOrImport || operationStatus?.busy}
                onClick={() => fileInputRef.current?.click()}
              >
                {importMutation.isPending ? 'Importing…' : 'Import'}
              </button>
              <input
                ref={fileInputRef}
                type="file"
                accept=".json"
                style={{ display: 'none' }}
                onChange={(e) => {
                  const file = e.currentTarget.files?.[0]
                  if (file) {
                    handleImportFile(file)
                  }
                }}
              />
            </div>
          </div>

          <div className="mt-1 text-sm text-base-content/60">
            Updated: {formatTimestamp(flows?.source.updatedAt)}
          </div>

          {loading ? <div className="mt-6"><LoadingState message="Loading flow inventory..." /></div> : null}

          {!loading && !flows?.items.length ? (
            <div className="mt-6">
              <EmptyState
                title="No flows found"
                description="No tabs were found in the configured flow file. Check that the file exists and contains Node-RED flow definitions."
              />
            </div>
          ) : null}

          {flows?.items.length ? (
            <div className="mt-6 overflow-x-auto">
              <table className="table table-zebra">
                <thead>
                  <tr>
                    <th>
                      <input
                        type="checkbox"
                        className="checkbox"
                        checked={allSelected}
                        ref={(el) => {
                          if (el) {
                            el.indeterminate = partialSelected
                          }
                        }}
                        onChange={() => {
                          if (allSelected) {
                            clearAll()
                          } else {
                            selectAll()
                          }
                        }}
                      />
                    </th>
                    <th>Flow</th>
                    <th>Nodes</th>
                    <th>Disabled</th>
                    <th>Wires</th>
                    <th>Subflows</th>
                    <th></th>
                  </tr>
                </thead>
                <tbody>
                  {flows.items.map((item) => {
                    const isSelected = item.id === selectedFlowId
                    const isChecked = selectedIds.has(item.id)
                    return (
                      <tr key={item.id} className={isSelected ? 'active' : undefined}>
                        <td>
                          <input
                            type="checkbox"
                            className="checkbox"
                            checked={isChecked}
                            onChange={() => toggleSelect(item.id)}
                          />
                        </td>
                        <td>
                          <div className="font-medium text-base-content">{item.label}</div>
                          <div className="text-xs text-base-content/55">{item.id}</div>
                        </td>
                        <td>{formatCount(item.nodeCount)}</td>
                        <td>{formatCount(item.disabledNodeCount)}</td>
                        <td>
                          {formatCount(item.inboundWireCount)} / {formatCount(item.outboundWireCount)}
                        </td>
                        <td>{formatCount(item.subflowUsageCount)}</td>
                        <td>
                          <Link className="btn btn-sm btn-ghost" to={`/app/flows/${encodeURIComponent(item.id)}`}>
                            Inspect
                          </Link>
                        </td>
                      </tr>
                    )
                  })}
                </tbody>
              </table>
            </div>
          ) : null}
        </article>

        <div className="space-y-6">
          <article className="surface-card border border-base-300/60 p-6 md:p-7">
            <div className="flex items-start justify-between gap-3">
              <div>
                <h3 className="section-title">Flow detail</h3>
                <p className="mt-1 text-sm text-base-content/60">
                  {selectedFlow ? `Inspecting ${selectedFlow.label}` : 'Select a flow from the list to inspect its nodes and type mix.'}
                </p>
              </div>
              {selectedFlowId ? (
                <Link className="btn btn-sm btn-ghost" to="/app/flows">
                  Clear
                </Link>
              ) : null}
            </div>

            {selectedFlowId && detailQuery.isLoading ? <div className="mt-6"><LoadingState message="Loading flow detail..." /></div> : null}

            {detailQuery.error ? (
              <div className="mt-6">
                <InlineNotice
                  tone="error"
                  title="Flow detail unavailable"
                  detail={formatErrorMessage(detailQuery.error, 'The selected flow could not be loaded.')}
                />
              </div>
            ) : null}

            {!selectedFlowId ? (
              <div className="mt-6">
                <EmptyState
                  title="No flow selected"
                  description="Choose a flow from the table to review node counts, wires, and the current node inventory."
                />
              </div>
            ) : null}

            {detailQuery.data ? (
              <div className="mt-6 space-y-6">
                <section className="grid grid-cols-2 gap-3">
                  <StatCard label="Nodes" value={formatCount(detailQuery.data.flow.nodeCount)} accent="ok" />
                  <StatCard label="Disabled" value={formatCount(detailQuery.data.flow.disabledNodeCount)} accent="warn" />
                  <StatCard label="Inbound wires" value={formatCount(detailQuery.data.flow.inboundWireCount)} accent="info" />
                  <StatCard label="Outbound wires" value={formatCount(detailQuery.data.flow.outboundWireCount)} accent="info" />
                </section>

                <section>
                  <h4 className="text-sm font-semibold uppercase tracking-[0.18em] text-base-content/55">Node type mix</h4>
                  {detailQuery.data.flow.nodeTypes.length ? (
                    <div className="mt-3 space-y-2">
                      {detailQuery.data.flow.nodeTypes.map((item) => (
                        <div key={item.type} className="flex items-center justify-between rounded-xl bg-base-200/60 px-3 py-2 text-sm">
                          <div>
                            <span className="font-medium text-base-content">{item.type}</span>
                            {item.custom ? <span className="ml-2 text-xs text-info">custom</span> : null}
                          </div>
                          <span className="text-base-content/70">{formatCount(item.count)}</span>
                        </div>
                      ))}
                    </div>
                  ) : (
                    <p className="mt-3 text-sm text-base-content/60">No node types available for this flow.</p>
                  )}
                </section>

                <section>
                  <h4 className="text-sm font-semibold uppercase tracking-[0.18em] text-base-content/55">Nodes</h4>
                  {detailQuery.data.flow.nodes.length ? (
                    <div className="mt-3 space-y-2">
                      {detailQuery.data.flow.nodes.map((node) => (
                        <div key={node.id} className="rounded-2xl border border-base-300/60 bg-base-100/70 p-3">
                          <div className="flex flex-wrap items-center justify-between gap-2">
                            <div>
                              <div className="font-medium text-base-content">{node.name}</div>
                              <div className="text-xs text-base-content/55">{node.type} • {node.id}</div>
                            </div>
                            <div className="flex flex-wrap gap-2 text-xs text-base-content/70">
                              <span className="rounded-full bg-base-200 px-2 py-1">Wires: {formatCount(node.wireCount)}</span>
                              {node.disabled ? <span className="rounded-full bg-warning/15 px-2 py-1 text-warning">Disabled</span> : null}
                            </div>
                          </div>
                        </div>
                      ))}
                    </div>
                  ) : (
                    <p className="mt-3 text-sm text-base-content/60">This flow has no operational nodes.</p>
                  )}
                </section>
              </div>
            ) : null}
          </article>

          <article className="surface-card border border-base-300/60 p-6 md:p-7">
            <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
              <div>
                <h3 className="section-title">Advisory AI analysis</h3>
                <p className="mt-1 text-sm text-base-content/60">
                  {selectedFlow ? `Run an explicit analysis for ${selectedFlow.label}. Results are advisory only and never modify the flow file.` : 'Select a single flow to request an advisory AI review.'}
                </p>
              </div>
              <button
                className="action-btn-primary"
                type="button"
                disabled={!selectedFlowId || detailQuery.isLoading || analysisMutation.isPending || operationStatus?.busy}
                onClick={() => analysisMutation.mutate()}
              >
                {analysisMutation.isPending ? 'Analyzing…' : 'Analyze selected flow'}
              </button>
            </div>

            <div className="mt-6 space-y-4">
              <InlineNotice
                tone="info"
                title="Advisory output"
                detail="This review is AI-generated guidance for operators. It summarizes likely strengths, issues, and next checks from the current flow structure only."
              />

              {!selectedFlowId ? (
                <EmptyState
                  title="No flow selected"
                  description="Choose one flow first. This MVP intentionally analyzes one selected flow at a time."
                />
              ) : null}

              {analysisMutation.error ? (
                <InlineNotice
                  tone="error"
                  title="Analysis unavailable"
                  detail={[formatErrorMessage(analysisMutation.error, 'The AI analysis could not be completed.'), analysisAction].filter(Boolean).join(' ')}
                />
              ) : null}

              {analysisMutation.isPending ? <LoadingState message="Requesting advisory AI analysis..." /> : null}

              {analysis ? (
                <div className="space-y-6">
                  <div className="rounded-2xl border border-base-300/60 bg-base-100/70 p-4">
                    <div className="flex flex-wrap items-center gap-2 text-xs uppercase tracking-[0.16em] text-base-content/55">
                      <span>{analysis.provider.name}</span>
                      <span>•</span>
                      <span>{analysis.provider.model}</span>
                      <span>•</span>
                      <span>{analysis.provider.local ? 'local-first' : 'remote'}</span>
                    </div>
                    <p className="mt-3 text-sm leading-6 text-base-content/85">{analysis.summary}</p>
                  </div>

                  <AnalysisList title="Strengths" items={analysis.strengths} />
                  <AnalysisList title="Issues" items={analysis.issues} />
                  <AnalysisList title="Suggestions" items={analysis.suggestions} />
                </div>
              ) : null}
            </div>
          </article>
        </div>
      </section>
    </div>
  )
}
