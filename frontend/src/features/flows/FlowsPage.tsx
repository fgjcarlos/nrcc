import { useQuery } from '@tanstack/react-query'
import { Link, useParams } from 'react-router-dom'

import { api, type FlowList, type OperationStatus } from '../../api'
import { EmptyState, InlineNotice, LoadingState, StatCard } from '../../common/components'
import { formatErrorMessage } from '../../common/utils/format'

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

  const detailQuery = useQuery({
    queryKey: ['flow-detail', selectedFlowId],
    queryFn: () => api.flow(selectedFlowId),
    enabled: selectedFlowId.length > 0,
  })

  const selectedFlow = !flows || !selectedFlowId ? undefined : flows.items.find((item) => item.id === selectedFlowId)

  return (
    <div className="space-y-6 sm:space-y-8">
      <header className="flex flex-col gap-6 lg:flex-row lg:items-end lg:justify-between">
        <div>
          <p className="text-xs uppercase tracking-[0.28em] text-base-content/50">Runtime</p>
          <h2 className="page-title text-3xl mt-1">Flows</h2>
          <p className="mt-2 max-w-3xl text-sm text-base-content/65">
            Inspect the configured Node-RED flow file from the control center. This module is read-only in this phase.
          </p>
        </div>
        <div className="flex flex-wrap gap-2 text-sm text-base-content/70">
          <span className="rounded-full bg-base-300/60 px-3 py-1">Read-only</span>
          <span className="rounded-full bg-base-300/60 px-3 py-1">Busy: {operationStatus?.busy ? 'Yes' : 'No'}</span>
        </div>
      </header>

      <InlineNotice
        tone="info"
        title="Inspection only"
        detail="Flows can be reviewed here, but editing, import/export, and bulk actions are intentionally out of scope for this phase."
      />

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
            <div className="text-sm text-base-content/60">Updated: {formatTimestamp(flows?.source.updatedAt)}</div>
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
                    return (
                      <tr key={item.id} className={isSelected ? 'active' : undefined}>
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
                description="Choose a flow from the table to review node counts, wires, and the current read-only node inventory."
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
      </section>
    </div>
  )
}
